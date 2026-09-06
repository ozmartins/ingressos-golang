package contract

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

const cinemaID = "b1b2c3d4-0000-4000-8000-000000000001"

const corpoCinemaValido = `{"nome":"CineMark - Beiramar","cidade":"Florianópolis",` +
	`"estado":"SC","endereco":"Rua Y, 200"}`

func redeDeTeste() []catalogo.Cinema {
	return []catalogo.Cinema{
		{ID: cinemaID, Nome: "CineMark - Shopping Centro", Cidade: "Florianópolis",
			Estado: "SC", Endereco: "Rua X, 100", Ativo: true},
	}
}

func montarComCinemas(t *testing.T, itens []catalogo.Cinema) *ambiente {
	t.Helper()
	return montar(t, func(a *ambiente) { a.cinemas.itens = itens })
}

func decodificarCinema(t *testing.T, corpo []byte) map[string]any {
	t.Helper()
	var cinema map[string]any
	if err := json.Unmarshal(corpo, &cinema); err != nil {
		t.Fatalf("resposta não é um cinema JSON: %v (corpo: %s)", err, corpo)
	}
	return cinema
}

func TestGetCinemasRetornaLocalizacao(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())
	resp, corpo := obter(t, amb.servidor, "/api/v1/cinemas")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	for _, campo := range []string{"id", "nome", "cidade", "estado", "endereco", "ativo"} {
		if _, ok := e.Itens[0][campo]; !ok {
			t.Errorf("campo obrigatório %q ausente", campo)
		}
	}
	if e.Itens[0]["estado"] != "SC" {
		t.Errorf("estado inesperado: %v", e.Itens[0]["estado"])
	}
}

func TestGetCinemasRecusaAtivoDesconhecido(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())
	resp, corpo := obter(t, amb.servidor, "/api/v1/cinemas?ativo=sim")

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, esperava 400", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "parametro-invalido") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestGetCinemaPorIDDevolveOCinema(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())
	resp, corpo := obter(t, amb.servidor, "/api/v1/cinemas/"+cinemaID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if cinema := decodificarCinema(t, corpo); cinema["nome"] != "CineMark - Shopping Centro" {
		t.Fatalf("cinema inesperado: %v", cinema)
	}
}

func TestGetCinemaInexistenteDevolve404DeCinema(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())
	resp, corpo := obter(t, amb.servidor, "/api/v1/cinemas/b1b2c3d4-0000-4000-8000-999999999999")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "cinema-nao-encontrado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestGetCinemaComIDMalformadoDevolve400(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())
	resp, corpo := obter(t, amb.servidor, "/api/v1/cinemas/nao-e-uuid")

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, esperava 400", resp.StatusCode)
	}
	decodificarProblem(t, resp, corpo)
}

func TestPostCinemaCriaEDevolveLocation(t *testing.T) {
	amb := montarComCinemas(t, nil)
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/cinemas", "token-bom", corpoCinemaValido)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	cinema := decodificarCinema(t, corpo)
	id, _ := cinema["id"].(string)
	if id == "" {
		t.Fatal("o serviço deveria gerar o id do cinema")
	}
	if local := resp.Header.Get("Location"); local != "/api/v1/cinemas/"+id {
		t.Fatalf("Location = %q, esperava o caminho do cinema criado", local)
	}
	if cinema["ativo"] != true {
		t.Fatalf("sem `ativo` no corpo, o cinema deveria nascer ativo, veio %v", cinema["ativo"])
	}

	respBusca, corpoBusca := obter(t, amb.servidor, "/api/v1/cinemas/"+id)
	if respBusca.StatusCode != http.StatusOK {
		t.Fatalf("o cinema criado deveria ser legível: status %d", respBusca.StatusCode)
	}
	if decodificarCinema(t, corpoBusca)["nome"] != "CineMark - Beiramar" {
		t.Fatal("o cinema lido não é o que foi criado")
	}
}

func TestEscritaDeCinemaExigeCredencial(t *testing.T) {
	casos := []struct {
		metodo  string
		caminho string
		corpo   string
	}{
		{http.MethodPost, "/api/v1/cinemas", corpoCinemaValido},
		{http.MethodPut, "/api/v1/cinemas/" + cinemaID, corpoCinemaValido},
		{http.MethodDelete, "/api/v1/cinemas/" + cinemaID, ""},
	}
	for _, caso := range casos {
		t.Run(caso.metodo, func(t *testing.T) {
			amb := montarComCinemas(t, redeDeTeste())
			resp, corpo := requisitar(t, amb.servidor, caso.metodo, caso.caminho, "", caso.corpo)

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status %d, esperava 401", resp.StatusCode)
			}
			if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "nao-autenticado") {
				t.Fatalf("type inesperado: %s", p.Type)
			}
		})
	}
}

func TestPostCinemaRecusaCorpoInvalido(t *testing.T) {
	casos := map[string]string{
		"nome vazio":         `{"nome":"  ","cidade":"Florianópolis","estado":"SC","endereco":"Rua X"}`,
		"cidade ausente":     `{"nome":"X","estado":"SC","endereco":"Rua X"}`,
		"estado com 1 letra": `{"nome":"X","cidade":"Florianópolis","estado":"S","endereco":"Rua X"}`,
		"estado com 3":       `{"nome":"X","cidade":"Florianópolis","estado":"SCA","endereco":"Rua X"}`,
		"endereco vazio":     `{"nome":"X","cidade":"Florianópolis","estado":"SC","endereco":"   "}`,
		"campo desconhecido": `{"nome":"X","cidade":"Florianópolis","estado":"SC","endereco":"Rua X","cep":"88000-000"}`,
		"json quebrado":      `{`,
	}
	for nome, corpoEnviado := range casos {
		t.Run(nome, func(t *testing.T) {
			amb := montarComCinemas(t, nil)
			resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/cinemas", "token-bom", corpoEnviado)

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status %d, esperava 400 (corpo: %s)", resp.StatusCode, corpo)
			}
			if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "corpo-invalido") {
				t.Fatalf("type inesperado: %s", p.Type)
			}
		})
	}
}

func TestPutCinemaSubstituiOCinemaInteiro(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())
	novo := `{"nome":"CineMark - Centro","cidade":"São José","estado":"sc","endereco":"Rua Z, 300"}`

	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, "/api/v1/cinemas/"+cinemaID, "token-bom", novo)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}

	cinema := decodificarCinema(t, corpo)
	if cinema["id"] != cinemaID || cinema["nome"] != "CineMark - Centro" || cinema["cidade"] != "São José" {
		t.Fatalf("cinema não substituído: %v", cinema)
	}
	if cinema["estado"] != "SC" {
		t.Errorf("a sigla deveria ser normalizada para maiúsculas, veio %v", cinema["estado"])
	}
	if cinema["ativo"] != true {
		t.Errorf("sem `ativo` no corpo, o cinema deveria ficar ativo, veio %v", cinema["ativo"])
	}
}

func TestPutCinemaInexistenteDevolve404(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut,
		"/api/v1/cinemas/b1b2c3d4-0000-4000-8000-999999999999", "token-bom", corpoCinemaValido)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "cinema-nao-encontrado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestDeleteCinemaDesativaSemApagar(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())

	resp, corpo := requisitar(t, amb.servidor, http.MethodDelete, "/api/v1/cinemas/"+cinemaID, "token-bom", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, esperava 204 (corpo: %s)", resp.StatusCode, corpo)
	}
	if len(corpo) != 0 {
		t.Errorf("204 não deveria ter corpo, veio %s", corpo)
	}

	_, corpoLista := obter(t, amb.servidor, "/api/v1/cinemas")
	for _, item := range decodificarEnvelope(t, corpoLista).Itens {
		if item["id"] == cinemaID {
			t.Fatal("o cinema removido não deveria aparecer na listagem pública")
		}
	}

	_, corpoBusca := obter(t, amb.servidor, "/api/v1/cinemas/"+cinemaID)
	if decodificarCinema(t, corpoBusca)["ativo"] != false {
		t.Fatal("a remoção é lógica: o cinema deveria seguir legível como inativo")
	}

	_, corpoInativos := obter(t, amb.servidor, "/api/v1/cinemas?ativo=false")
	if decodificarEnvelope(t, corpoInativos).Pagina.Total != 1 {
		t.Fatal("o filtro ativo=false deveria alcançar o cinema desativado")
	}
}

func TestDeleteCinemaInexistenteDevolve404(t *testing.T) {
	amb := montarComCinemas(t, redeDeTeste())
	resp, corpo := requisitar(t, amb.servidor, http.MethodDelete,
		"/api/v1/cinemas/b1b2c3d4-0000-4000-8000-999999999999", "token-bom", "")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	decodificarProblem(t, resp, corpo)
}
