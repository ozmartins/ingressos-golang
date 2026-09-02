package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const filaSucesso = "estoque.pagamento-sucesso"
const filaFalhou = "estoque.pagamento-falhou"

func cenario(t *testing.T) (*estoqueFalso, *prazoFalso, *logFalso, string) {
	t.Helper()
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1", "A2", "A3")
	prazo, log := novoPrazoFalso(), &logFalso{}

	resultado, err := montarBloqueio(estoque, prazo, log).
		Executar(context.Background(), sessao, usuario, []string{"A1", "A2"})
	if err != nil || !resultado.Concedido {
		t.Fatalf("cenário exige bloqueio concedido: %v", err)
	}
	return estoque, prazo, log, resultado.Reserva.ID
}

func confirmador(e *estoqueFalso, p *prazoFalso, l *logFalso) ConfirmarReserva {
	return ConfirmarReserva{Reservas: e, Prazo: p, Relogio: shared.NovoRelogioFixo(agora), Log: l}
}

func cancelador(e *estoqueFalso, p *prazoFalso, l *logFalso) CancelarReserva {
	return CancelarReserva{Reservas: e, Prazo: p, Relogio: shared.NovoRelogioFixo(agora), Log: l}
}

func TestConfirmarTornaPosseDefinitiva(t *testing.T) {
	estoque, prazo, log, reservaID := cenario(t)

	res, err := confirmador(estoque, prazo, log).Executar(context.Background(), filaSucesso, reservaID, reservaID)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if res != TransicaoAplicada {
		t.Fatalf("resultado = %s, esperado aplicada", res)
	}
	if got := estoque.statusReserva(reservaID); got != reserva.Confirmada {
		t.Errorf("reserva = %s, esperado CONFIRMADA", got)
	}
	for _, rotulo := range []string{"A1", "A2"} {
		if got := estoque.statusDe(sessao, rotulo); got != poltrona.Ocupada {
			t.Errorf("%s = %s, esperado OCUPADA", rotulo, got)
		}
	}
	if !prazo.liberados[reservaID] {
		t.Error("índice de prazo devia ser liberado após a confirmação")
	}
}

func TestConfirmarEhIdempotente(t *testing.T) {
	estoque, prazo, log, reservaID := cenario(t)
	uc := confirmador(estoque, prazo, log)

	if _, err := uc.Executar(context.Background(), filaSucesso, reservaID, reservaID); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	res, err := uc.Executar(context.Background(), filaSucesso, reservaID, reservaID)
	if err != nil {
		t.Fatalf("reentrega não pode virar erro: %v", err)
	}
	if res != TransicaoIgnoradaDuplicata {
		t.Errorf("resultado = %s, esperado ignorada-duplicata", res)
	}
	if got := estoque.statusReserva(reservaID); got != reserva.Confirmada {
		t.Errorf("reserva = %s, esperado CONFIRMADA", got)
	}
}

func TestConfirmarReservaDesconhecidaEhIgnoradaEAuditada(t *testing.T) {
	estoque, prazo, log, _ := cenario(t)

	res, err := confirmador(estoque, prazo, log).
		Executar(context.Background(), filaSucesso, "msg-1", "reserva-que-nao-existe")
	if err != nil {
		t.Fatalf("reserva desconhecida não pode virar erro: %v", err)
	}
	if res != TransicaoIgnoradaInexistente {
		t.Errorf("resultado = %s, esperado ignorada-inexistente", res)
	}
	if !log.avisou() {
		t.Error("divergência devia ser registrada de forma auditável (FR-022)")
	}
}

func TestCancelarDevolvePoltronasAoEstoque(t *testing.T) {
	estoque, prazo, log, reservaID := cenario(t)

	res, err := cancelador(estoque, prazo, log).Executar(context.Background(), filaFalhou, reservaID, reservaID)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if res != TransicaoAplicada {
		t.Fatalf("resultado = %s, esperado aplicada", res)
	}
	if got := estoque.statusReserva(reservaID); got != reserva.Cancelada {
		t.Errorf("reserva = %s, esperado CANCELADA", got)
	}
	for _, rotulo := range []string{"A1", "A2"} {
		if got := estoque.statusDe(sessao, rotulo); got != poltrona.Livre {
			t.Errorf("%s = %s, esperado LIVRE", rotulo, got)
		}
	}
	if !prazo.liberados[reservaID] {
		t.Error("índice de prazo devia ser liberado após o cancelamento")
	}

	novo, err := montarBloqueio(estoque, prazo, log).
		Executar(context.Background(), sessao, "outra-pessoa", []string{"A1"})
	if err != nil || !novo.Concedido {
		t.Fatalf("poltrona liberada devia ser bloqueável de novo: %v", err)
	}
}

func TestPrimeiroDesfechoPrevalece(t *testing.T) {
	t.Run("confirmada depois recusada", func(t *testing.T) {
		estoque, prazo, log, reservaID := cenario(t)

		if _, err := confirmador(estoque, prazo, log).Executar(context.Background(), filaSucesso, "m1", reservaID); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		res, err := cancelador(estoque, prazo, log).Executar(context.Background(), filaFalhou, "m2", reservaID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if res != TransicaoIgnoradaEstadoFinal {
			t.Errorf("resultado = %s, esperado ignorada-estado-final", res)
		}
		if got := estoque.statusDe(sessao, "A1"); got != poltrona.Ocupada {
			t.Errorf("A1 = %s, esperado OCUPADA", got)
		}
		if !log.avisou() {
			t.Error("divergência devia ser registrada")
		}
	})

	t.Run("cancelada depois aprovada", func(t *testing.T) {
		estoque, prazo, log, reservaID := cenario(t)

		if _, err := cancelador(estoque, prazo, log).Executar(context.Background(), filaFalhou, "m1", reservaID); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		res, err := confirmador(estoque, prazo, log).Executar(context.Background(), filaSucesso, "m2", reservaID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if res != TransicaoIgnoradaEstadoFinal {
			t.Errorf("resultado = %s, esperado ignorada-estado-final", res)
		}
		if got := estoque.statusDe(sessao, "A1"); got != poltrona.Livre {
			t.Errorf("A1 = %s, esperado LIVRE — aprovação tardia não pode retomar poltrona já liberada", got)
		}
	})
}

func TestLerDesfechoPagamento(t *testing.T) {
	valido := []byte(`{"evento":"PAGAMENTO_SUCESSO","versao":1,"reserva_id":"r-1"}`)
	d, err := LerDesfechoPagamento(valido)
	if err != nil {
		t.Fatalf("corpo válido recusado: %v", err)
	}
	if d.ReservaID != "r-1" {
		t.Errorf("reserva_id = %q", d.ReservaID)
	}

	for nome, corpo := range map[string][]byte{
		"json inválido":      []byte(`{{{`),
		"reserva_id ausente": []byte(`{"evento":"PAGAMENTO_SUCESSO"}`),
	} {
		if _, err := LerDesfechoPagamento(corpo); err == nil {
			t.Errorf("%s devia ser recusado", nome)
		}
	}
}

func TestExpirarLiberaReservasVencidas(t *testing.T) {
	estoque, prazo, log, reservaID := cenario(t)
	relogio := shared.NovoRelogioFixo(agora)
	uc := ExpirarReservas{Reservas: estoque, Prazo: prazo, Relogio: relogio, Log: log, LotePorVarredura: 100}

	relogio.Avancar(9 * time.Minute)
	if n, err := uc.Varrer(context.Background()); err != nil || n != 0 {
		t.Fatalf("varredura antes do prazo: n=%d err=%v", n, err)
	}

	relogio.Avancar(2 * time.Minute)
	n, err := uc.Varrer(context.Background())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if n != 1 {
		t.Fatalf("expiradas = %d, esperado 1", n)
	}
	if got := estoque.statusReserva(reservaID); got != reserva.Expirada {
		t.Errorf("reserva = %s, esperado EXPIRADA", got)
	}
	for _, rotulo := range []string{"A1", "A2"} {
		if got := estoque.statusDe(sessao, rotulo); got != poltrona.Livre {
			t.Errorf("%s = %s, esperado LIVRE", rotulo, got)
		}
	}
	if !prazo.liberados[reservaID] {
		t.Error("índice de prazo devia ser liberado após a expiração")
	}
}

func TestReservaConfirmadaNuncaExpira(t *testing.T) {
	estoque, prazo, log, reservaID := cenario(t)
	relogio := shared.NovoRelogioFixo(agora)

	if _, err := (ConfirmarReserva{Reservas: estoque, Prazo: prazo, Relogio: relogio, Log: log}).
		Executar(context.Background(), filaSucesso, "m1", reservaID); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	relogio.Avancar(24 * time.Hour)
	uc := ExpirarReservas{Reservas: estoque, Prazo: prazo, Relogio: relogio, Log: log}
	if n, err := uc.Varrer(context.Background()); err != nil || n != 0 {
		t.Fatalf("reserva confirmada não pode expirar: n=%d err=%v", n, err)
	}
	if got := estoque.statusDe(sessao, "A1"); got != poltrona.Ocupada {
		t.Errorf("A1 = %s, esperado OCUPADA", got)
	}
}

func TestExpirarUmaEhIdempotente(t *testing.T) {
	estoque, prazo, log, reservaID := cenario(t)
	relogio := shared.NovoRelogioFixo(agora.Add(11 * time.Minute))
	uc := ExpirarReservas{Reservas: estoque, Prazo: prazo, Relogio: relogio, Log: log}

	primeira, err := uc.ExpirarUma(context.Background(), reservaID)
	if err != nil || primeira != TransicaoAplicada {
		t.Fatalf("primeira expiração: %s %v", primeira, err)
	}
	segunda, err := uc.ExpirarUma(context.Background(), reservaID)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if segunda == TransicaoAplicada {
		t.Error("segunda expiração não podia ser aplicada")
	}
}
