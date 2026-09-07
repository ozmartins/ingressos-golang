package estoque

import (
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// As razões que o estoque declara no `ErrorInfo` do seu contrato de erros
// (specs/001-estoque-bloqueio-poltronas/contracts/erros.md). Elas são parte do
// contrato e só mudam com versão nova; a mensagem humana, não — por isso a
// tradução olha a razão, e nunca o texto.
const (
	razaoSolicitacaoInvalida   = "SOLICITACAO_INVALIDA"
	razaoLimiteExcedido        = "LIMITE_POLTRONAS_EXCEDIDO"
	razaoSessaoNaoProvisionada = "SESSAO_NAO_PROVISIONADA"
	razaoPoltronaInexistente   = "POLTRONA_INEXISTENTE"
)

// traduzirErroDeChamada converte o erro gRPC do estoque num erro de domínio
// deste serviço. O que ela separa é quem errou: uma recusa que o estoque decidiu
// é erro de quem chamou, e não pode chegar ao cliente como "estoque
// indisponível".
//
// A categoria é o código de status; a razão refina dentro dele. Um código que o
// contrato não prevê cai em indisponibilidade, que é o desfecho conservador.
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
		// UNAVAILABLE, DEADLINE_EXCEEDED, UNAUTHENTICATED e o que mais vier:
		// falha de canal ou do parceiro, sobre a qual repetir faz sentido.
		return fmt.Errorf("%w: %v", shared.ErrEstoqueIndisponivel, err)
	}
}

// recusaDoEstoque distingue "o estoque decidiu não" de "o estoque falhou". O
// circuito de recusa rápida existe para proteger contra parceiro doente, e uma
// entrada inválida repetida não é doença do parceiro: contá-la abriria o
// circuito e derrubaria a reserva para todo mundo.
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
