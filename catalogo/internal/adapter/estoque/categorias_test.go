package estoque

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func TestTraduzirCadaCategoriaDoContratoDoEstoque(t *testing.T) {
	casos := map[string]struct {
		erro     error
		esperado error
		recusa   bool
	}{
		"solicitação inválida": {
			comRazao(codes.InvalidArgument, "rótulo fora do formato", razaoSolicitacaoInvalida, nil),
			shared.ErrSolicitacaoRecusadaPeloEstoque, true,
		},
		"limite de poltronas excedido": {
			comRazao(codes.InvalidArgument, "acima do limite", razaoLimiteExcedido,
				map[string]string{"limite": "10"}),
			shared.ErrSolicitacaoRecusadaPeloEstoque, true,
		},
		"poltrona inexistente": {
			comRazao(codes.FailedPrecondition, "poltrona Z9 não existe", razaoPoltronaInexistente, nil),
			shared.ErrPoltronaInexistente, true,
		},
		"sessão sem matriz provisionada": {
			comRazao(codes.FailedPrecondition, "sessão não provisionada", razaoSessaoNaoProvisionada, nil),
			shared.ErrSessaoSemPoltronas, true,
		},
		"dependência do estoque indisponível": {
			comRazao(codes.Unavailable, "banco fora", "DEPENDENCIA_INDISPONIVEL", nil),
			shared.ErrEstoqueIndisponivel, false,
		},
		"prazo estourado": {
			status.Error(codes.DeadlineExceeded, "orçamento estourado"),
			shared.ErrEstoqueIndisponivel, false,
		},
		"defeito do estoque": {
			comRazao(codes.Internal, "erro interno", "ERRO_INTERNO", nil),
			shared.ErrEstoqueComDefeito, false,
		},
		"handshake recusado": {
			status.Error(codes.Unauthenticated, "sem identidade de serviço"),
			shared.ErrEstoqueIndisponivel, false,
		},
		"código não previsto": {
			status.Error(codes.ResourceExhausted, "algo novo"),
			shared.ErrEstoqueIndisponivel, false,
		},
		"erro que não é status gRPC": {
			errors.New("conexão caiu no meio"),
			shared.ErrEstoqueIndisponivel, false,
		},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			if got := traduzirErroDeChamada(caso.erro); !errors.Is(got, caso.esperado) {
				t.Fatalf("erro = %v, esperado que envolvesse %v", got, caso.esperado)
			}
			if got := recusaDoEstoque(caso.erro); got != caso.recusa {
				t.Fatalf("recusaDoEstoque = %v, esperado %v", got, caso.recusa)
			}
		})
	}
}

func TestLimiteExcedidoRepassaOLimiteVigente(t *testing.T) {
	err := traduzirErroDeChamada(comRazao(codes.InvalidArgument, "acima do limite",
		razaoLimiteExcedido, map[string]string{"limite": "10"}))

	if !strings.Contains(err.Error(), "10") {
		t.Fatalf("a mensagem deveria dizer o limite: %v", err)
	}
}

func TestLimiteExcedidoSemMetadadoNaoInventaNumero(t *testing.T) {
	err := traduzirErroDeChamada(comRazao(codes.InvalidArgument, "acima do limite",
		razaoLimiteExcedido, nil))

	if !errors.Is(err, shared.ErrSolicitacaoRecusadaPeloEstoque) {
		t.Fatalf("erro = %v", err)
	}
	if strings.Contains(err.Error(), "no máximo") {
		t.Fatalf("sem metadado não há limite a informar: %v", err)
	}
}

func comRazao(codigo codes.Code, mensagem, razao string, metadados map[string]string) error {
	st := status.New(codigo, mensagem)
	comInfo, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: razao, Domain: "estoque.ingressos", Metadata: metadados,
	})
	if err != nil {
		return st.Err()
	}
	return comInfo.Err()
}
