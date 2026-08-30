package grpc

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
)

// Desfechos observáveis do bloqueio (FR-043).
const (
	desfechoConcedido    = "concedido"
	desfechoIndisponivel = "indisponivel"
	desfechoInvalido     = "invalido"
	desfechoFalha        = "falha"
	desfechoOK           = "ok"
)

// metricas agrupa os instrumentos das operações síncronas.
type metricas struct {
	total   metric.Int64Counter
	duracao metric.Float64Histogram
}

func novasMetricas(obs *observability.Observabilidade) (*metricas, error) {
	total, err := obs.Medidor.Int64Counter("estoque.operacao.total",
		metric.WithDescription("operações do canal síncrono por desfecho"))
	if err != nil {
		return nil, err
	}
	duracao, err := obs.Medidor.Float64Histogram("estoque.operacao.duracao",
		metric.WithDescription("duração das operações do canal síncrono"),
		metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}
	return &metricas{total: total, duracao: duracao}, nil
}

func (m *metricas) registrar(ctx context.Context, operacao, desfecho string, inicio time.Time) {
	atributos := metric.WithAttributes(
		observability.Atributo("operacao", operacao),
		observability.Atributo("desfecho", desfecho),
	)
	m.total.Add(ctx, 1, atributos)
	m.duracao.Record(ctx, float64(time.Since(inicio).Microseconds())/1000.0, atributos)
}
