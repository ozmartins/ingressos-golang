package amqp

import (
	"context"

	"sync"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
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
	gesto := c.Classificar(ctx, d.Body)

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
		return Quarentena
	}

	desfecho, err := c.Caso.Executar(ctx, a)
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
