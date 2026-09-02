package estoque

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	estoquepb "github.com/oseias/ingressos-golang/catalogo/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"
)

type Desfecho int

const (
	DesfechoResposta Desfecho = iota
	DesfechoFalha
	DesfechoRecusado
)

type RecusaRapida struct {
	cb       *gobreaker.CircuitBreaker[*estoquepb.RespostaBloqueio]
	metricas *observability.Metricas
}

func NovaRecusaRapida(falhasParaAbrir uint32, intervaloAberto time.Duration, m *observability.Metricas) *RecusaRapida {
	if falhasParaAbrir == 0 {
		falhasParaAbrir = 5
	}
	if intervaloAberto <= 0 {
		intervaloAberto = 30 * time.Second
	}
	r := &RecusaRapida{metricas: m}
	r.cb = gobreaker.NewCircuitBreaker[*estoquepb.RespostaBloqueio](gobreaker.Settings{
		Name:        "estoque.bloqueio",
		MaxRequests: 1,
		Timeout:     intervaloAberto,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= falhasParaAbrir
		},
		OnStateChange: func(_ string, _, para gobreaker.State) {
			r.registrarEstado(para)
		},
	})
	return r
}

func (r *RecusaRapida) Executar(chamada func() (*estoquepb.RespostaBloqueio, error)) (*estoquepb.RespostaBloqueio, Desfecho, error) {
	resposta, err := r.cb.Execute(func() (*estoquepb.RespostaBloqueio, error) {
		resp, err := chamada()
		if err != nil {
			return nil, err
		}
		return resp, nil
	})

	switch {
	case err == nil:
		return resposta, DesfechoResposta, nil
	case errors.Is(err, gobreaker.ErrOpenState), errors.Is(err, gobreaker.ErrTooManyRequests):
		return nil, DesfechoRecusado, err
	default:
		return nil, DesfechoFalha, err
	}
}

func (r *RecusaRapida) registrarEstado(s gobreaker.State) {
	if r.metricas == nil {
		return
	}
	var v int64
	switch s {
	case gobreaker.StateClosed:
		v = 0
	case gobreaker.StateHalfOpen:
		v = 1
	case gobreaker.StateOpen:
		v = 2
	}
	r.metricas.BreakerEstado.Record(context.Background(), v)
}

func ehTempoExcedido(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	s, ok := status.FromError(err)
	return ok && s.Code() == codes.DeadlineExceeded
}
