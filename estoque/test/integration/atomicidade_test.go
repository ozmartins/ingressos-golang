//go:build integration

package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func TestFalhaNoFatoDesfazTodaAConcessao(t *testing.T) {
	c := montarCenario(t, false)
	sessao := c.novaSessao(t, []string{"A"}, 3)
	ctx := context.Background()

	sol, err := reserva.NovaSolicitacao(sessao, usuario, []string{"A1", "A2"}, valorDeTeste, 10)
	if err != nil {
		t.Fatal(err)
	}
	res := reserva.Nova(sol, c.Relogio.Agora(), 10*time.Minute)

	fato := usecase.FatoPendente{
		MessageID:  strings.Repeat("x", 65),
		RoutingKey: "reserva.criada",
		Payload:    []byte(`{}`),
	}

	err = c.Reservas.Conceder(ctx, sol, res, fato)
	if !errors.Is(err, shared.ErrDependenciaIndisponivel) {
		t.Fatalf("erro = %v, esperado ErrDependenciaIndisponivel", err)
	}

	for _, rotulo := range []string{"A1", "A2", "A3"} {
		if got := c.statusPoltrona(t, sessao, rotulo); got != poltrona.Livre {
			t.Errorf("%s = %s, esperado LIVRE — a falha deixou rastro", rotulo, got)
		}
	}

	var reservas, vinculos, fatos int
	if err := c.Pool.QueryRowContext(ctx,
		`SELECT count(*) FROM reservas WHERE sessao_id = $1`, sessao).Scan(&reservas); err != nil {
		t.Fatal(err)
	}
	if err := c.Pool.QueryRowContext(ctx,
		`SELECT count(*) FROM reserva_poltronas WHERE reserva_id = $1`, res.ID).Scan(&vinculos); err != nil {
		t.Fatal(err)
	}
	if err := c.Pool.QueryRowContext(ctx,
		`SELECT count(*) FROM outbox_eventos WHERE routing_key = 'reserva.criada' AND message_id = $1`, fato.MessageID).Scan(&fatos); err != nil {
		t.Fatal(err)
	}
	if reservas != 0 || vinculos != 0 || fatos != 0 {
		t.Errorf("sobraram reservas=%d vínculos=%d fatos=%d, esperado tudo 0", reservas, vinculos, fatos)
	}

	out, err := c.Bloquear.Executar(ctx, sessao, usuario, []string{"A1", "A2"}, valorDeTeste)
	if err != nil || !out.Concedido {
		t.Fatalf("bloqueio posterior: concedido=%v err=%v", out.Concedido, err)
	}
}
