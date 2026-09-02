package observability

import (
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

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

func Propagador() propagation.TextMapPropagator {
	p := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	)
	otel.SetTextMapPropagator(p)
	return p
}

func Tracer() trace.Tracer {
	return otel.Tracer("servico-notificacao")
}
