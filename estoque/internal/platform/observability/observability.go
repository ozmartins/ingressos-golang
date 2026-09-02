package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const nomeServico = "servico-estoque"

type Observabilidade struct {
	Log        *slog.Logger
	Tracer     trace.Tracer
	Medidor    metric.Meter
	Propagador propagation.TextMapPropagator

	desligar []func(context.Context) error
}

func Iniciar(ctx context.Context, nivel, endpointOTLP string) (*Observabilidade, error) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseNivel(nivel)}))
	slog.SetDefault(log)

	propagador := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagador)

	obs := &Observabilidade{Log: log, Propagador: propagador}

	if endpointOTLP == "" {
		obs.Tracer = otel.Tracer(nomeServico)
		obs.Medidor = otel.Meter(nomeServico)
		log.Info("observabilidade sem exportação", "motivo", "OTEL_EXPORTER_OTLP_ENDPOINT ausente")
		return obs, nil
	}

	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(nomeServico)))
	if err != nil {
		return nil, err
	}

	expTrace, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(endpointOTLP))
	if err != nil {
		return nil, err
	}
	provTrace := sdktrace.NewTracerProvider(sdktrace.WithBatcher(expTrace), sdktrace.WithResource(res))
	otel.SetTracerProvider(provTrace)
	obs.desligar = append(obs.desligar, provTrace.Shutdown)

	expMetric, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpointURL(endpointOTLP))
	if err != nil {
		return nil, err
	}
	provMetric := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(expMetric)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(provMetric)
	obs.desligar = append(obs.desligar, provMetric.Shutdown)

	obs.Tracer = otel.Tracer(nomeServico)
	obs.Medidor = otel.Meter(nomeServico)
	return obs, nil
}

func (o *Observabilidade) Desligar(ctx context.Context) {
	for _, f := range o.desligar {
		if err := f(ctx); err != nil {
			o.Log.Warn("falha ao desligar exportador", "erro", err.Error())
		}
	}
}

func (o *Observabilidade) LogOperacao(ctx context.Context, operacao, desfecho string, inicio time.Time, extras ...any) {
	campos := []any{
		"operacao", operacao,
		"desfecho", desfecho,
		"duracao_ms", time.Since(inicio).Milliseconds(),
		"trace_id", TraceID(ctx),
	}
	campos = append(campos, extras...)

	if desfecho == "falha" {
		o.Log.ErrorContext(ctx, "operação concluída", campos...)
		return
	}
	o.Log.InfoContext(ctx, "operação concluída", campos...)
}

func TraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

func Atributo(chave, valor string) attribute.KeyValue { return attribute.String(chave, valor) }

func parseNivel(nivel string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(nivel)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
