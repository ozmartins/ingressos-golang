// Package observability configura logs, rastreamento e métricas.
package observability

import (
	"context"
	"fmt"
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
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const NomeServico = "servico-catalogo"

// init instala a propagação W3C assim que o pacote é carregado.
//
// Fica aqui, e não em Iniciar, porque propagar o contexto recebido é uma
// obrigação do serviço (FR-036) que não depende de haver coletor configurado —
// e porque um teste que não chama Iniciar ainda assim exercita a propagação
// real, em vez de passar por engano contra um propagador nulo.
func init() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
}

// Desfechos das chamadas ao estoque, usados como rótulo da métrica
// estoque.bloqueio.total (FR-035).
const (
	// DesfechoSucesso: o estoque respondeu e concedeu o bloqueio.
	DesfechoSucesso = "sucesso"
	// DesfechoIndisponivel: o estoque não foi alcançado (erro de transporte).
	DesfechoIndisponivel = "indisponivel"
	// DesfechoTimeout: a chamada excedeu o tempo máximo de espera.
	DesfechoTimeout = "timeout"
	// DesfechoRecusaRapida: a chamada nem foi feita, por recusa rápida ativa.
	DesfechoRecusaRapida = "recusa_rapida"
	// DesfechoPoltronasIndisponiveis: o estoque respondeu e negou. É desfecho de
	// negócio, não falha de integração — separá-lo evita que uma disputa normal
	// por assentos pareça degradação do parceiro nos painéis.
	DesfechoPoltronasIndisponiveis = "poltronas_indisponiveis"
)

// Metricas reúne os instrumentos nomeados pelo plano.
type Metricas struct {
	EstoqueDuracao metric.Float64Histogram
	EstoqueTotal   metric.Int64Counter
	BreakerEstado  metric.Int64Gauge
	HTTPDuracao    metric.Float64Histogram
}

// Encerrar libera os provedores; devolvido por Iniciar.
type Encerrar func(context.Context) error

// ConfigurarLogger instala o logger estruturado em JSON como padrão do processo.
func ConfigurarLogger(nivel string) *slog.Logger {
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.ToLower(nivel))); err != nil {
		l = slog.LevelInfo
	}
	logger := slog.New(&manipuladorComRastro{
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}),
	})
	slog.SetDefault(logger)
	return logger
}

// manipuladorComRastro injeta trace_id e span_id em todo registro emitido dentro
// de um span ativo. É o que permite ligar um log a um rastro (SC-011).
type manipuladorComRastro struct{ slog.Handler }

func (h *manipuladorComRastro) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *manipuladorComRastro) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &manipuladorComRastro{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *manipuladorComRastro) WithGroup(name string) slog.Handler {
	return &manipuladorComRastro{Handler: h.Handler.WithGroup(name)}
}

// Iniciar configura rastreamento e métricas. Sem endpoint OTLP configurado, os
// provedores ficam sem exportador: o serviço continua funcionando e apenas não
// envia sinais — a ausência de coletor não é motivo para derrubar o processo.
func Iniciar(ctx context.Context, endpointOTLP string) (*Metricas, Encerrar, error) {
	// A versão do semconv precisa acompanhar a do SDK: resource.Merge recusa
	// juntar descrições com URLs de esquema diferentes, e a falha só aparece na
	// inicialização do processo — não em teste que não chame Iniciar.
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(NomeServico),
	))
	if err != nil {
		return nil, nil, fmt.Errorf("montando resource: %w", err)
	}

	var encerradores []func(context.Context) error

	if endpointOTLP != "" {
		expTrace, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("exportador de rastros: %w", err)
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(expTrace),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		encerradores = append(encerradores, tp.Shutdown)

		expMetric, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("exportador de métricas: %w", err)
		}
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(expMetric, sdkmetric.WithInterval(15*time.Second))),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		encerradores = append(encerradores, mp.Shutdown)
	}

	m, err := NovasMetricas()
	if err != nil {
		return nil, nil, err
	}

	encerrar := func(ctx context.Context) error {
		for _, fn := range encerradores {
			if err := fn(ctx); err != nil {
				return err
			}
		}
		return nil
	}
	return m, encerrar, nil
}

// NovasMetricas registra os instrumentos no provedor corrente.
func NovasMetricas() (*Metricas, error) {
	meter := otel.Meter(NomeServico)

	httpDur, err := meter.Float64Histogram("http.server.request.duration",
		metric.WithDescription("Duração das requisições HTTP atendidas"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	estoqueDur, err := meter.Float64Histogram("estoque.bloqueio.duration",
		metric.WithDescription("Duração das chamadas de bloqueio ao Servico-Estoque"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	estoqueTotal, err := meter.Int64Counter("estoque.bloqueio.total",
		metric.WithDescription("Chamadas de bloqueio por desfecho"))
	if err != nil {
		return nil, err
	}
	breaker, err := meter.Int64Gauge("estoque.breaker.state",
		metric.WithDescription("Estado da recusa rápida: 0 fechado, 1 semiaberto, 2 aberto"))
	if err != nil {
		return nil, err
	}
	return &Metricas{
		HTTPDuracao:    httpDur,
		EstoqueDuracao: estoqueDur,
		EstoqueTotal:   estoqueTotal,
		BreakerEstado:  breaker,
	}, nil
}

// RotuloDesfecho monta o atributo de desfecho da chamada ao estoque.
func RotuloDesfecho(desfecho string) attribute.KeyValue {
	return attribute.String("desfecho", desfecho)
}
