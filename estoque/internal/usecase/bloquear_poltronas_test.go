package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const sessao = "f781a9b2-11e2-4f81-a901-8890bc123456"
const usuario = "c394c8b3-76a1-4328-b803-02f5923b7a15"

var agora = time.Date(2026, 8, 29, 21, 33, 0, 0, time.UTC)

func montarBloqueio(e *estoqueFalso, p *prazoFalso, l *logFalso) BloquearPoltronas {
	return BloquearPoltronas{
		Reservas: e,
		Prazo:    p,
		Relogio:  shared.NovoRelogioFixo(agora),
		Log:      l,
		TTL:      10 * time.Minute,
		Limite:   10,
		TraceContextDe: func(context.Context) map[string]string {
			return map[string]string{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}
		},
	}
}

func TestBloqueioConcedido(t *testing.T) {
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1", "A2", "A3")
	prazo, log := novoPrazoFalso(), &logFalso{}

	resultado, err := montarBloqueio(estoque, prazo, log).Executar(context.Background(), sessao, usuario, []string{"A1", "A2"})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !resultado.Concedido {
		t.Fatal("esperava bloqueio concedido")
	}
	if !resultado.Reserva.ExpiraEm.Equal(agora.Add(10 * time.Minute)) {
		t.Errorf("expira_em = %v, esperado agora + 10min", resultado.Reserva.ExpiraEm)
	}
	for _, rotulo := range []string{"A1", "A2"} {
		if got := estoque.statusDe(sessao, rotulo); got != poltrona.Reservada {
			t.Errorf("%s = %s, esperado RESERVADA", rotulo, got)
		}
	}
	if got := estoque.statusDe(sessao, "A3"); got != poltrona.Livre {
		t.Errorf("A3 não foi solicitada e ficou %s", got)
	}
	if _, marcado := prazo.marcados[resultado.Reserva.ID]; !marcado {
		t.Error("índice de prazo não foi marcado")
	}
}

func TestBloqueioPublicaFatoComRotulosEContexto(t *testing.T) {
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1", "A2")

	resultado, err := montarBloqueio(estoque, novoPrazoFalso(), &logFalso{}).
		Executar(context.Background(), sessao, usuario, []string{"a2", "A1"})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(estoque.fatos) != 1 {
		t.Fatalf("esperava 1 fato na caixa de saída, veio %d", len(estoque.fatos))
	}

	fato := estoque.fatos[0]
	if fato.MessageID != resultado.Reserva.ID {
		t.Errorf("message_id = %q, esperado o id da reserva", fato.MessageID)
	}
	if fato.RoutingKey != RoutingKeyReservaCriada {
		t.Errorf("routing key = %q, esperado %q", fato.RoutingKey, RoutingKeyReservaCriada)
	}
	if fato.TraceContext["traceparent"] == "" {
		t.Error("fato sem contexto de rastreamento")
	}

	var evento EventoReservaCriada
	if err := json.Unmarshal(fato.Payload, &evento); err != nil {
		t.Fatalf("payload não é JSON válido: %v", err)
	}
	if evento.Evento != "RESERVA_CRIADA" || evento.Versao != 1 {
		t.Errorf("envelope = %s v%d", evento.Evento, evento.Versao)
	}
	if len(evento.PoltronasIDs) != 2 || evento.PoltronasIDs[0] != "A2" || evento.PoltronasIDs[1] != "A1" {
		t.Errorf("poltronas_ids = %v, esperado [A2 A1]", evento.PoltronasIDs)
	}
}

func TestBloqueioRecusadoPorIndisponibilidadeNaoEhErro(t *testing.T) {
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1", "A2")
	uc := montarBloqueio(estoque, novoPrazoFalso(), &logFalso{})

	if _, err := uc.Executar(context.Background(), sessao, usuario, []string{"A1"}); err != nil {
		t.Fatalf("primeiro bloqueio devia passar: %v", err)
	}

	resultado, err := uc.Executar(context.Background(), sessao, "outra-pessoa", []string{"A1", "A2"})
	if err != nil {
		t.Fatalf("indisponibilidade é desfecho de negócio, não erro: %v", err)
	}
	if resultado.Concedido {
		t.Fatal("esperava recusa por indisponibilidade")
	}
	if got := estoque.statusDe(sessao, "A2"); got != poltrona.Livre {
		t.Errorf("A2 = %s, esperado LIVRE — bloqueio recusado não altera estado", got)
	}
}

func TestBloqueioRecusaSolicitacaoInvalida(t *testing.T) {
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1")
	uc := montarBloqueio(estoque, novoPrazoFalso(), &logFalso{})

	casos := map[string]struct {
		usuario string
		rotulos []string
		erro    error
	}{
		"lista vazia":     {usuario, nil, shared.ErrSolicitacaoInvalida},
		"rótulo repetido": {usuario, []string{"A1", "A1"}, shared.ErrSolicitacaoInvalida},
		"usuário ausente": {"", []string{"A1"}, shared.ErrSolicitacaoInvalida},
		"acima do limite": {usuario, []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10", "A11"}, shared.ErrLimiteExcedido},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := uc.Executar(context.Background(), sessao, caso.usuario, caso.rotulos)
			if !errors.Is(err, caso.erro) {
				t.Fatalf("erro = %v, esperado %v", err, caso.erro)
			}
			if len(estoque.reservas) != 0 {
				t.Errorf("solicitação inválida criou %d reserva(s)", len(estoque.reservas))
			}
			if got := estoque.statusDe(sessao, "A1"); got != poltrona.Livre {
				t.Errorf("A1 = %s, esperado LIVRE", got)
			}
		})
	}
}

func TestBloqueioRecusaSessaoNaoProvisionada(t *testing.T) {
	uc := montarBloqueio(novoEstoqueFalso(), novoPrazoFalso(), &logFalso{})

	_, err := uc.Executar(context.Background(), "sessao-sem-matriz", usuario, []string{"A1"})
	if !errors.Is(err, shared.ErrSessaoNaoProvisionada) {
		t.Fatalf("erro = %v, esperado ErrSessaoNaoProvisionada", err)
	}
}

func TestBloqueioRecusaRotuloInexistente(t *testing.T) {
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1")

	_, err := montarBloqueio(estoque, novoPrazoFalso(), &logFalso{}).
		Executar(context.Background(), sessao, usuario, []string{"Z9"})
	if !errors.Is(err, shared.ErrPoltronaInexistente) {
		t.Fatalf("erro = %v, esperado ErrPoltronaInexistente", err)
	}
}

func TestBloqueioPropagaFalhaDoRepositorio(t *testing.T) {
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1")
	estoque.erroForcado = shared.ErrDependenciaIndisponivel

	_, err := montarBloqueio(estoque, novoPrazoFalso(), &logFalso{}).
		Executar(context.Background(), sessao, usuario, []string{"A1"})
	if !errors.Is(err, shared.ErrDependenciaIndisponivel) {
		t.Fatalf("erro = %v, esperado ErrDependenciaIndisponivel", err)
	}
}

func TestBloqueioSobreviveAoIndiceDePrazoIndisponivel(t *testing.T) {
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1")
	prazo := novoPrazoFalso()
	prazo.erro = errors.New("redis fora do ar")
	log := &logFalso{}

	resultado, err := montarBloqueio(estoque, prazo, log).Executar(context.Background(), sessao, usuario, []string{"A1"})
	if err != nil {
		t.Fatalf("falha do índice de prazo não pode derrubar o bloqueio: %v", err)
	}
	if !resultado.Concedido {
		t.Fatal("esperava bloqueio concedido mesmo sem índice de prazo")
	}
	if !log.avisou() {
		t.Error("degradação do índice de prazo devia ser registrada")
	}
}
