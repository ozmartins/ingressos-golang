package obs

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// manipuladorDeTrace acrescenta trace_id e span_id a todo registro cujo contexto
// carregue um span válido, e request_id quando o contexto tiver um. É o que liga o log ao trace: use sempre as variantes
// com contexto (slog.InfoContext, ...).
type manipuladorDeTrace struct{ slog.Handler }

func (m manipuladorDeTrace) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return m.Handler.Handle(ctx, r)
}

func (m manipuladorDeTrace) WithAttrs(a []slog.Attr) slog.Handler {
	return manipuladorDeTrace{m.Handler.WithAttrs(a)}
}

func (m manipuladorDeTrace) WithGroup(n string) slog.Handler {
	return manipuladorDeTrace{m.Handler.WithGroup(n)}
}

func configurarLog(servico string) {
	var nivel slog.LevelVar
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		nivel.Set(slog.LevelDebug)
	case "warn":
		nivel.Set(slog.LevelWarn)
	case "error":
		nivel.Set(slog.LevelError)
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: &nivel})
	slog.SetDefault(slog.New(manipuladorDeTrace{h}).With("service", servico))
}

func erroDeLog(msg string, err error) { slog.Error(msg, "erro", err) }
