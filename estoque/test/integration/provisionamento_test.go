//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

const filaSessao = "estoque.sessao-criada"

func eventoDeSessao(sessaoID string, poltronas ...usecase.LayoutPoltrona) usecase.EventoSessaoCriada {
	return usecase.EventoSessaoCriada{
		Evento: "SESSAO_CRIADA", Versao: 1, SessaoID: sessaoID,
		OcorridoEm: time.Now().UTC().Format(time.RFC3339), Poltronas: poltronas,
	}
}

// TestProvisionamentoDisponibilizaMatrizParaBloqueio cobre SC-015.
func TestProvisionamentoDisponibilizaMatrizParaBloqueio(t *testing.T) {
	c := montarCenario(t, false)
	ctx := context.Background()
	sessaoID := uuid.NewString()

	evento := eventoDeSessao(sessaoID,
		usecase.LayoutPoltrona{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
		usecase.LayoutPoltrona{Fileira: "A", Numero: 2, Tipo: "PCD"},
		usecase.LayoutPoltrona{Fileira: "B", Numero: 1, Tipo: "NAMORADEIRA"},
	)

	if _, err := c.Provisionar.Executar(ctx, filaSessao, sessaoID, evento); err != nil {
		t.Fatalf("provisionar: %v", err)
	}

	mapa, err := c.Consultar.Executar(ctx, sessaoID)
	if err != nil {
		t.Fatalf("consultar mapa: %v", err)
	}
	if len(mapa) != 3 {
		t.Fatalf("poltronas = %d, esperado 3", len(mapa))
	}
	// Ordenado por fileira e número, como a sala é desenhada.
	if mapa[0].Rotulo != "A1" || mapa[1].Rotulo != "A2" || mapa[2].Rotulo != "B1" {
		t.Errorf("ordem = %s %s %s", mapa[0].Rotulo, mapa[1].Rotulo, mapa[2].Rotulo)
	}
	for _, p := range mapa {
		if p.Status != poltrona.Livre {
			t.Errorf("%s nasceu %s, esperado LIVRE", p.Rotulo, p.Status)
		}
	}
	if mapa[1].Tipo != poltrona.PCD || mapa[2].Tipo != poltrona.Namoradeira {
		t.Errorf("tipos não preservados: %s %s", mapa[1].Tipo, mapa[2].Tipo)
	}

	resultado, err := c.Bloquear.Executar(ctx, sessaoID, usuario, []string{"A1"})
	if err != nil || !resultado.Concedido {
		t.Fatalf("matriz provisionada devia aceitar bloqueio: %v", err)
	}
}

// TestReanuncioNaoDuplicaNemReiniciaEstado cobre SC-004 do lado do provisionamento.
func TestReanuncioNaoDuplicaNemReiniciaEstado(t *testing.T) {
	c := montarCenario(t, false)
	ctx := context.Background()
	sessaoID := uuid.NewString()

	evento := eventoDeSessao(sessaoID,
		usecase.LayoutPoltrona{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
		usecase.LayoutPoltrona{Fileira: "A", Numero: 2, Tipo: "NORMAL"},
	)
	if _, err := c.Provisionar.Executar(ctx, filaSessao, sessaoID, evento); err != nil {
		t.Fatalf("provisionar: %v", err)
	}
	if _, err := c.Bloquear.Executar(ctx, sessaoID, usuario, []string{"A1"}); err != nil {
		t.Fatalf("bloqueio: %v", err)
	}

	// Reentrega com a mesma chave de idempotência.
	res, err := c.Provisionar.Executar(ctx, filaSessao, sessaoID, evento)
	if err != nil {
		t.Fatalf("reanúncio não pode virar erro: %v", err)
	}
	if res != usecase.TransicaoIgnoradaDuplicata {
		t.Errorf("resultado = %s, esperado ignorada-duplicata", res)
	}

	// Reentrega com chave diferente: a chave única da tabela é a segunda linha
	// de defesa e precisa preservar o estado corrente.
	if _, err := c.Provisionar.Executar(ctx, filaSessao, "outra-chave", evento); err != nil {
		t.Fatalf("reanúncio com outra chave: %v", err)
	}

	contagem := c.contarPorStatus(t, sessaoID)
	if contagem["RESERVADA"] != 1 || contagem["LIVRE"] != 1 {
		t.Errorf("contagem = %v, esperado 1 RESERVADA e 1 LIVRE", contagem)
	}
}

func TestProvisionamentoRecusaLayoutInvalidoSemDeixarRastro(t *testing.T) {
	c := montarCenario(t, false)
	ctx := context.Background()

	casos := map[string][]usecase.LayoutPoltrona{
		"fileira e número repetidos": {
			{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
			{Fileira: "a", Numero: 1, Tipo: "NORMAL"},
		},
		"tipo desconhecido": {
			{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
			{Fileira: "A", Numero: 2, Tipo: "TRONO"},
		},
	}

	for nome, layout := range casos {
		t.Run(nome, func(t *testing.T) {
			sessaoID := uuid.NewString()
			_, err := c.Provisionar.Executar(ctx, filaSessao, sessaoID, eventoDeSessao(sessaoID, layout...))
			if err == nil {
				t.Fatal("esperava recusa do layout")
			}
			// Erro de conteúdo, para ir à DLQ e não voltar para a fila.
			if !errors.Is(err, shared.ErrSolicitacaoInvalida) {
				t.Errorf("erro = %v, esperado ErrSolicitacaoInvalida", err)
			}
			// FR-035: tudo-ou-nada.
			if contagem := c.contarPorStatus(t, sessaoID); len(contagem) != 0 {
				t.Errorf("layout inválido provisionou %v", contagem)
			}
		})
	}
}

// TestBloqueioAntesDoProvisionamento cobre FR-036 e o edge case da spec: o
// bloqueio que chega antes do fato de sessão criada é recusado, e passa a ser
// aceito assim que a matriz existe.
func TestBloqueioAntesDoProvisionamento(t *testing.T) {
	c := montarCenario(t, false)
	ctx := context.Background()
	sessaoID := uuid.NewString()

	_, err := c.Bloquear.Executar(ctx, sessaoID, usuario, []string{"A1"})
	if !errors.Is(err, shared.ErrSessaoNaoProvisionada) {
		t.Fatalf("erro = %v, esperado ErrSessaoNaoProvisionada", err)
	}

	if _, err := c.Provisionar.Executar(ctx, filaSessao, sessaoID,
		eventoDeSessao(sessaoID, usecase.LayoutPoltrona{Fileira: "A", Numero: 1, Tipo: "NORMAL"})); err != nil {
		t.Fatalf("provisionar: %v", err)
	}

	resultado, err := c.Bloquear.Executar(ctx, sessaoID, usuario, []string{"A1"})
	if err != nil || !resultado.Concedido {
		t.Fatalf("bloqueio após provisionamento: %v", err)
	}
}

// TestMapaCoerenteDuranteBloqueioConcorrente cobre o edge case de leitura: a
// consulta devolve um retrato coerente, sem estado intermediário.
func TestMapaCoerenteDuranteBloqueioConcorrente(t *testing.T) {
	c := montarCenario(t, false)
	sessao := c.novaSessao(t, []string{"A"}, 10)
	ctx := context.Background()

	fim := make(chan struct{})
	go func() {
		defer close(fim)
		for i := 1; i <= 5; i++ {
			_, _ = c.Bloquear.Executar(ctx, sessao, usuario,
				[]string{poltrona.MontarRotulo("A", i)})
		}
	}()

	for i := 0; i < 50; i++ {
		mapa, err := c.Consultar.Executar(ctx, sessao)
		if err != nil {
			t.Fatalf("consulta durante bloqueio: %v", err)
		}
		if len(mapa) != 10 {
			t.Fatalf("mapa parcial durante bloqueio: %d poltronas", len(mapa))
		}
		for _, p := range mapa {
			switch p.Status {
			case poltrona.Livre, poltrona.Reservada, poltrona.Ocupada:
			default:
				t.Fatalf("%s em estado intermediário: %s", p.Rotulo, p.Status)
			}
		}
	}
	<-fim
}
