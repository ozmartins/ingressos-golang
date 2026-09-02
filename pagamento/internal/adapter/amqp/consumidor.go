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

type Consumidor struct {
	Canal      *amqp.Channel
	Fila       string
	Prefetch   int
	Caso       usecase.ProcessarPagamento
	Log        *slog.Logger
	Propagador propagation.TextMapPropagator

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

	if c.Propagador != nil {
		ctx = c.Propagador.Extract(ctx, portadorAMQP(d.Headers))
	}

	var i usecase.Intencao
	if err := json.Unmarshal(d.Body, &i); err != nil {
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
	default:
		log.Warn("intenção devolvida para nova tentativa", "erro", err)
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
		}
		_ = d.Nack(false, true)
	}
}
