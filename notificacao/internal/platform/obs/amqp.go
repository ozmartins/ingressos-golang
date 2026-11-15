package obs

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const nomeTracer = "ingressos"

func tracer() trace.Tracer { return otel.Tracer(nomeTracer) }

// carrierAMQP adapta os headers de uma mensagem AMQP ao propagador do
// OpenTelemetry: é o que leva o traceparent de um serviço a outro pelo broker.
type carrierAMQP amqp.Table

func (c carrierAMQP) Get(k string) string {
	if v, ok := c[k].(string); ok {
		return v
	}
	return ""
}
func (c carrierAMQP) Set(k, v string) { c[k] = v }
func (c carrierAMQP) Keys() []string {
	ks := make([]string, 0, len(c))
	for k := range c {
		ks = append(ks, k)
	}
	return ks
}

// IniciarPublicacao abre o span produtor de uma mensagem e devolve os headers
// AMQP já com o traceparent desse span. Quem chama deve fazer span.End() e
// FalharSpan em caso de erro.
func IniciarPublicacao(ctx context.Context, exchange, routingKey string) (context.Context, trace.Span, amqp.Table) {
	ctx, span := tracer().Start(ctx, routingKey+" publish", trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.operation", "publish"),
			attribute.String("messaging.destination.name", exchange),
			attribute.String("messaging.rabbitmq.destination.routing_key", routingKey),
		))
	headers := amqp.Table{}
	otel.GetTextMapPropagator().Inject(ctx, carrierAMQP(headers))
	return ctx, span, headers
}

// IniciarConsumo continua o trace de quem publicou a mensagem: extrai o
// traceparent dos headers e abre o span consumidor como filho dele. Mensagem
// sem header (publicada à mão, por exemplo) começa um trace novo.
func IniciarConsumo(ctx context.Context, fila string, msg amqp.Delivery) (context.Context, trace.Span) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrierAMQP(msg.Headers))
	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.operation", "process"),
		attribute.String("messaging.destination.name", fila),
		attribute.String("messaging.rabbitmq.destination.routing_key", msg.RoutingKey),
	}
	if msg.MessageId != "" {
		attrs = append(attrs, attribute.String("messaging.message.id", msg.MessageId))
	}
	return tracer().Start(ctx, fila+" process", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attrs...))
}

// FalharSpan marca o span como erro.
func FalharSpan(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// Traceparent serializa o contexto de trace corrente ("" se não houver). É o
// que a caixa de saída grava junto do fato, para o publicador, que roda depois
// e fora da requisição, continuar o mesmo trace.
func Traceparent(ctx context.Context) string {
	m := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, m)
	return m["traceparent"]
}

// ComTraceparent é o inverso de Traceparent.
func ComTraceparent(ctx context.Context, tp string) context.Context {
	if tp == "" {
		return ctx
	}
	return propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier{"traceparent": tp})
}

// DesfechoDe converte um valor qualquer num rótulo de métrica.
func DesfechoDe(v any) string { return fmt.Sprint(v) }
