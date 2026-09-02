package amqp

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/platform/observability"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Gesto int

const (
	Confirmar Gesto = iota
	Quarentena
	Devolver
)

func (g Gesto) String() string {
	switch g {
	case Confirmar:
		return "ack"
	case Quarentena:
		return "nack_sem_reentrega"
	default:
		return "nack_com_reentrega"
	}
}

type Consumidor struct {
	Canal                 *amqp.Channel
	Fila                  string
	Prefetch              int
	Caso                  usecase.EmitirIngresso
	Log                   *slog.Logger
	Propagador            propagation.TextMapPropagator
	EsperaAntesDeDevolver time.Duration
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
	if c.Propagador != nil {
		ctx = c.Propagador.Extract(ctx, portadorAMQP(d.Headers))
	}
	ctx, span := observability.Tracer().Start(ctx, "emitir_ingresso")
	defer span.End()

	gesto := c.Classificar(ctx, d.Body, span.SpanContext().IsValid())

	switch gesto {
	case Confirmar:
		if err := d.Ack(false); err != nil {
			c.log().Error("falha ao confirmar entrega", "erro", err)
		}
	case Quarentena:
		_ = d.Nack(false, false)
	default:
		select {
		case <-time.After(c.espera()):
		case <-ctx.Done():
		}
		_ = d.Nack(false, true)
	}
}

func (c *Consumidor) Classificar(ctx context.Context, corpo []byte, comRastro bool) Gesto {
	a, err := usecase.DecodificarAnuncio(corpo)
	if err != nil {
		c.log().Error("anúncio ilegível, indo para a quarentena",
			"erro", err, "gesto", Quarentena.String())
		return Quarentena
	}
	if comRastro {
		if span := trechoAtual(ctx); span != nil {
			span.SetAttributes(attribute.String("reserva_id", a.ReservaID))
		}
	}

	desfecho, err := c.Caso.Executar(ctx, a)
	switch desfecho {
	case usecase.Confirmar:
		return Confirmar
	case usecase.Quarentena:
		c.log().Warn("anúncio encaminhado para a quarentena",
			"reserva_id", a.ReservaID, "erro", err, "gesto", Quarentena.String())
		return Quarentena
	default:
		c.log().Warn("anúncio devolvido para nova tentativa",
			"reserva_id", a.ReservaID, "erro", err, "gesto", Devolver.String())
		return Devolver
	}
}

func (c *Consumidor) espera() time.Duration {
	if c.EsperaAntesDeDevolver > 0 {
		return c.EsperaAntesDeDevolver
	}
	return 200 * time.Millisecond
}

func (c *Consumidor) log() *slog.Logger {
	if c.Log != nil {
		return c.Log
	}
	return slog.Default()
}

type portadorAMQP amqp.Table

func (p portadorAMQP) Get(chave string) string {
	if v, ok := p[chave]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (p portadorAMQP) Set(chave, valor string) { p[chave] = valor }

func (p portadorAMQP) Keys() []string {
	ks := make([]string, 0, len(p))
	for k := range p {
		ks = append(ks, k)
	}
	return ks
}

func trechoAtual(ctx context.Context) interface {
	SetAttributes(...attribute.KeyValue)
} {
	s := trace.SpanFromContext(ctx)
	if !s.SpanContext().IsValid() {
		return nil
	}
	return s
}
