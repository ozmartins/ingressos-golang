package contract

import (
	"net/http"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

func gradeDeTeste(n int) []catalogo.SessaoDetalhada {
	base := agora().Add(24 * time.Hour)
	var g []catalogo.SessaoDetalhada
	for i := 0; i < n; i++ {
		g = append(g, catalogo.SessaoDetalhada{
			ID:             "f781a9b2-11e2-4f81-a901-88900000000" + string(rune('0'+i%10)),
			FilmeID:        "c394c8b3-76a1-4328-b803-02f5923b7a15",
			FilmeTitulo:    "Duna: Parte 2",
			CinemaID:       "b1b2c3d4-0000-4000-8000-000000000001",
			CinemaNome:     "CineMark - Shopping Centro",
			SalaNumero:     3,
			TipoTela:       catalogo.TelaIMAX,
			DataHoraInicio: base.Add(time.Duration(i) * time.Hour),
			Idioma:         catalogo.Legendado,
			PrecoBase:      catalogo.DinheiroDeCentavos(4200),
		})
	}
	return g
}

func TestGetSessoesRetornaCamposConsolidados(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.sessoes.grade = gradeDeTeste(3) })
	resp, corpo := obter(t, amb.servidor, "/api/v1/sessoes")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	if len(e.Itens) != 3 {
		t.Fatalf("esperava 3 sessões, obteve %d", len(e.Itens))
	}
	obrigatorios := []string{"id", "filme_id", "filme_titulo", "cinema_id", "cinema_nome",
		"sala_numero", "tipo_tela", "data_hora_inicio", "idioma", "preco_base"}
	for _, campo := range obrigatorios {
		if _, ok := e.Itens[0][campo]; !ok {
			t.Errorf("campo obrigatório %q ausente", campo)
		}
	}
}

func TestGetSessoesPrecoBaseEhTextoExato(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.sessoes.grade = gradeDeTeste(1) })
	_, corpo := obter(t, amb.servidor, "/api/v1/sessoes")
	e := decodificarEnvelope(t, corpo)

	preco, ok := e.Itens[0]["preco_base"].(string)
	if !ok {
		t.Fatalf("preco_base deveria ser string, veio %T", e.Itens[0]["preco_base"])
	}
	if preco != "42.00" {
		t.Fatalf("esperava \"42.00\", obteve %q", preco)
	}
}

func TestGetSessoesPaginacaoNaoRepeteNemOmite(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.sessoes.grade = gradeDeTeste(5) })

	_, corpo1 := obter(t, amb.servidor, "/api/v1/sessoes?page=1&page_size=2")
	_, corpo2 := obter(t, amb.servidor, "/api/v1/sessoes?page=2&page_size=2")
	p1, p2 := decodificarEnvelope(t, corpo1), decodificarEnvelope(t, corpo2)

	if p1.Pagina.Total != 5 || p2.Pagina.Total != 5 {
		t.Fatalf("total deve refletir o filtro, não a página: %d/%d", p1.Pagina.Total, p2.Pagina.Total)
	}
	if !p1.Pagina.TemProxima || !p2.Pagina.TemProxima {
		t.Error("com 5 registros e páginas de 2, ambas deveriam indicar próxima")
	}
	vistos := map[string]bool{}
	for _, item := range append(p1.Itens, p2.Itens...) {
		id := item["id"].(string)
		if vistos[id] {
			t.Errorf("sessão %s apareceu em duas páginas consecutivas", id)
		}
		vistos[id] = true
	}
	if len(vistos) != 4 {
		t.Fatalf("esperava 4 sessões distintas em duas páginas de 2, obteve %d", len(vistos))
	}
}

func TestGetSessoesRecusaDataInvalida(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.sessoes.grade = gradeDeTeste(1) })
	for _, data := range []string{"01-09-2026", "2026/09/01", "amanha", "2026-02-31"} {
		resp, corpo := obter(t, amb.servidor, "/api/v1/sessoes?data="+data)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("data %q: esperava 400, obteve %d", data, resp.StatusCode)
			continue
		}
		p := decodificarProblem(t, resp, corpo)
		if p.Type != "https://cinema.example/errors/parametro-invalido" {
			t.Errorf("data %q: type inesperado %s", data, p.Type)
		}
	}
}

func TestGetSessoesAceitaDataValida(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.sessoes.grade = gradeDeTeste(1) })
	resp, _ := obter(t, amb.servidor, "/api/v1/sessoes?data=2026-09-01")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", resp.StatusCode)
	}
}

func TestGetSessoesRecusaFiltrosMalformados(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.sessoes.grade = gradeDeTeste(1) })
	for _, q := range []string{"?filme_id=abc", "?cinema_id=123"} {
		resp, _ := obter(t, amb.servidor, "/api/v1/sessoes"+q)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: esperava 400, obteve %d", q, resp.StatusCode)
		}
	}
}
