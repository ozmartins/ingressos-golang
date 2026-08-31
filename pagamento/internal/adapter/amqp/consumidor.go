package amqp

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/platform/observability"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

// Consumidor consome reserva.criada e aplica o caso de uso. O prefetch é o teto
// de cobranças simultâneas: com a confirmação acontecendo só ao fim do trabalho,
// o broker nunca entrega mais do que o teto ao mesmo tempo (FR-019, D6).
type Consumidor struct {
	Canal      *amqp.Channel
	Fila       string
	Prefetch   int
	Caso       usecase.ProcessarPagamento
	Log        *slog.Logger
	Propagador propagation.TextMapPropagator

	// EmAndamento é o medidor de cobranças simultâneas, exposto para o teste de
	// vazão poder afirmar que o teto foi respeitado (SC-004).
	EmAndamento *Medidor
}

// Medidor acompanha o pico de trabalho concorrente.
type Medidor struct {
	mu     sync.Mutex
	atual  int
	maximo int
}

func (m *Medidor) entrar() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.atual++
	if m.atual > m.maximo {
		m.maximo = m.atual
	}
}
func (m *Medidor) sair() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.atual--
}

// Maximo devolve o maior número de processamentos simultâneos observado.
func (m *Medidor) Maximo() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maximo
}

// Consumir bloqueia até o contexto ser cancelado ou o canal fechar.
func (c *Consumidor) Consumir(ctx context.Context) error {
	if err := c.Canal.Qos(c.Prefetch, 0, false); err != nil {
		return err
	}
	entregas, err := c.Canal.ConsumeWithContext(ctx, c.Fila, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for i := 0; i < c.Prefetch; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range entregas {
				c.tratar(ctx, d)
			}
		}()
	}
	wg.Wait()
	return ctx.Err()
}

func (c *Consumidor) tratar(ctx context.Context, d amqp.Delivery) {
	if c.EmAndamento != nil {
		c.EmAndamento.entrar()
		defer c.EmAndamento.sair()
	}

	// Abre o processamento como continuação do rastro do bloqueio.
	if c.Propagador != nil {
		ctx = c.Propagador.Extract(ctx, portadorAMQP(d.Headers))
	}

	var i usecase.Intencao
	if err := json.Unmarshal(d.Body, &i); err != nil {
		// JSON quebrado é falha definitiva: nunca vai melhorar na reentrega.
		c.Log.Error("anúncio ilegível, indo para a quarentena", "erro", err)
		_ = d.Nack(false, false)
		return
	}

	ctx, span := observability.Tracer().Start(ctx, "processar_pagamento")
	defer span.End()
	span.SetAttributes(attribute.String("reserva_id", i.ReservaID))

	log := c.Log.With("reserva_id", i.ReservaID)
	inicio := time.Now()
	desfecho, err := c.Caso.Executar(ctx, i)
	log = log.With("duracao_ms", time.Since(inicio).Milliseconds())
	if tr, e := c.Caso.Repo.BuscarPorReserva(ctx, i.ReservaID); e == nil {
		log = log.With("transacao_id", tr.ID, "status", string(tr.Status))
		span.SetAttributes(
			attribute.String("transacao_id", tr.ID),
			attribute.String("status", string(tr.Status)),
		)
	}

	switch desfecho {
	case usecase.Confirmar:
		if err := d.Ack(false); err != nil {
			log.Error("falha ao confirmar entrega", "erro", err)
		} else {
			log.Info("intenção processada")
		}
	case usecase.Quarentena:
		log.Warn("intenção encaminhada para a quarentena", "erro", err)
		_ = d.Nack(false, false)
	default: // Requeue
		log.Warn("intenção devolvida para nova tentativa", "erro", err)
		// Pequena espera antes de devolver, para não girar em falso enquanto a
		// dependência não volta. O limite de entregas da fila continua valendo.
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
		}
		_ = d.Nack(false, true)
	}
}
