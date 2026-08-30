package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const filaSessao = "estoque.sessao-criada"

func eventoSessao(poltronas ...LayoutPoltrona) EventoSessaoCriada {
	return EventoSessaoCriada{
		Evento: "SESSAO_CRIADA", Versao: 1, SessaoID: sessao, Poltronas: poltronas,
	}
}

func TestProvisionarCriaMatrizCompletaEmLivre(t *testing.T) {
	estoque, log := novoEstoqueFalso(), &logFalso{}
	uc := ProvisionarSessao{Poltronas: estoque, Log: log}

	evento := eventoSessao(
		LayoutPoltrona{Fileira: "B", Numero: 2, Tipo: "PCD"},
		LayoutPoltrona{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
		LayoutPoltrona{Fileira: "A", Numero: 2, Tipo: "NAMORADEIRA"},
	)

	res, err := uc.Executar(context.Background(), filaSessao, sessao, evento)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if res != TransicaoAplicada {
		t.Fatalf("resultado = %s, esperado aplicada", res)
	}

	mapa, err := (ConsultarMapa{Poltronas: estoque}).Executar(context.Background(), sessao)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(mapa) != 3 {
		t.Fatalf("poltronas = %d, esperado 3", len(mapa))
	}
	for _, p := range mapa {
		if p.Status != poltrona.Livre {
			t.Errorf("%s nasceu %s, esperado LIVRE", p.Rotulo, p.Status)
		}
	}
}

func TestProvisionarEhIdempotenteENaoReiniciaEstado(t *testing.T) {
	estoque, prazo, log := novoEstoqueFalso(), novoPrazoFalso(), &logFalso{}
	uc := ProvisionarSessao{Poltronas: estoque, Log: log}
	evento := eventoSessao(
		LayoutPoltrona{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
		LayoutPoltrona{Fileira: "A", Numero: 2, Tipo: "NORMAL"},
	)

	if _, err := uc.Executar(context.Background(), filaSessao, sessao, evento); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if _, err := montarBloqueio(estoque, prazo, log).
		Executar(context.Background(), sessao, usuario, []string{"A1"}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Reanúncio do mesmo fato, com a mesma chave de idempotência.
	res, err := uc.Executar(context.Background(), filaSessao, sessao, evento)
	if err != nil {
		t.Fatalf("reanúncio não pode virar erro: %v", err)
	}
	if res != TransicaoIgnoradaDuplicata {
		t.Errorf("resultado = %s, esperado ignorada-duplicata", res)
	}

	mapa, _ := (ConsultarMapa{Poltronas: estoque}).Executar(context.Background(), sessao)
	if len(mapa) != 2 {
		t.Errorf("poltronas = %d, esperado 2 — reanúncio duplicou a matriz", len(mapa))
	}
	// O ponto que mais importa: a poltrona já reservada não pode voltar a LIVRE.
	if got := estoque.statusDe(sessao, "A1"); got != poltrona.Reservada {
		t.Errorf("A1 = %s, esperado RESERVADA — reanúncio reiniciou estado", got)
	}
}

func TestProvisionarRecusaLayoutInvalido(t *testing.T) {
	casos := map[string]EventoSessaoCriada{
		"fileira e número repetidos": eventoSessao(
			LayoutPoltrona{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
			LayoutPoltrona{Fileira: "a", Numero: 1, Tipo: "NORMAL"},
		),
		"tipo desconhecido": eventoSessao(
			LayoutPoltrona{Fileira: "A", Numero: 1, Tipo: "TRONO"},
		),
		"número inválido": eventoSessao(
			LayoutPoltrona{Fileira: "A", Numero: 0, Tipo: "NORMAL"},
		),
	}

	for nome, evento := range casos {
		t.Run(nome, func(t *testing.T) {
			estoque := novoEstoqueFalso()
			uc := ProvisionarSessao{Poltronas: estoque, Log: &logFalso{}}

			if _, err := uc.Executar(context.Background(), filaSessao, sessao, evento); err == nil {
				t.Fatal("esperava recusa do layout")
			}
			// FR-035: tudo-ou-nada — nada pode ter sido provisionado.
			mapa, _ := estoque.MapaDaSessao(context.Background(), sessao)
			if len(mapa) != 0 {
				t.Errorf("layout inválido provisionou %d poltrona(s)", len(mapa))
			}
		})
	}
}

func TestLerSessaoCriada(t *testing.T) {
	valido := []byte(`{"evento":"SESSAO_CRIADA","versao":1,"sessao_id":"s-1","poltronas":[{"fileira":"A","numero":1,"tipo":"NORMAL"}]}`)
	if _, err := LerSessaoCriada(valido); err != nil {
		t.Fatalf("corpo válido recusado: %v", err)
	}
	for nome, corpo := range map[string][]byte{
		"json inválido":     []byte(`nao sou json`),
		"sessao_id ausente": []byte(`{"poltronas":[{"fileira":"A","numero":1,"tipo":"NORMAL"}]}`),
		"sem poltronas":     []byte(`{"sessao_id":"s-1","poltronas":[]}`),
	} {
		if _, err := LerSessaoCriada(corpo); err == nil {
			t.Errorf("%s devia ser recusado", nome)
		}
	}
}

func TestConsultarMapaDistingueSessaoDesconhecida(t *testing.T) {
	estoque := novoEstoqueFalso()
	uc := ConsultarMapa{Poltronas: estoque}

	_, err := uc.Executar(context.Background(), "sessao-que-nao-existe")
	if !errors.Is(err, shared.ErrSessaoDesconhecida) {
		t.Fatalf("erro = %v, esperado ErrSessaoDesconhecida", err)
	}

	if _, err := uc.Executar(context.Background(), ""); !errors.Is(err, shared.ErrSolicitacaoInvalida) {
		t.Errorf("sessão vazia: erro = %v, esperado ErrSolicitacaoInvalida", err)
	}

	estoque.provisionar(sessao, "A1")
	mapa, err := uc.Executar(context.Background(), sessao)
	if err != nil || len(mapa) != 1 {
		t.Fatalf("sessão provisionada: mapa=%d err=%v", len(mapa), err)
	}
}
