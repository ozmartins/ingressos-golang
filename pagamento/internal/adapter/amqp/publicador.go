package amqp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/oseias/ingressos-golang/pagamento/internal/platform/obs"
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

func (p *Publicador) Publicar(ctx context.Context, f usecase.Fato) (err error) {
	// Span produtor + traceparent nos headers: quem consome (estoque,
	// notificação) continua este trace.
	ctx, span, headers := obs.IniciarPublicacao(ctx, p.exchange, f.RoutingKey)
	defer func() {
		obs.FalharSpan(span, err)
		span.End()
		obs.RegistrarPublicacao(f.RoutingKey, err)
		if err != nil {
			slog.ErrorContext(ctx, "falha ao publicar fato", "routing_key", f.RoutingKey, "message_id", f.MessageID, "erro", err)
		} else {
			slog.InfoContext(ctx, "fato publicado", "routing_key", f.RoutingKey, "message_id", f.MessageID)
		}
	}()

	p.mu.Lock()
	defer p.mu.Unlock()

	conf, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, p.exchange, f.RoutingKey, true, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    f.MessageID,
			Headers:      headers,
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
