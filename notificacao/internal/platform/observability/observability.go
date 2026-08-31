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

// Logger devolve um logger JSON no nível pedido.
//
// Regra dura da FR-021: o codigo_qr NUNCA vai para o log, nem para atributo de
// rastro, nem para mensagem de erro. O que identifica a operação é o
// ingresso_id. Log é copiado, agregado e lido por muita gente — um código de
// acesso em log é um ingresso utilizável em log (research.md D13).
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

// Propagador é o contexto de rastreamento W3C lido dos cabeçalhos AMQP: o
// processamento abre span filho do pagamento, de modo que a jornada de compra
// fique num rastro só (research.md D13).
func Propagador() propagation.TextMapPropagator {
	p := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	)
	otel.SetTextMapPropagator(p)
	return p
}

// Tracer devolve o tracer nomeado do serviço, para abrir spans no consumo.
func Tracer() trace.Tracer {
	return otel.Tracer("servico-notificacao")
}
