package contract

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/estoque"
)

func TestConsultarMapaDevolveEstadoCorrente(t *testing.T) {
	estoque := novoEstoqueDeTeste()
	cliente := servidorEmMemoria(t, estoque)

	resp, err := cliente.ConsultarMapaPoltronas(context.Background(),
		&pb.SolicitacaoMapa{SessaoId: sessaoProvisionada})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(resp.GetPoltronas()) != len(estoque.poltronas) {
		t.Fatalf("poltronas = %d, esperado %d", len(resp.GetPoltronas()), len(estoque.poltronas))
	}
	for _, p := range resp.GetPoltronas() {
		if p.GetRotulo() == "" || p.GetFileira() == "" || p.GetNumero() == 0 {
			t.Errorf("poltrona incompleta: %+v", p)
		}
		if p.GetStatus() != pb.StatusPoltrona_LIVRE {
			t.Errorf("%s = %v, esperado LIVRE", p.GetRotulo(), p.GetStatus())
		}
	}

	// Depois de um bloqueio, a consulta reflete o estado corrente (FR-030).
	if _, err := cliente.BloquearPoltronas(context.Background(), &pb.SolicitacaoBloqueio{
		SessaoId: sessaoProvisionada, PoltronasIds: []string{"A1"}, UsuarioId: usuario,
	}); err != nil {
		t.Fatalf("bloqueio: %v", err)
	}

	depois, err := cliente.ConsultarMapaPoltronas(context.Background(),
		&pb.SolicitacaoMapa{SessaoId: sessaoProvisionada})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	for _, p := range depois.GetPoltronas() {
		if p.GetRotulo() == "A1" && p.GetStatus() != pb.StatusPoltrona_RESERVADA {
			t.Errorf("A1 = %v, esperado RESERVADA", p.GetStatus())
		}
	}
}

func TestConsultarMapaDeSessaoDesconhecida(t *testing.T) {
	cliente := servidorEmMemoria(t, novoEstoqueDeTeste())

	_, err := cliente.ConsultarMapaPoltronas(context.Background(),
		&pb.SolicitacaoMapa{SessaoId: "sessao-que-nao-existe"})
	if err == nil {
		t.Fatal("esperava erro")
	}
	codigo, razao := razaoDe(t, err)
	if codigo != codes.NotFound {
		t.Errorf("código = %s, esperado NotFound", codigo)
	}
	if razao != "SESSAO_DESCONHECIDA" {
		t.Errorf("reason = %q, esperado SESSAO_DESCONHECIDA", razao)
	}
}

func TestConsultarMapaSemSessao(t *testing.T) {
	cliente := servidorEmMemoria(t, novoEstoqueDeTeste())

	_, err := cliente.ConsultarMapaPoltronas(context.Background(), &pb.SolicitacaoMapa{})
	codigo, _ := razaoDe(t, err)
	if codigo != codes.InvalidArgument {
		t.Errorf("código = %s, esperado InvalidArgument", codigo)
	}
}
