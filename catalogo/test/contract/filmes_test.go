package contract

import (
	"net/http"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

func ptr(s string) *string { return &s }

func catalogoDeTeste() []catalogo.Filme {
	return []catalogo.Filme{
		{ID: "c394c8b3-76a1-4328-b803-02f5923b7a15", Titulo: "Duna: Parte 2", Sinopse: ptr("Paul Atreides..."),
			DuracaoMinutos: 166, ClassificacaoEtaria: "14 anos", Genero: "Ficção Científica",
			ImagemURL: ptr("https://cdn.cinema.com/posters/duna2.jpg"), Status: catalogo.StatusEmCartaz},
		{ID: "a1b2c3d4-0000-4000-8000-000000000002", Titulo: "Aurora", DuracaoMinutos: 100,
			ClassificacaoEtaria: "Livre", Genero: "Drama", Status: catalogo.StatusBreve},
		{ID: "a1b2c3d4-0000-4000-8000-000000000003", Titulo: "Zebra", DuracaoMinutos: 90,
			ClassificacaoEtaria: "Livre", Genero: "Comédia", Status: catalogo.StatusForaDeCartaz},
	}
}

func TestGetFilmesRespondeEnvelopeDePaginacao(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := obter(t, s, "/api/v1/filmes")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)

	if e.Pagina.Total != 2 {
		t.Fatalf("esperava 2 filmes públicos, obteve %d", e.Pagina.Total)
	}
	if e.Pagina.Pagina != 1 || e.Pagina.Tamanho != 20 || e.Pagina.TemProxima {
		t.Fatalf("bloco de paginação inconsistente: %+v", e.Pagina)
	}

	obrigatorios := []string{"id", "titulo", "duracao_minutos", "classificacao_etaria", "genero", "status"}
	for _, item := range e.Itens {
		for _, campo := range obrigatorios {
			if _, ok := item[campo]; !ok {
				t.Errorf("campo obrigatório %q ausente em %v", campo, item)
			}
		}
	}
}

func TestGetFilmesOmiteCamposOpcionaisAusentes(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	_, corpo := obter(t, s, "/api/v1/filmes")
	e := decodificarEnvelope(t, corpo)

	for _, item := range e.Itens {
		if item["titulo"] == "Aurora" {
			if _, presente := item["sinopse"]; presente {
				t.Error("sinopse ausente deveria ser omitida do JSON")
			}
			if _, presente := item["imagem_url"]; presente {
				t.Error("imagem_url ausente deveria ser omitida do JSON")
			}
		}
	}
}

func TestGetFilmesFiltraPorStatus(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	_, corpo := obter(t, s, "/api/v1/filmes?status=EM_CARTAZ")
	e := decodificarEnvelope(t, corpo)

	if e.Pagina.Total != 1 || e.Itens[0]["status"] != "EM_CARTAZ" {
		t.Fatalf("filtro não aplicado: %+v", e)
	}
}

func TestGetFilmesSemResultadosDevolvePaginaVazia(t *testing.T) {
	s := montarComFilmes(t, nil)
	resp, corpo := obter(t, s, "/api/v1/filmes")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	if e.Itens == nil {
		t.Fatal("itens deveria ser [] e não null")
	}
	if len(e.Itens) != 0 || e.Pagina.Total != 0 {
		t.Fatalf("esperava página vazia, obteve %+v", e)
	}
}

func TestGetFilmesRecusaStatusDesconhecido(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := obter(t, s, "/api/v1/filmes?status=EM_BREVE")

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("esperava 400, obteve %d", resp.StatusCode)
	}
	p := decodificarProblem(t, resp, corpo)
	if p.Type != "https://cinema.example/errors/parametro-invalido" {
		t.Fatalf("type inesperado: %s", p.Type)
	}
	for _, aceito := range []string{"EM_CARTAZ", "BREVE", "FORA_DE_CARTAZ"} {
		if !contemString(p.Detail, aceito) {
			t.Errorf("detail deveria listar %s: %s", aceito, p.Detail)
		}
	}
}

func TestGetFilmesRecusaPageSizeAcimaDoTeto(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := obter(t, s, "/api/v1/filmes?page_size=500")

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("esperava 400, obteve %d", resp.StatusCode)
	}
	p := decodificarProblem(t, resp, corpo)
	if !contemString(p.Detail, "100") {
		t.Errorf("detail deveria informar o máximo aceito: %s", p.Detail)
	}
}

func TestGetFilmesRecusaPaginacaoMalformada(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	for _, q := range []string{"?page=0", "?page=-1", "?page=abc", "?page_size=0", "?page_size=xyz"} {
		resp, _ := obter(t, s, "/api/v1/filmes"+q)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: esperava 400, obteve %d", q, resp.StatusCode)
		}
	}
}

func TestGetFilmesPaginaAlemDoFim(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := obter(t, s, "/api/v1/filmes?page=9999")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	if len(e.Itens) != 0 {
		t.Fatalf("esperava página vazia, obteve %d itens", len(e.Itens))
	}
	if e.Pagina.Total != 2 || e.Pagina.TemProxima {
		t.Fatalf("total ou tem_proxima errados: %+v", e.Pagina)
	}
}

func contemString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
