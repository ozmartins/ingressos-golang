//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func TestBloqueioGravaTudoNaMesmaTransacao(t *testing.T) {
	c := montarCenario(t, false)
	sessao := c.novaSessao(t, []string{"A"}, 5)
	ctx := context.Background()

	resultado, err := c.Bloquear.Executar(ctx, sessao, usuario, []string{"A1", "A2"})
	if err != nil || !resultado.Concedido {
		t.Fatalf("bloqueio: %v", err)
	}

	if got := c.statusReserva(t, resultado.Reserva.ID); got != "PENDENTE" {
		t.Errorf("reserva = %s, esperado PENDENTE", got)
	}
	for _, rotulo := range []string{"A1", "A2"} {
		if got := c.statusPoltrona(t, sessao, rotulo); got != poltrona.Reservada {
			t.Errorf("%s = %s, esperado RESERVADA", rotulo, got)
		}
	}
	if got := c.statusPoltrona(t, sessao, "A3"); got != poltrona.Livre {
		t.Errorf("A3 = %s, esperado LIVRE", got)
	}

	var vinculos int
	if err := c.Pool.QueryRow(ctx,
		`SELECT count(*) FROM reserva_poltronas WHERE reserva_id = $1`, resultado.Reserva.ID).Scan(&vinculos); err != nil {
		t.Fatal(err)
	}
	if vinculos != 2 {
		t.Errorf("vínculos = %d, esperado 2", vinculos)
	}

	var payload []byte
	var traceContext []byte
	var publicadoEm *string
	err = c.Pool.QueryRow(ctx,
		`SELECT payload, trace_context, publicado_em FROM outbox_eventos WHERE message_id = $1`,
		resultado.Reserva.ID).Scan(&payload, &traceContext, &publicadoEm)
	if err != nil {
		t.Fatalf("fato não foi para a caixa de saída: %v", err)
	}
	if publicadoEm != nil {
		t.Error("fato não pode nascer publicado")
	}

	var contexto map[string]string
	if err := json.Unmarshal(traceContext, &contexto); err != nil || contexto["traceparent"] == "" {
		t.Errorf("fato sem contexto de rastreamento: %s", traceContext)
	}

	var evento usecase.EventoReservaCriada
	if err := json.Unmarshal(payload, &evento); err != nil {
		t.Fatalf("payload inválido: %v", err)
	}
	if len(evento.PoltronasIDs) != 2 || evento.PoltronasIDs[0] != "A1" {
		t.Errorf("poltronas_ids = %v", evento.PoltronasIDs)
	}
}

func TestBloqueioRecusadoNaoAlteraEstado(t *testing.T) {
	c := montarCenario(t, false)
	sessao := c.novaSessao(t, []string{"A"}, 3)
	ctx := context.Background()

	if _, err := c.Bloquear.Executar(ctx, sessao, usuario, []string{"A1"}); err != nil {
		t.Fatalf("primeiro bloqueio: %v", err)
	}

	resultado, err := c.Bloquear.Executar(ctx, sessao, "outra-pessoa", []string{"A1", "A2", "A3"})
	if err != nil {
		t.Fatalf("indisponibilidade não é erro: %v", err)
	}
	if resultado.Concedido {
		t.Fatal("esperava recusa")
	}
	for _, rotulo := range []string{"A2", "A3"} {
		if got := c.statusPoltrona(t, sessao, rotulo); got != poltrona.Livre {
			t.Errorf("%s = %s, esperado LIVRE", rotulo, got)
		}
	}

	var reservas int
	if err := c.Pool.QueryRow(ctx, `SELECT count(*) FROM reservas WHERE sessao_id = $1`, sessao).Scan(&reservas); err != nil {
		t.Fatal(err)
	}
	if reservas != 1 {
		t.Errorf("reservas = %d, esperado 1 — a recusa criou reserva", reservas)
	}
}

func TestBloqueioRecusaSessaoNaoProvisionada(t *testing.T) {
	c := montarCenario(t, false)

	_, err := c.Bloquear.Executar(context.Background(), "sessao-inexistente", usuario, []string{"A1"})
	if !errors.Is(err, shared.ErrSessaoNaoProvisionada) {
		t.Fatalf("erro = %v, esperado ErrSessaoNaoProvisionada", err)
	}
}

func TestBloqueioRecusaRotuloInexistenteNaSessao(t *testing.T) {
	c := montarCenario(t, false)
	sessao := c.novaSessao(t, []string{"A"}, 2)

	_, err := c.Bloquear.Executar(context.Background(), sessao, usuario, []string{"A1", "Z9"})
	if !errors.Is(err, shared.ErrPoltronaInexistente) {
		t.Fatalf("erro = %v, esperado ErrPoltronaInexistente", err)
	}
	if got := c.statusPoltrona(t, sessao, "A1"); got != poltrona.Livre {
		t.Errorf("A1 = %s, esperado LIVRE", got)
	}
}

func TestConcorrenciaExatamenteUmVencedor(t *testing.T) {
	c := montarCenario(t, false)
	sessao := c.novaSessao(t, []string{"A"}, 10)
	ctx := context.Background()

	const paralelas = 100
	var largada sync.WaitGroup
	var fim sync.WaitGroup
	largada.Add(1)

	concedidos := make([]bool, paralelas)
	erros := make([]error, paralelas)

	for i := 0; i < paralelas; i++ {
		fim.Add(1)
		go func(i int) {
			defer fim.Done()
			largada.Wait()
			resultado, err := c.Bloquear.Executar(ctx, sessao, usuario, []string{"A1", "A2"})
			concedidos[i] = err == nil && resultado.Concedido
			if err != nil && !errors.Is(err, shared.ErrPoltronasIndisponiveis) {
				erros[i] = err
			}
		}(i)
	}

	largada.Done()
	fim.Wait()

	vencedores := 0
	for i, ok := range concedidos {
		if ok {
			vencedores++
		}
		if erros[i] != nil {
			t.Errorf("solicitação %d falhou de forma inesperada: %v", i, erros[i])
		}
	}
	if vencedores != 1 {
		t.Fatalf("vencedores = %d, esperado exatamente 1", vencedores)
	}

	var duplicadas int
	err := c.Pool.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT poltrona_id FROM reserva_poltronas rp
			  JOIN reservas r ON r.id = rp.reserva_id
			  JOIN poltronas p ON p.id = rp.poltrona_id
			 WHERE p.sessao_id = $1 AND r.status = 'PENDENTE'
			 GROUP BY poltrona_id HAVING count(*) > 1
		) AS d`, sessao).Scan(&duplicadas)
	if err != nil {
		t.Fatal(err)
	}
	if duplicadas != 0 {
		t.Fatalf("%d poltrona(s) vinculadas a mais de uma reserva ativa", duplicadas)
	}
}

func TestBancoIndisponivelRecusaSemConceder(t *testing.T) {
	c := montarCenario(t, false)
	sessao := c.novaSessao(t, []string{"A"}, 2)

	c.Banco.Fechar()

	_, err := c.Bloquear.Executar(context.Background(), sessao, usuario, []string{"A1"})
	if err == nil {
		t.Fatal("esperava recusa com o banco indisponível")
	}
	if !errors.Is(err, shared.ErrDependenciaIndisponivel) {
		t.Fatalf("erro = %v, esperado ErrDependenciaIndisponivel", err)
	}

	verificacao := montarCenario(t, false)
	contagem := verificacao.contarPorStatus(t, sessao)
	if contagem["RESERVADA"] != 0 {
		t.Errorf("poltronas reservadas = %d, esperado 0", contagem["RESERVADA"])
	}
}
