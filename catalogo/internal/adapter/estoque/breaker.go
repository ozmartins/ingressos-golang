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

// Desfecho classifica o que aconteceu na tentativa de chamada.
type Desfecho int

// Desfechos possíveis de uma tentativa de chamada ao estoque.
const (
	DesfechoResposta Desfecho = iota // o estoque respondeu (com sucesso ou não)
	DesfechoFalha                    // erro de transporte ou tempo excedido
	DesfechoRecusado                 // nem chegou a chamar: recusa rápida ativa
)

// RecusaRapida interrompe as chamadas após uma sequência de falhas e retoma
// sozinha depois de um intervalo (FR-030, SC-007).
//
// Sem isso, cada solicitação pagaria o timeout inteiro durante uma queda do
// estoque — e continuaria pressionando um serviço que já está em dificuldade.
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
		MaxRequests: 1, // no estado semiaberto, uma única chamada de prova
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

// Executar roda a chamada sob a proteção da recusa rápida.
func (r *RecusaRapida) Executar(chamada func() (*estoquepb.RespostaBloqueio, error)) (*estoquepb.RespostaBloqueio, Desfecho, error) {
	resposta, err := r.cb.Execute(func() (*estoquepb.RespostaBloqueio, error) {
		resp, err := chamada()
		if err != nil {
			return nil, err
		}
		// Poltrona ocupada é resposta legítima do estoque, não falha dele: não
		// pode contar para abrir a recusa rápida.
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

// ehTempoExcedido distingue o tempo excedido dos demais erros de transporte,
// para efeito de métrica. Para o cliente, ambos são a mesma resposta.
func ehTempoExcedido(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	s, ok := status.FromError(err)
	return ok && s.Code() == codes.DeadlineExceeded
}
