package amqp

import (
	"context"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

// chaveIdempotencia devolve a chave de deduplicação da mensagem. O contrato diz
// que ela viaja no message_id; quando o publicador não a preenche, caímos no
// identificador do próprio agregado, que é a chave declarada no contrato.
func chaveIdempotencia(msg amqp.Delivery, alternativa string) string {
	if msg.MessageId != "" {
		return msg.MessageId
	}
	return alternativa
}

// ConsumirPagamentoSucesso liga a fila de pagamento aprovado ao caso de uso.
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
				// Falha de infraestrutura: devolve à fila e tenta de novo.
				return "falha", err
			}
			return resultado.String(), nil
		},
	}
	return c.Iniciar(ctx)
}

// ConsumirPagamentoFalhou liga a fila de pagamento recusado ao caso de uso.
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

// ConsumirSessaoCriada liga a fila de sessão criada ao provisionamento.
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
				// Layout inválido é definitivo; o resto é transitório.
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

// Layout inválido é erro de conteúdo e vai para a DLQ; falha de infraestrutura
// é transitória e volta para a fila. A distinção vem do erro de domínio, não de
// comparação de texto (constituição, princípio IV).
