// Package observability configura logs estruturados e OpenTelemetry.
package observability

import (
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Logger devolve um logger JSON no nível pedido. Todo registro do fluxo de
// pagamento carrega reserva_id e transacao_id (FR-024).
func Logger(nivel string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(nivel) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}

// Propagador é o contexto de rastreamento W3C usado nos cabeçalhos AMQP: o
// consumo abre span filho do bloqueio, e a publicação reinjeta o contexto, de
// modo que a jornada de compra fique num rastro só (research.md D11).
func Propagador() propagation.TextMapPropagator {
	p := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	)
	otel.SetTextMapPropagator(p)
	return p
}

// Tracer devolve o tracer nomeado do serviço, para abrir spans no consumo.
func Tracer() trace.Tracer {
	return otel.Tracer("servico-pagamento")
}
