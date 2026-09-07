package usecase

import (
	"context"
	"testing"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const filaCancelamento = "estoque.sessao-cancelada"

func montarCancelarSessao(e *estoqueFalso, p *prazoFalso, l *logFalso) CancelarSessao {
	return CancelarSessao{
		Reservas: e, Prazo: p, Relogio: shared.NovoRelogioFixo(agora), Log: l,
	}
}

// A sessão saiu da grade: as reservas pendentes precisam soltar as poltronas, e o
// prazo de cada uma precisa sair do índice — senão a varredura de expiração fica
// perseguindo reservas de uma sessão que não existe mais.
func TestCancelarSessaoSoltaAsPendentes(t *testing.T) {
	estoque, prazo, log := novoEstoqueFalso(), novoPrazoFalso(), &logFalso{}
	estoque.provisionar(sessao, "A1", "A2", "A3")

	bloqueio := montarBloqueio(estoque, prazo, log)
	primeira, err := bloqueio.Executar(context.Background(), sessao, usuario, []string{"A1"}, valorDeTeste)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := bloqueio.Executar(context.Background(), sessao, "outra-pessoa", []string{"A2"}, valorDeTeste)
	if err != nil {
		t.Fatal(err)
	}

	uc := montarCancelarSessao(estoque, prazo, log)
	resultado, err := uc.Executar(context.Background(), filaCancelamento, "msg-1", sessao)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if resultado != TransicaoAplicada {
		t.Fatalf("resultado = %v, esperava TransicaoAplicada", resultado)
	}

	for _, id := range []string{primeira.Reserva.ID, segunda.Reserva.ID} {
		if got := estoque.statusReserva(id); got != reserva.Cancelada {
			t.Errorf("reserva %s = %s, esperava CANCELADA", id, got)
		}
		if !prazo.liberados[id] {
			t.Errorf("o prazo da reserva %s deveria ter saído do índice", id)
		}
	}
	for _, rotulo := range []string{"A1", "A2", "A3"} {
		if got := estoque.statusDe(sessao, rotulo); got != poltrona.Livre {
			t.Errorf("poltrona %s = %s, esperava LIVRE", rotulo, got)
		}
	}
}

// Uma reserva confirmada é um ingresso pago: cancelar a sessão não a apaga, e não
// devolve a poltrona ao estoque. O que o caso de uso faz é contar, para que
// alguém saiba que existe algo a resolver fora do sistema.
func TestCancelarSessaoPreservaAsConfirmadas(t *testing.T) {
	estoque, prazo, log := novoEstoqueFalso(), novoPrazoFalso(), &logFalso{}
	estoque.provisionar(sessao, "A1", "A2")

	bloqueio := montarBloqueio(estoque, prazo, log)
	paga, err := bloqueio.Executar(context.Background(), sessao, usuario, []string{"A1"}, valorDeTeste)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := estoque.Confirmar(context.Background(), "", "", paga.Reserva.ID, agora); err != nil {
		t.Fatal(err)
	}
	pendente, err := bloqueio.Executar(context.Background(), sessao, "outra-pessoa", []string{"A2"}, valorDeTeste)
	if err != nil {
		t.Fatal(err)
	}

	uc := montarCancelarSessao(estoque, prazo, log)
	if _, err := uc.Executar(context.Background(), filaCancelamento, "msg-1", sessao); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if got := estoque.statusReserva(paga.Reserva.ID); got != reserva.Confirmada {
		t.Fatalf("a reserva paga ficou %s; ingresso vendido não se apaga", got)
	}
	if got := estoque.statusDe(sessao, "A1"); got != poltrona.Ocupada {
		t.Fatalf("a poltrona paga ficou %s, esperava OCUPADA", got)
	}
	if got := estoque.statusReserva(pendente.Reserva.ID); got != reserva.Cancelada {
		t.Fatalf("a pendente ficou %s, esperava CANCELADA", got)
	}
}

func TestCancelarSessaoEhIdempotente(t *testing.T) {
	estoque, prazo, log := novoEstoqueFalso(), novoPrazoFalso(), &logFalso{}
	estoque.provisionar(sessao, "A1")
	if _, err := montarBloqueio(estoque, prazo, log).
		Executar(context.Background(), sessao, usuario, []string{"A1"}, valorDeTeste); err != nil {
		t.Fatal(err)
	}

	uc := montarCancelarSessao(estoque, prazo, log)
	if r, err := uc.Executar(context.Background(), filaCancelamento, "msg-1", sessao); err != nil || r != TransicaoAplicada {
		t.Fatalf("primeira entrega: %v / %v", r, err)
	}
	r, err := uc.Executar(context.Background(), filaCancelamento, "msg-1", sessao)
	if err != nil {
		t.Fatalf("erro inesperado na reentrega: %v", err)
	}
	if r != TransicaoIgnoradaDuplicata {
		t.Fatalf("resultado = %v, esperava TransicaoIgnoradaDuplicata", r)
	}
}

// Sessão que este serviço nunca viu, ou sem reserva alguma: não há o que soltar,
// e o cancelamento não tem por que falhar por isso — a mensagem é confirmada.
func TestCancelarSessaoSemReservasNaoFalha(t *testing.T) {
	estoque, prazo, log := novoEstoqueFalso(), novoPrazoFalso(), &logFalso{}
	uc := montarCancelarSessao(estoque, prazo, log)

	r, err := uc.Executar(context.Background(), filaCancelamento, "msg-1", "sessao-desconhecida")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r != TransicaoIgnoradaInexistente {
		t.Fatalf("resultado = %v, esperava TransicaoIgnoradaInexistente", r)
	}
}

func TestCancelarSessaoDevolveErroDeInfra(t *testing.T) {
	estoque, prazo, log := novoEstoqueFalso(), novoPrazoFalso(), &logFalso{}
	estoque.erroForcado = shared.ErrDependenciaIndisponivel
	uc := montarCancelarSessao(estoque, prazo, log)

	if _, err := uc.Executar(context.Background(), filaCancelamento, "msg-1", sessao); err == nil {
		t.Fatal("esperava que a falha de infraestrutura subisse, para a mensagem voltar à fila")
	}
}
