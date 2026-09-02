package contract

import (
	"context"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

func razaoDe(t *testing.T, err error) (codes.Code, string) {
	t.Helper()
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("erro não é status gRPC: %v", err)
	}
	for _, detalhe := range st.Details() {
		if info, ok := detalhe.(*errdetails.ErrorInfo); ok {
			return st.Code(), info.GetReason()
		}
	}
	return st.Code(), ""
}

func TestBloquearConcedeERecusaPorIndisponibilidade(t *testing.T) {
	cliente := servidorEmMemoria(t, novoEstoqueDeTeste())
	req := &pb.SolicitacaoBloqueio{
		SessaoId: sessaoProvisionada, PoltronasIds: []string{"A1", "A2"}, UsuarioId: usuario,
	}

	resp, err := cliente.BloquearPoltronas(context.Background(), req)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !resp.GetSucesso() {
		t.Fatal("esperava sucesso=true")
	}
	if resp.GetReservaId() == "" || resp.GetExpiraEm() == 0 {
		t.Errorf("resposta incompleta: reserva=%q expira_em=%d", resp.GetReservaId(), resp.GetExpiraEm())
	}
	if resp.GetMotivo() != pb.MotivoFalha_MOTIVO_NAO_INFORMADO {
		t.Errorf("motivo = %v em resposta de sucesso", resp.GetMotivo())
	}

	repetida, err := cliente.BloquearPoltronas(context.Background(), req)
	if err != nil {
		t.Fatalf("indisponibilidade não pode virar status de erro: %v", err)
	}
	if repetida.GetSucesso() {
		t.Fatal("esperava sucesso=false")
	}
	if repetida.GetMotivo() != pb.MotivoFalha_POLTRONAS_INDISPONIVEIS {
		t.Errorf("motivo = %v, esperado POLTRONAS_INDISPONIVEIS", repetida.GetMotivo())
	}
	if repetida.GetReservaId() != "" {
		t.Errorf("recusa não pode devolver reserva: %q", repetida.GetReservaId())
	}
}

func TestBloquearMapeiaCadaCategoriaDeErro(t *testing.T) {
	casos := []struct {
		nome   string
		req    *pb.SolicitacaoBloqueio
		codigo codes.Code
		razao  string
	}{
		{
			"lista vazia",
			&pb.SolicitacaoBloqueio{SessaoId: sessaoProvisionada, UsuarioId: usuario},
			codes.InvalidArgument, "SOLICITACAO_INVALIDA",
		},
		{
			"rótulos repetidos",
			&pb.SolicitacaoBloqueio{SessaoId: sessaoProvisionada, PoltronasIds: []string{"A1", "a1"}, UsuarioId: usuario},
			codes.InvalidArgument, "SOLICITACAO_INVALIDA",
		},
		{
			"identidade da pessoa ausente",
			&pb.SolicitacaoBloqueio{SessaoId: sessaoProvisionada, PoltronasIds: []string{"A1"}},
			codes.InvalidArgument, "SOLICITACAO_INVALIDA",
		},
		{
			"rótulo malformado",
			&pb.SolicitacaoBloqueio{SessaoId: sessaoProvisionada, PoltronasIds: []string{"poltrona-boa"}, UsuarioId: usuario},
			codes.InvalidArgument, "SOLICITACAO_INVALIDA",
		},
		{
			"acima do limite por bloqueio",
			&pb.SolicitacaoBloqueio{SessaoId: sessaoProvisionada, UsuarioId: usuario,
				PoltronasIds: []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10", "A11"}},
			codes.InvalidArgument, "LIMITE_POLTRONAS_EXCEDIDO",
		},
		{
			"poltrona inexistente na sessão",
			&pb.SolicitacaoBloqueio{SessaoId: sessaoProvisionada, PoltronasIds: []string{"Z9"}, UsuarioId: usuario},
			codes.FailedPrecondition, "POLTRONA_INEXISTENTE",
		},
		{
			"sessão sem matriz provisionada",
			&pb.SolicitacaoBloqueio{SessaoId: "sessao-desconhecida", PoltronasIds: []string{"A1"}, UsuarioId: usuario},
			codes.FailedPrecondition, "SESSAO_NAO_PROVISIONADA",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			estoque := novoEstoqueDeTeste()
			cliente := servidorEmMemoria(t, estoque)

			_, err := cliente.BloquearPoltronas(context.Background(), caso.req)
			if err == nil {
				t.Fatal("esperava erro")
			}
			codigo, razao := razaoDe(t, err)
			if codigo != caso.codigo {
				t.Errorf("código = %s, esperado %s", codigo, caso.codigo)
			}
			if razao != caso.razao {
				t.Errorf("reason = %q, esperado %q", razao, caso.razao)
			}
			for rotulo, status := range estoque.poltronas {
				if status != "LIVRE" {
					t.Errorf("recusa alterou %s para %s", rotulo, status)
				}
			}
		})
	}
}

func TestLimiteExcedidoInformaOLimiteVigente(t *testing.T) {
	cliente := servidorEmMemoria(t, novoEstoqueDeTeste())

	_, err := cliente.BloquearPoltronas(context.Background(), &pb.SolicitacaoBloqueio{
		SessaoId: sessaoProvisionada, UsuarioId: usuario,
		PoltronasIds: []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10", "A11"},
	})
	st, _ := status.FromError(err)
	for _, detalhe := range st.Details() {
		if info, ok := detalhe.(*errdetails.ErrorInfo); ok {
			if info.GetMetadata()["limite"] != "10" {
				t.Errorf("metadata[limite] = %q, esperado 10", info.GetMetadata()["limite"])
			}
			return
		}
	}
	t.Fatal("resposta sem ErrorInfo")
}

func TestDependenciaIndisponivelViraUnavailableSemVazarDetalhe(t *testing.T) {
	estoque := novoEstoqueDeTeste()
	estoque.erro = shared.ErrDependenciaIndisponivel
	cliente := servidorEmMemoria(t, estoque)

	_, err := cliente.BloquearPoltronas(context.Background(), &pb.SolicitacaoBloqueio{
		SessaoId: sessaoProvisionada, PoltronasIds: []string{"A1"}, UsuarioId: usuario,
	})
	codigo, razao := razaoDe(t, err)
	if codigo != codes.Unavailable {
		t.Errorf("código = %s, esperado Unavailable", codigo)
	}
	if razao != "DEPENDENCIA_INDISPONIVEL" {
		t.Errorf("reason = %q", razao)
	}

	st, _ := status.FromError(err)
	for _, proibido := range []string{"pgx", "SQL", "postgres", "5432", "goroutine"} {
		if contemIgnorandoCaixa(st.Message(), proibido) {
			t.Errorf("mensagem vazou detalhe interno (%q): %s", proibido, st.Message())
		}
	}
}

func TestIdentidadeDaPessoaNaoEhValidadaPeloEstoque(t *testing.T) {
	cliente := servidorEmMemoria(t, novoEstoqueDeTeste())

	resp, err := cliente.BloquearPoltronas(context.Background(), &pb.SolicitacaoBloqueio{
		SessaoId:     sessaoProvisionada,
		PoltronasIds: []string{"B1"},
		UsuarioId:    "identidade-arbitraria-sem-token",
	})
	if err != nil {
		t.Fatalf("o estoque não valida credencial de pessoa: %v", err)
	}
	if !resp.GetSucesso() {
		t.Fatal("esperava bloqueio concedido")
	}
}

func contemIgnorandoCaixa(texto, agulha string) bool {
	return len(agulha) > 0 && len(texto) >= len(agulha) &&
		stringContains(toLower(texto), toLower(agulha))
}

func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
