package amqp

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Topologia struct {
	Exchange       string
	ExchangeDLX    string
	Fila           string
	FilaDLQ        string
	Binding        string
	LimiteEntregas int
}

func (t Topologia) Declarar(ch *amqp.Channel) error {
	for _, ex := range []string{t.Exchange, t.ExchangeDLX} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("declarar exchange %s: %w", ex, err)
		}
	}

	if _, err := ch.QueueDeclare(t.FilaDLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declarar fila morta %s: %w", t.FilaDLQ, err)
	}
	if err := ch.QueueBind(t.FilaDLQ, t.Fila, t.ExchangeDLX, false, nil); err != nil {
		return fmt.Errorf("vincular fila morta: %w", err)
	}

	reentregas := t.LimiteEntregas - 1
	if reentregas < 0 {
		reentregas = 0
	}
	args := amqp.Table{
		"x-queue-type":              "quorum",
		"x-delivery-limit":          int32(reentregas),
		"x-dead-letter-exchange":    t.ExchangeDLX,
		"x-dead-letter-routing-key": t.Fila,
	}
	if _, err := ch.QueueDeclare(t.Fila, true, false, false, false, args); err != nil {
		return fmt.Errorf("declarar fila %s: %w", t.Fila, err)
	}
	if err := ch.QueueBind(t.Fila, t.Binding, t.Exchange, false, nil); err != nil {
		return fmt.Errorf("vincular fila %s: %w", t.Fila, err)
	}
	return nil
}
