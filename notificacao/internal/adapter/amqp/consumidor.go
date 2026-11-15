package amqp

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/platform/obs"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/codes"
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
	Canal    *amqp.Channel
	Fila     string
	Prefetch int
	Caso     usecase.EmitirIngresso

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
	// Continua o trace de quem publicou a mensagem (traceparent nos headers).
	ctx, span := obs.IniciarConsumo(ctx, c.Fila, d)
	defer span.End()
	inicio := time.Now()

	gesto := c.Classificar(ctx, d.Body)
	obs.RegistrarConsumo(c.Fila, gesto.String(), time.Since(inicio))
	if gesto == Confirmar {
		slog.InfoContext(ctx, "mensagem tratada", "queue", c.Fila, "message_id", d.MessageId, "gesto", gesto.String())
	} else {
		span.SetStatus(codes.Error, gesto.String())
		slog.ErrorContext(ctx, "mensagem não confirmada", "queue", c.Fila, "message_id", d.MessageId, "gesto", gesto.String())
	}

	switch gesto {
	case Confirmar:
		_ = d.Ack(false)

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

func (c *Consumidor) Classificar(ctx context.Context, corpo []byte) Gesto {
	a, err := usecase.DecodificarAnuncio(corpo)
	if err != nil {
		slog.ErrorContext(ctx, "anúncio de pagamento inválido", "erro", err)
		return Quarentena
	}

	desfecho, err := c.Caso.Executar(ctx, a)
	if err != nil {
		slog.ErrorContext(ctx, "falha ao emitir ingresso", "erro", err)
	}
	switch desfecho {
	case usecase.Confirmar:
		return Confirmar
	case usecase.Quarentena:

		return Quarentena
	default:

		return Devolver
	}
}

func (c *Consumidor) espera() time.Duration {
	if c.EsperaAntesDeDevolver > 0 {
		return c.EsperaAntesDeDevolver
	}
	return 200 * time.Millisecond
}
