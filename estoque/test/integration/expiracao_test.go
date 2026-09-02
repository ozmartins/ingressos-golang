//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func TestExpiracaoLiberaPoltronas(t *testing.T) {
	c := montarCenario(t, false)
	sessao, reservaID := reservaPendente(t, c)
	ctx := context.Background()

	c.Relogio.Avancar(9 * time.Minute)
	if _, err := c.Expirar.Varrer(ctx); err != nil {
		t.Fatalf("varredura: %v", err)
	}
	if got := c.statusReserva(t, reservaID); got != "PENDENTE" {
		t.Fatalf("reserva = %s antes do prazo", got)
	}

	c.Relogio.Avancar(2 * time.Minute)
	if _, err := c.Expirar.Varrer(ctx); err != nil {
		t.Fatalf("varredura: %v", err)
	}

	if got := c.statusReserva(t, reservaID); got != "EXPIRADA" {
		t.Errorf("reserva = %s, esperado EXPIRADA", got)
	}
	for _, rotulo := range []string{"A1", "A2"} {
		if got := c.statusPoltrona(t, sessao, rotulo); got != poltrona.Livre {
			t.Errorf("%s = %s, esperado LIVRE", rotulo, got)
		}
	}

	novo, err := c.Bloquear.Executar(ctx, sessao, "outra-pessoa", []string{"A1"})
	if err != nil || !novo.Concedido {
		t.Fatalf("poltrona expirada devia ser bloqueável: %v", err)
	}
}

func TestExpiracaoRecuperaReservasVencidasDuranteParada(t *testing.T) {
	c := montarCenario(t, false)
	sessao := c.novaSessao(t, []string{"A", "B"}, 5)
	ctx := context.Background()

	var reservas []string
	for _, rotulo := range []string{"A1", "A2", "A3", "B1", "B2"} {
		resultado, err := c.Bloquear.Executar(ctx, sessao, usuario, []string{rotulo})
		if err != nil || !resultado.Concedido {
			t.Fatalf("bloqueio de %s: %v", rotulo, err)
		}
		reservas = append(reservas, resultado.Reserva.ID)
	}

	c.Relogio.Avancar(15 * time.Minute)

	n, err := c.Expirar.Varrer(ctx)
	if err != nil {
		t.Fatalf("varredura: %v", err)
	}
	if n != len(reservas) {
		t.Fatalf("expiradas = %d, esperado %d", n, len(reservas))
	}
	for _, id := range reservas {
		if got := c.statusReserva(t, id); got != "EXPIRADA" {
			t.Errorf("reserva %s = %s", id, got)
		}
	}
	contagem := c.contarPorStatus(t, sessao)
	if contagem["LIVRE"] != 10 {
		t.Errorf("livres = %d, esperado 10", contagem["LIVRE"])
	}
}

func TestExpiracaoFuncionaSemRedis(t *testing.T) {
	c := montarCenario(t, false)
	sessao, reservaID := reservaPendente(t, c)

	c.Relogio.Avancar(11 * time.Minute)
	if _, err := c.Expirar.Varrer(context.Background()); err != nil {
		t.Fatalf("varredura: %v", err)
	}

	if got := c.statusReserva(t, reservaID); got != "EXPIRADA" {
		t.Errorf("reserva = %s, esperado EXPIRADA sem Redis", got)
	}
	if got := c.statusPoltrona(t, sessao, "A1"); got != poltrona.Livre {
		t.Errorf("A1 = %s, esperado LIVRE sem Redis", got)
	}
}

func TestExpiracaoComIndiceDePrazo(t *testing.T) {
	c := montarCenario(t, true)
	sessao, reservaID := reservaPendente(t, c)

	c.Relogio.Avancar(11 * time.Minute)
	res, err := c.Expirar.ExpirarUma(context.Background(), reservaID)
	if err != nil {
		t.Fatalf("expirar: %v", err)
	}
	if res != usecase.TransicaoAplicada {
		t.Fatalf("resultado = %s, esperado aplicada", res)
	}
	if got := c.statusPoltrona(t, sessao, "A1"); got != poltrona.Livre {
		t.Errorf("A1 = %s, esperado LIVRE", got)
	}
}

func TestCorridaEntreExpiracaoEConfirmacao(t *testing.T) {
	for i := 0; i < 20; i++ {
		c := montarCenario(t, false)
		sessao, reservaID := reservaPendente(t, c)
		ctx := context.Background()
		c.Relogio.Avancar(11 * time.Minute)

		pronto := make(chan struct{})
		resultados := make(chan string, 2)

		go func() {
			<-pronto
			res, err := c.Confirmar.Executar(ctx, filaSucesso, "m1", reservaID)
			if err != nil {
				resultados <- "erro:" + err.Error()
				return
			}
			resultados <- "confirmar:" + res.String()
		}()
		go func() {
			<-pronto
			res, err := c.Expirar.ExpirarUma(ctx, reservaID)
			if err != nil {
				resultados <- "erro:" + err.Error()
				return
			}
			resultados <- "expirar:" + res.String()
		}()

		close(pronto)
		<-resultados
		<-resultados

		status := c.statusReserva(t, reservaID)
		if status != "CONFIRMADA" && status != "EXPIRADA" {
			t.Fatalf("iteração %d: reserva terminou em %s", i, status)
		}
		esperado := poltrona.Ocupada
		if status == "EXPIRADA" {
			esperado = poltrona.Livre
		}
		for _, rotulo := range []string{"A1", "A2"} {
			if got := c.statusPoltrona(t, sessao, rotulo); got != esperado {
				t.Fatalf("iteração %d: reserva %s mas %s = %s", i, status, rotulo, got)
			}
		}
	}
}
