package amqp

import (
	"context"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func chaveIdempotencia(msg amqp.Delivery, alternativa string) string {
	if msg.MessageId != "" {
		return msg.MessageId
	}
	return alternativa
}

func ConsumirPagamentoSucesso(ctx context.Context, conexao *Conexao, prefetch int,
	obs *observability.Observabilidade, uc usecase.ConfirmarReserva) error {

	c := &Consumidor{
		Conexao: conexao, Fila: FilaPagamentoSucesso, Prefetch: prefetch, Obs: obs,
		Trata: func(ctx context.Context, msg amqp.Delivery) (string, error) {
			desfecho, err := usecase.LerDesfechoPagamento(msg.Body)
			if err != nil {
				return "invalida", Definitivo(err)
			}
			resultado, err := uc.Executar(ctx, FilaPagamentoSucesso,
				chaveIdempotencia(msg, desfecho.ReservaID), desfecho.ReservaID)
			if err != nil {
				return "falha", err
			}
			return resultado.String(), nil
		},
	}
	return c.Iniciar(ctx)
}

func ConsumirPagamentoFalhou(ctx context.Context, conexao *Conexao, prefetch int,
	obs *observability.Observabilidade, uc usecase.CancelarReserva) error {

	c := &Consumidor{
		Conexao: conexao, Fila: FilaPagamentoFalhou, Prefetch: prefetch, Obs: obs,
		Trata: func(ctx context.Context, msg amqp.Delivery) (string, error) {
			desfecho, err := usecase.LerDesfechoPagamento(msg.Body)
			if err != nil {
				return "invalida", Definitivo(err)
			}
			resultado, err := uc.Executar(ctx, FilaPagamentoFalhou,
				chaveIdempotencia(msg, desfecho.ReservaID), desfecho.ReservaID)
			if err != nil {
				return "falha", err
			}
			return resultado.String(), nil
		},
	}
	return c.Iniciar(ctx)
}

func ConsumirSessaoCriada(ctx context.Context, conexao *Conexao, prefetch int,
	obs *observability.Observabilidade, uc usecase.ProvisionarSessao) error {

	c := &Consumidor{
		Conexao: conexao, Fila: FilaSessaoCriada, Prefetch: prefetch, Obs: obs,
		Trata: func(ctx context.Context, msg amqp.Delivery) (string, error) {
			evento, err := usecase.LerSessaoCriada(msg.Body)
			if err != nil {
				return "invalida", Definitivo(err)
			}
			resultado, err := uc.Executar(ctx, FilaSessaoCriada,
				chaveIdempotencia(msg, evento.SessaoID), evento)
			if err != nil {
				if errors.Is(err, shared.ErrSolicitacaoInvalida) {
					return "invalida", Definitivo(err)
				}
				return "falha", err
			}
			return resultado.String(), nil
		},
	}
	return c.Iniciar(ctx)
}
