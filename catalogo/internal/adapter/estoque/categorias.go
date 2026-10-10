package estoque

import (
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

const (
	razaoSolicitacaoInvalida   = "SOLICITACAO_INVALIDA"
	razaoLimiteExcedido        = "LIMITE_POLTRONAS_EXCEDIDO"
	razaoSessaoNaoProvisionada = "SESSAO_NAO_PROVISIONADA"
	razaoPoltronaInexistente   = "POLTRONA_INEXISTENTE"
)

func traduzirErroDeChamada(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("%w: %v", shared.ErrEstoqueIndisponivel, err)
	}

	switch st.Code() {
	case codes.InvalidArgument:
		if razao(st) == razaoLimiteExcedido {
			if limite := metadado(st, "limite"); limite != "" {
				return fmt.Errorf("%w: no máximo %s poltronas por reserva",
					shared.ErrSolicitacaoRecusadaPeloEstoque, limite)
			}
		}
		return fmt.Errorf("%w: %s", shared.ErrSolicitacaoRecusadaPeloEstoque, st.Message())

	case codes.FailedPrecondition:
		switch razao(st) {
		case razaoPoltronaInexistente:
			return fmt.Errorf("%w: %s", shared.ErrPoltronaInexistente, st.Message())
		case razaoSessaoNaoProvisionada:
			return shared.ErrSessaoSemPoltronas
		}
		return fmt.Errorf("%w: %s", shared.ErrSolicitacaoRecusadaPeloEstoque, st.Message())

	case codes.Internal:
		return fmt.Errorf("%w: %v", shared.ErrEstoqueComDefeito, st.Message())

	default:
		return fmt.Errorf("%w: %v", shared.ErrEstoqueIndisponivel, err)
	}
}

func recusaDoEstoque(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.NotFound:
		return true
	default:
		return false
	}
}

func razao(st *status.Status) string {
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok {
			return info.GetReason()
		}
	}
	return ""
}

func metadado(st *status.Status, chave string) string {
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok {
			return info.GetMetadata()[chave]
		}
	}
	return ""
}
