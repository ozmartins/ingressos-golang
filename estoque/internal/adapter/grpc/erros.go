// Package grpc adapta o núcleo ao contrato gRPC servido por este serviço.
package grpc

import (
	"errors"
	"strconv"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// dominioErros é o domínio declarado nos ErrorInfo. Faz parte do contrato.
const dominioErros = "estoque.ingressos"

// Razões estáveis. Constituição, princípio IV: a categoria é versionada como
// parte do contrato; o texto humano pode mudar de redação livremente.
const (
	razaoSolicitacaoInvalida     = "SOLICITACAO_INVALIDA"
	razaoLimiteExcedido          = "LIMITE_POLTRONAS_EXCEDIDO"
	razaoSessaoNaoProvisionada   = "SESSAO_NAO_PROVISIONADA"
	razaoPoltronaInexistente     = "POLTRONA_INEXISTENTE"
	razaoSessaoDesconhecida      = "SESSAO_DESCONHECIDA"
	razaoDependenciaIndisponivel = "DEPENDENCIA_INDISPONIVEL"
)

// paraStatus traduz um erro de domínio no status gRPC de contracts/erros.md.
//
// Nenhum detalhe interno atravessa esta função: a mensagem que sai é a do erro
// de domínio, nunca a do driver, do SQL ou do endereço da dependência. O detalhe
// vai para o log, correlacionado pelo trace_id (princípio IV).
func paraStatus(err error, limitePoltronas int) error {
	switch {
	case err == nil:
		return nil

	case errors.Is(err, shared.ErrLimiteExcedido):
		return comDetalhe(codes.InvalidArgument, "quantidade de poltronas acima do limite por bloqueio",
			razaoLimiteExcedido, map[string]string{"limite": strconv.Itoa(limitePoltronas)})

	case errors.Is(err, shared.ErrSolicitacaoInvalida):
		return comDetalhe(codes.InvalidArgument, err.Error(), razaoSolicitacaoInvalida, nil)

	case errors.Is(err, shared.ErrSessaoNaoProvisionada):
		return comDetalhe(codes.FailedPrecondition, "sessão sem matriz de poltronas provisionada",
			razaoSessaoNaoProvisionada, nil)

	case errors.Is(err, shared.ErrPoltronaInexistente):
		return comDetalhe(codes.FailedPrecondition, "poltrona inexistente na sessão informada",
			razaoPoltronaInexistente, nil)

	case errors.Is(err, shared.ErrSessaoDesconhecida):
		return comDetalhe(codes.NotFound, "sessão desconhecida", razaoSessaoDesconhecida, nil)

	case errors.Is(err, shared.ErrDependenciaIndisponivel):
		// Detalhe interno fica de fora deliberadamente: o cliente só precisa
		// saber que não foi possível decidir agora.
		return comDetalhe(codes.Unavailable, "serviço temporariamente indisponível",
			razaoDependenciaIndisponivel, nil)

	default:
		return comDetalhe(codes.Internal, "erro interno", "ERRO_INTERNO", nil)
	}
}

func comDetalhe(codigo codes.Code, mensagem, razao string, metadados map[string]string) error {
	st := status.New(codigo, mensagem)
	comInfo, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason:   razao,
		Domain:   dominioErros,
		Metadata: metadados,
	})
	if err != nil {
		// Anexar detalhe nunca deve custar a resposta.
		return st.Err()
	}
	return comInfo.Err()
}
