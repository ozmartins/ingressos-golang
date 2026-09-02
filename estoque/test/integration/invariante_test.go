//go:build integration

package integration

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
)

func TestInvarianteApos1000Ciclos(t *testing.T) {
	c := montarCenario(t, false)
	ctx := context.Background()
	sessao := c.novaSessao(t, []string{"A", "B", "C", "D"}, 10)

	aleatorio := rand.New(rand.NewSource(20260829))
	const ciclos = 1000
	abandonadas := 0

	for i := 0; i < ciclos; i++ {
		fileira := string(rune('A' + aleatorio.Intn(4)))
		numero := 1 + aleatorio.Intn(10)
		rotulo := poltrona.MontarRotulo(fileira, numero)

		resultado, err := c.Bloquear.Executar(ctx, sessao, usuario, []string{rotulo})
		if err != nil {
			t.Fatalf("ciclo %d: bloqueio falhou: %v", i, err)
		}
		if !resultado.Concedido {
			continue
		}

		switch aleatorio.Intn(3) {
		case 0:
			if _, err := c.Confirmar.Executar(ctx, filaSucesso, resultado.Reserva.ID, resultado.Reserva.ID); err != nil {
				t.Fatalf("ciclo %d: confirmação falhou: %v", i, err)
			}
		case 1:
			if _, err := c.Cancelar.Executar(ctx, filaFalhou, resultado.Reserva.ID, resultado.Reserva.ID); err != nil {
				t.Fatalf("ciclo %d: cancelamento falhou: %v", i, err)
			}
		default:
			abandonadas++
		}
	}

	c.Relogio.Avancar(11 * time.Minute)
	for {
		n, err := c.Expirar.Varrer(ctx)
		if err != nil {
			t.Fatalf("varredura: %v", err)
		}
		if n == 0 {
			break
		}
	}

	contagem := c.contarPorStatus(t, sessao)
	if contagem["RESERVADA"] != 0 {
		t.Errorf("%d poltrona(s) presas em RESERVADA após todos os desfechos", contagem["RESERVADA"])
	}
	if contagem["LIVRE"]+contagem["OCUPADA"] != 40 {
		t.Errorf("contagem final = %v, esperado 40 poltronas entre LIVRE e OCUPADA", contagem)
	}

	var pendentes int
	if err := c.Pool.QueryRow(ctx,
		`SELECT count(*) FROM reservas WHERE sessao_id = $1 AND status = 'PENDENTE'`, sessao).Scan(&pendentes); err != nil {
		t.Fatal(err)
	}
	if pendentes != 0 {
		t.Errorf("%d reserva(s) ainda pendentes", pendentes)
	}

	var incoerentes int
	err := c.Pool.QueryRow(ctx, `
		SELECT count(*) FROM poltronas p
		 WHERE p.sessao_id = $1 AND p.status = 'OCUPADA'
		   AND (SELECT count(*) FROM reserva_poltronas rp
		          JOIN reservas r ON r.id = rp.reserva_id
		         WHERE rp.poltrona_id = p.id AND r.status = 'CONFIRMADA') <> 1`, sessao).Scan(&incoerentes)
	if err != nil {
		t.Fatal(err)
	}
	if incoerentes != 0 {
		t.Errorf("%d poltrona(s) OCUPADA sem exatamente uma reserva confirmada", incoerentes)
	}

	var vazando int
	err = c.Pool.QueryRow(ctx, `
		SELECT count(*) FROM poltronas p
		  JOIN reserva_poltronas rp ON rp.poltrona_id = p.id
		  JOIN reservas r ON r.id = rp.reserva_id
		 WHERE p.sessao_id = $1 AND p.status = 'LIVRE' AND r.status = 'PENDENTE'`, sessao).Scan(&vazando)
	if err != nil {
		t.Fatal(err)
	}
	if vazando != 0 {
		t.Errorf("%d poltrona(s) LIVRE vinculadas a reserva pendente", vazando)
	}

	t.Logf("ciclos=%d abandonadas=%d final=%v", ciclos, abandonadas, contagem)
}
