//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

const (
	filaSucesso = "estoque.pagamento-sucesso"
	filaFalhou  = "estoque.pagamento-falhou"
)

// reservaPendente cria uma sessão e bloqueia A1 e A2.
func reservaPendente(t *testing.T, c *Cenario) (sessao, reservaID string) {
	t.Helper()
	sessao = c.novaSessao(t, []string{"A"}, 5)
	resultado, err := c.Bloquear.Executar(context.Background(), sessao, usuario, []string{"A1", "A2"})
	if err != nil || !resultado.Concedido {
		t.Fatalf("bloqueio: %v", err)
	}
	return sessao, resultado.Reserva.ID
}

func TestConfirmacaoTornaPosseDefinitiva(t *testing.T) {
	c := montarCenario(t, false)
	sessao, reservaID := reservaPendente(t, c)

	res, err := c.Confirmar.Executar(context.Background(), filaSucesso, reservaID, reservaID)
	if err != nil || res != usecase.TransicaoAplicada {
		t.Fatalf("confirmação: %s %v", res, err)
	}

	if got := c.statusReserva(t, reservaID); got != "CONFIRMADA" {
		t.Errorf("reserva = %s, esperado CONFIRMADA", got)
	}
	for _, rotulo := range []string{"A1", "A2"} {
		if got := c.statusPoltrona(t, sessao, rotulo); got != poltrona.Ocupada {
			t.Errorf("%s = %s, esperado OCUPADA", rotulo, got)
		}
	}

	// FR-014: o prazo passa e as poltronas continuam ocupadas.
	c.Relogio.Avancar(time.Hour)
	if n, err := c.Expirar.Varrer(context.Background()); err != nil {
		t.Fatalf("varredura: %v", err)
	} else if n > 0 {
		if got := c.statusPoltrona(t, sessao, "A1"); got != poltrona.Ocupada {
			t.Fatalf("A1 = %s após varredura — reserva confirmada foi expirada", got)
		}
	}
}

func TestCancelamentoDevolvePoltronasEPermiteNovoBloqueio(t *testing.T) {
	c := montarCenario(t, false)
	sessao, reservaID := reservaPendente(t, c)
	ctx := context.Background()

	res, err := c.Cancelar.Executar(ctx, filaFalhou, reservaID, reservaID)
	if err != nil || res != usecase.TransicaoAplicada {
		t.Fatalf("cancelamento: %s %v", res, err)
	}
	if got := c.statusReserva(t, reservaID); got != "CANCELADA" {
		t.Errorf("reserva = %s, esperado CANCELADA", got)
	}
	for _, rotulo := range []string{"A1", "A2"} {
		if got := c.statusPoltrona(t, sessao, rotulo); got != poltrona.Livre {
			t.Errorf("%s = %s, esperado LIVRE", rotulo, got)
		}
	}

	novo, err := c.Bloquear.Executar(ctx, sessao, "outra-pessoa", []string{"A1", "A2"})
	if err != nil || !novo.Concedido {
		t.Fatalf("poltronas liberadas deviam ser bloqueáveis: %v", err)
	}
}

// TestReentregaProduzMesmoEstadoFinal cobre SC-004.
func TestReentregaProduzMesmoEstadoFinal(t *testing.T) {
	c := montarCenario(t, false)
	sessao, reservaID := reservaPendente(t, c)
	ctx := context.Background()

	if _, err := c.Confirmar.Executar(ctx, filaSucesso, reservaID, reservaID); err != nil {
		t.Fatalf("confirmação: %v", err)
	}

	for i := 0; i < 3; i++ {
		res, err := c.Confirmar.Executar(ctx, filaSucesso, reservaID, reservaID)
		if err != nil {
			t.Fatalf("reentrega %d não pode virar erro: %v", i, err)
		}
		if res != usecase.TransicaoIgnoradaDuplicata {
			t.Errorf("reentrega %d devolveu %s, esperado ignorada-duplicata", i, res)
		}
	}

	contagem := c.contarPorStatus(t, sessao)
	if contagem["OCUPADA"] != 2 {
		t.Errorf("ocupadas = %d, esperado 2 — reentrega alterou o estado", contagem["OCUPADA"])
	}
}

// TestPrimeiroDesfechoPrevalece cobre a chegada fora de ordem (FR-022).
func TestPrimeiroDesfechoPrevalece(t *testing.T) {
	t.Run("recusa depois de aprovada", func(t *testing.T) {
		c := montarCenario(t, false)
		sessao, reservaID := reservaPendente(t, c)
		ctx := context.Background()

		if _, err := c.Confirmar.Executar(ctx, filaSucesso, "m1", reservaID); err != nil {
			t.Fatal(err)
		}
		res, err := c.Cancelar.Executar(ctx, filaFalhou, "m2", reservaID)
		if err != nil {
			t.Fatal(err)
		}
		if res != usecase.TransicaoIgnoradaEstadoFinal {
			t.Errorf("resultado = %s, esperado ignorada-estado-final", res)
		}
		if got := c.statusPoltrona(t, sessao, "A1"); got != poltrona.Ocupada {
			t.Errorf("A1 = %s, esperado OCUPADA", got)
		}
	})

	t.Run("aprovação depois de cancelada", func(t *testing.T) {
		c := montarCenario(t, false)
		sessao, reservaID := reservaPendente(t, c)
		ctx := context.Background()

		if _, err := c.Cancelar.Executar(ctx, filaFalhou, "m1", reservaID); err != nil {
			t.Fatal(err)
		}
		res, err := c.Confirmar.Executar(ctx, filaSucesso, "m2", reservaID)
		if err != nil {
			t.Fatal(err)
		}
		if res != usecase.TransicaoIgnoradaEstadoFinal {
			t.Errorf("resultado = %s, esperado ignorada-estado-final", res)
		}
		// A poltrona já pode ter sido retomada por outra pessoa; a aprovação
		// tardia não pode sobrescrever o estado atual.
		if got := c.statusPoltrona(t, sessao, "A1"); got != poltrona.Livre {
			t.Errorf("A1 = %s, esperado LIVRE", got)
		}
	})
}

func TestDesfechoParaReservaDesconhecidaEhIgnorado(t *testing.T) {
	c := montarCenario(t, false)

	res, err := c.Confirmar.Executar(context.Background(), filaSucesso, "m-desconhecida", "reserva-que-nao-existe")
	if err != nil {
		t.Fatalf("reserva desconhecida não pode virar erro: %v", err)
	}
	if res != usecase.TransicaoIgnoradaInexistente {
		t.Errorf("resultado = %s, esperado ignorada-inexistente", res)
	}
}
