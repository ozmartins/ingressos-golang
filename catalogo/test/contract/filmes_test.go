package contract

import (
	"encoding/json"
	"net/http"
	"strings"
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

const corpoFilmeValido = `{"titulo":"Novo Filme","duracao_minutos":120,` +
	`"classificacao_etaria":"12 anos","genero":"Drama"}`

func decodificarFilme(t *testing.T, corpo []byte) map[string]any {
	t.Helper()
	var filme map[string]any
	if err := json.Unmarshal(corpo, &filme); err != nil {
		t.Fatalf("resposta não é um filme JSON: %v (corpo: %s)", err, corpo)
	}
	return filme
}

func TestGetFilmePorIDDevolveOFilme(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := obter(t, s, "/api/v1/filmes/c394c8b3-76a1-4328-b803-02f5923b7a15")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if filme := decodificarFilme(t, corpo); filme["titulo"] != "Duna: Parte 2" {
		t.Fatalf("filme inesperado: %v", filme)
	}
}

func TestGetFilmePorIDEnxergaForaDeCartaz(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := obter(t, s, "/api/v1/filmes/a1b2c3d4-0000-4000-8000-000000000003")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("o recorte público vale só para a listagem; status %d", resp.StatusCode)
	}
	if filme := decodificarFilme(t, corpo); filme["status"] != "FORA_DE_CARTAZ" {
		t.Fatalf("status inesperado: %v", filme["status"])
	}
}

func TestGetFilmeInexistenteDevolve404DeFilme(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := obter(t, s, "/api/v1/filmes/a1b2c3d4-0000-4000-8000-999999999999")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "filme-nao-encontrado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestGetFilmeComIDMalformadoDevolve400(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := obter(t, s, "/api/v1/filmes/nao-e-uuid")

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, esperava 400", resp.StatusCode)
	}
	decodificarProblem(t, resp, corpo)
}

func TestPostFilmeCriaEDevolveLocation(t *testing.T) {
	s := montarComFilmes(t, nil)
	resp, corpo := requisitar(t, s, http.MethodPost, "/api/v1/filmes", "token-bom", corpoFilmeValido)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	filme := decodificarFilme(t, corpo)
	id, _ := filme["id"].(string)
	if id == "" {
		t.Fatal("o serviço deveria gerar o id do filme")
	}
	if local := resp.Header.Get("Location"); local != "/api/v1/filmes/"+id {
		t.Fatalf("Location = %q, esperava o caminho do filme criado", local)
	}
	if filme["status"] != "EM_CARTAZ" {
		t.Fatalf("sem status no corpo, o filme deveria nascer EM_CARTAZ, veio %v", filme["status"])
	}

	respBusca, corpoBusca := obter(t, s, "/api/v1/filmes/"+id)
	if respBusca.StatusCode != http.StatusOK {
		t.Fatalf("o filme criado deveria ser legível: status %d", respBusca.StatusCode)
	}
	if decodificarFilme(t, corpoBusca)["titulo"] != "Novo Filme" {
		t.Fatal("o filme lido não é o que foi criado")
	}
}

func TestEscritaDeFilmeExigeCredencial(t *testing.T) {
	casos := []struct {
		metodo  string
		caminho string
		corpo   string
	}{
		{http.MethodPost, "/api/v1/filmes", corpoFilmeValido},
		{http.MethodPut, "/api/v1/filmes/c394c8b3-76a1-4328-b803-02f5923b7a15", corpoFilmeValido},
		{http.MethodDelete, "/api/v1/filmes/c394c8b3-76a1-4328-b803-02f5923b7a15", ""},
	}
	for _, caso := range casos {
		t.Run(caso.metodo, func(t *testing.T) {
			s := montarComFilmes(t, catalogoDeTeste())
			resp, corpo := requisitar(t, s, caso.metodo, caso.caminho, "", caso.corpo)

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status %d, esperava 401", resp.StatusCode)
			}
			if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "nao-autenticado") {
				t.Fatalf("type inesperado: %s", p.Type)
			}
		})
	}
}

func TestPostFilmeRecusaCorpoInvalido(t *testing.T) {
	casos := map[string]string{
		"titulo vazio":        `{"titulo":"  ","duracao_minutos":120,"classificacao_etaria":"12 anos","genero":"Drama"}`,
		"duracao ausente":     `{"titulo":"X","classificacao_etaria":"12 anos","genero":"Drama"}`,
		"status desconhecido": `{"titulo":"X","duracao_minutos":10,"classificacao_etaria":"L","genero":"Drama","status":"ARQUIVADO"}`,
		"campo desconhecido":  `{"titulo":"X","duracao_minutos":10,"classificacao_etaria":"L","genero":"Drama","diretor":"Y"}`,
		"json quebrado":       `{`,
	}
	for nome, corpoEnviado := range casos {
		t.Run(nome, func(t *testing.T) {
			s := montarComFilmes(t, nil)
			resp, corpo := requisitar(t, s, http.MethodPost, "/api/v1/filmes", "token-bom", corpoEnviado)

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status %d, esperava 400 (corpo: %s)", resp.StatusCode, corpo)
			}
			if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "corpo-invalido") {
				t.Fatalf("type inesperado: %s", p.Type)
			}
		})
	}
}

func TestPutFilmeSubstituiOFilmeInteiro(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	const id = "c394c8b3-76a1-4328-b803-02f5923b7a15"
	novo := `{"titulo":"Duna: Parte 3","duracao_minutos":170,` +
		`"classificacao_etaria":"14 anos","genero":"Ficção Científica","status":"BREVE"}`

	resp, corpo := requisitar(t, s, http.MethodPut, "/api/v1/filmes/"+id, "token-bom", novo)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}

	filme := decodificarFilme(t, corpo)
	if filme["id"] != id || filme["titulo"] != "Duna: Parte 3" || filme["status"] != "BREVE" {
		t.Fatalf("filme não substituído: %v", filme)
	}
	if _, presente := filme["sinopse"]; presente {
		t.Error("campo omitido no PUT deveria ficar ausente: a substituição é total")
	}
}

func TestPutFilmeInexistenteDevolve404(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := requisitar(t, s, http.MethodPut,
		"/api/v1/filmes/a1b2c3d4-0000-4000-8000-999999999999", "token-bom", corpoFilmeValido)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "filme-nao-encontrado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestDeleteFilmeTiraDoCartazSemApagar(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	const id = "c394c8b3-76a1-4328-b803-02f5923b7a15"

	resp, corpo := requisitar(t, s, http.MethodDelete, "/api/v1/filmes/"+id, "token-bom", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, esperava 204 (corpo: %s)", resp.StatusCode, corpo)
	}
	if len(corpo) != 0 {
		t.Errorf("204 não deveria ter corpo, veio %s", corpo)
	}

	_, corpoLista := obter(t, s, "/api/v1/filmes")
	for _, item := range decodificarEnvelope(t, corpoLista).Itens {
		if item["id"] == id {
			t.Fatal("o filme removido não deveria aparecer na listagem pública")
		}
	}

	_, corpoBusca := obter(t, s, "/api/v1/filmes/"+id)
	if decodificarFilme(t, corpoBusca)["status"] != "FORA_DE_CARTAZ" {
		t.Fatal("a remoção é lógica: o filme deveria seguir legível como FORA_DE_CARTAZ")
	}
}

func TestDeleteFilmeInexistenteDevolve404(t *testing.T) {
	s := montarComFilmes(t, catalogoDeTeste())
	resp, corpo := requisitar(t, s, http.MethodDelete,
		"/api/v1/filmes/a1b2c3d4-0000-4000-8000-999999999999", "token-bom", "")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	decodificarProblem(t, resp, corpo)
}
