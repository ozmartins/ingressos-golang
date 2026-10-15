package amqp

import (
	"context"
	"fmt"
	"sync"

	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publicador struct {
	mu       sync.Mutex
	ch       *amqp.Channel
	exchange string
}

func NovoPublicador(ch *amqp.Channel, exchange string) (*Publicador, error) {
	if err := ch.Confirm(false); err != nil {
		return nil, fmt.Errorf("habilitar confirms: %w", err)
	}
	return &Publicador{ch: ch, exchange: exchange}, nil
}

func (p *Publicador) Publicar(ctx context.Context, f usecase.Fato) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	conf, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, p.exchange, f.RoutingKey, true, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    f.MessageID,
			Body:         f.Payload,
		})
	if err != nil {
		return fmt.Errorf("publicar %s: %w", f.RoutingKey, err)
	}
	ok, err := conf.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("aguardar confirmação de %s: %w", f.RoutingKey, err)
	}
	if !ok {
		return fmt.Errorf("broker recusou %s", f.RoutingKey)
	}
	return nil
}
