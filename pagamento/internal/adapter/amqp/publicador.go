package amqp

import (
	"context"
	"fmt"
	"sync"

	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
)

type Publicador struct {
	mu         sync.Mutex
	ch         *amqp.Channel
	exchange   string
	propagador propagation.TextMapPropagator
}

func NovoPublicador(ch *amqp.Channel, exchange string, p propagation.TextMapPropagator) (*Publicador, error) {
	if err := ch.Confirm(false); err != nil {
		return nil, fmt.Errorf("habilitar confirms: %w", err)
	}
	return &Publicador{ch: ch, exchange: exchange, propagador: p}, nil
}

func (p *Publicador) Publicar(ctx context.Context, f usecase.Fato) error {
	cabecalhos := amqp.Table{}
	if p.propagador != nil {
		p.propagador.Inject(ctx, portadorAMQP(cabecalhos))
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	conf, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, p.exchange, f.RoutingKey, true, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    f.MessageID,
			Headers:      cabecalhos,
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

type portadorAMQP amqp.Table

func (c portadorAMQP) Get(k string) string {
	if v, ok := c[k].(string); ok {
		return v
	}
	return ""
}
func (c portadorAMQP) Set(k, v string) { c[k] = v }
func (c portadorAMQP) Keys() []string {
	ks := make([]string, 0, len(c))
	for k := range c {
		ks = append(ks, k)
	}
	return ks
}
