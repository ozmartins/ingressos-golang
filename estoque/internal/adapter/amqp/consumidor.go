package amqp

import (
	"context"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ErroDefinitivo struct{ Err error }

func (e ErroDefinitivo) Error() string { return e.Err.Error() }
func (e ErroDefinitivo) Unwrap() error { return e.Err }

func Definitivo(err error) error { return ErroDefinitivo{Err: err} }

type Manipulador func(ctx context.Context, msg amqp.Delivery) (desfecho string, err error)

type Consumidor struct {
	Conexao  *Conexao
	Fila     string
	Prefetch int

	Trata Manipulador
}

func (c *Consumidor) Iniciar(ctx context.Context) error {
	canal, err := c.Conexao.Canal()
	if err != nil {
		return err
	}
	if err := canal.Qos(c.Prefetch, 0, false); err != nil {
		canal.Close()
		return err
	}

	entregas, err := canal.Consume(c.Fila, "", false, false, false, false, nil)
	if err != nil {
		canal.Close()
		return err
	}

	go func() {
		defer canal.Close()
		for {
			select {
			case <-ctx.Done():

				return
			case msg, aberto := <-entregas:
				if !aberto {
					return
				}
				c.processar(ctx, msg)
			}
		}
	}()
	return nil
}

func (c *Consumidor) processar(ctx context.Context, msg amqp.Delivery) {
	_, err := c.Trata(ctx, msg)

	switch {
	case err == nil:
		_ = msg.Ack(false)
	case errors.As(err, &ErroDefinitivo{}):
		_ = msg.Nack(false, false)
	default:
		time.Sleep(500 * time.Millisecond)
		_ = msg.Nack(false, true)
	}
}
