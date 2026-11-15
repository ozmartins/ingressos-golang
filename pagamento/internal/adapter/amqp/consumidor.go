package amqp

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/platform/obs"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumidor struct {
	Canal    *amqp.Channel
	Fila     string
	Prefetch int
	Caso     usecase.RegistrarIntencao

	EmAndamento *Medidor
}

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

func (m *Medidor) Maximo() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maximo
}

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

	// Continua o trace de quem publicou a mensagem (traceparent nos headers).
	ctx, span := obs.IniciarConsumo(ctx, c.Fila, d)
	defer span.End()
	inicio := time.Now()

	var i usecase.Intencao
	if err := json.Unmarshal(d.Body, &i); err != nil {
		_ = d.Nack(false, false)
		obs.FalharSpan(span, err)
		obs.RegistrarConsumo(c.Fila, "invalida", time.Since(inicio))
		slog.ErrorContext(ctx, "mensagem inválida enviada à fila morta", "queue", c.Fila, "message_id", d.MessageId, "erro", err)
		return
	}

	desfecho, errCaso := c.Caso.Executar(ctx, i)
	nome := nomeDoDesfecho(desfecho)
	obs.FalharSpan(span, errCaso)
	obs.RegistrarConsumo(c.Fila, nome, time.Since(inicio))
	if errCaso != nil {
		slog.ErrorContext(ctx, "falha ao registrar intenção de pagamento", "queue", c.Fila,
			"message_id", d.MessageId, "outcome", nome, "erro", errCaso)
	} else {
		slog.InfoContext(ctx, "mensagem tratada", "queue", c.Fila, "message_id", d.MessageId,
			"reserva_id", i.ReservaID, "outcome", nome)
	}

	switch desfecho {
	case usecase.Confirmar:
		_ = d.Ack(false)
	case usecase.Quarentena:
		_ = d.Nack(false, false)

	default:
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
		}
		_ = d.Nack(false, true)
	}
}

func nomeDoDesfecho(d usecase.Desfecho) string {
	switch d {
	case usecase.Confirmar:
		return "confirmar"
	case usecase.Quarentena:
		return "quarentena"
	default:
		return "devolver"
	}
}
