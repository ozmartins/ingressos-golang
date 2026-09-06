package contract

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

const (
	salaID          = "c1c2c3c4-0000-4000-8000-000000000001"
	caminhoDasSalas = "/api/v1/cinemas/" + cinemaID + "/salas"
	corpoSalaValido = `{"numero":7,"tipo_tela":"IMAX","capacidade_total":180}`
)

func salaDeTeste() catalogo.Sala {
	return catalogo.Sala{ID: salaID, CinemaID: cinemaID, Numero: 3,
		TipoTela: catalogo.TelaIMAX, CapacidadeTotal: 120, Ativo: true}
}

func montarComSalas(t *testing.T, itens []catalogo.Sala) *ambiente {
	t.Helper()
	return montar(t, func(a *ambiente) {
		a.cinemas.itens = redeDeTeste()
		a.salas.itens = itens
	})
}

func decodificarSala(t *testing.T, corpo []byte) map[string]any {
	t.Helper()
	var sala map[string]any
	if err := json.Unmarshal(corpo, &sala); err != nil {
		t.Fatalf("resposta não é uma sala JSON: %v (corpo: %s)", err, corpo)
	}
	return sala
}

func TestGetSalasListaAsSalasDoCinema(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	resp, corpo := obter(t, amb.servidor, caminhoDasSalas)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	for _, campo := range []string{"id", "cinema_id", "numero", "tipo_tela", "capacidade_total", "ativo"} {
		if _, ok := e.Itens[0][campo]; !ok {
			t.Errorf("campo obrigatório %q ausente", campo)
		}
	}
}

func TestGetSalasFiltraPorAtivo(t *testing.T) {
	inativa := salaDeTeste()
	inativa.ID, inativa.Numero, inativa.Ativo = "c1c2c3c4-0000-4000-8000-000000000002", 4, false
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste(), inativa})

	_, corpo := obter(t, amb.servidor, caminhoDasSalas+"?ativo=true")
	if e := decodificarEnvelope(t, corpo); len(e.Itens) != 1 || e.Itens[0]["ativo"] != true {
		t.Fatalf("o filtro deveria devolver só a sala ativa: %v", e.Itens)
	}
	_, corpoTodas := obter(t, amb.servidor, caminhoDasSalas)
	if e := decodificarEnvelope(t, corpoTodas); len(e.Itens) != 2 {
		t.Fatalf("sem filtro, esperava as duas salas, obteve %d", len(e.Itens))
	}
}

func TestGetSalaPorIDDevolveASala(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	resp, corpo := obter(t, amb.servidor, caminhoDasSalas+"/"+salaID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	if sala := decodificarSala(t, corpo); sala["numero"] != float64(3) {
		t.Fatalf("sala inesperada: %v", sala)
	}
}

func TestGetSalaDeOutroCinemaDevolve404DeSala(t *testing.T) {
	deOutroCinema := salaDeTeste()
	deOutroCinema.CinemaID = "b1b2c3d4-0000-4000-8000-000000000999"
	amb := montarComSalas(t, []catalogo.Sala{deOutroCinema})

	resp, corpo := obter(t, amb.servidor, caminhoDasSalas+"/"+salaID)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "sala-nao-encontrada") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestGetSalaComSalaIDMalformadoDevolve400(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	resp, corpo := obter(t, amb.servidor, caminhoDasSalas+"/nao-e-uuid")

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, esperava 400", resp.StatusCode)
	}
	decodificarProblem(t, resp, corpo)
}

func TestPostSalaCriaEDevolveLocation(t *testing.T) {
	amb := montarComSalas(t, nil)
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, caminhoDasSalas, "token-bom", corpoSalaValido)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	sala := decodificarSala(t, corpo)
	id, _ := sala["id"].(string)
	if id == "" {
		t.Fatal("o serviço deveria gerar o id da sala")
	}
	if local := resp.Header.Get("Location"); local != caminhoDasSalas+"/"+id {
		t.Fatalf("Location = %q, esperava o caminho da sala criada", local)
	}
	if sala["cinema_id"] != cinemaID {
		t.Fatalf("cinema_id = %v, esperava o do caminho", sala["cinema_id"])
	}
	if sala["ativo"] != true {
		t.Fatalf("sem `ativo` no corpo, a sala deveria nascer ativa, veio %v", sala["ativo"])
	}

	respBusca, corpoBusca := obter(t, amb.servidor, caminhoDasSalas+"/"+id)
	if respBusca.StatusCode != http.StatusOK {
		t.Fatalf("a sala criada deveria ser legível: status %d (corpo: %s)", respBusca.StatusCode, corpoBusca)
	}
}

func TestPostSalaSemTokenDevolve401(t *testing.T) {
	amb := montarComSalas(t, nil)
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, caminhoDasSalas, "", corpoSalaValido)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, esperava 401", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "nao-autenticado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPostSalaRecusaCorpoInvalido(t *testing.T) {
	amb := montarComSalas(t, nil)
	casos := map[string]string{
		"sem numero":         `{"tipo_tela":"IMAX","capacidade_total":180}`,
		"numero zero":        `{"numero":0,"tipo_tela":"IMAX","capacidade_total":180}`,
		"tela desconhecida":  `{"numero":7,"tipo_tela":"4DX","capacidade_total":180}`,
		"capacidade zero":    `{"numero":7,"tipo_tela":"IMAX","capacidade_total":0}`,
		"campo desconhecido": `{"numero":7,"tipo_tela":"IMAX","capacidade_total":180,"cinema_id":"x"}`,
		"json quebrado":      `{"numero":`,
	}
	for nome, corpoPedido := range casos {
		t.Run(nome, func(t *testing.T) {
			resp, corpo := requisitar(t, amb.servidor, http.MethodPost, caminhoDasSalas, "token-bom", corpoPedido)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status %d, esperava 400 (corpo: %s)", resp.StatusCode, corpo)
			}
			if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "corpo-invalido") {
				t.Fatalf("type inesperado: %s", p.Type)
			}
		})
	}
}

func TestPostSalaEmCinemaInexistenteDevolve404DeCinema(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.cinemas.existe = false })
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, caminhoDasSalas, "token-bom", corpoSalaValido)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "cinema-nao-encontrado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPostSalaComNumeroJaUsadoDevolve409(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	repetida := `{"numero":3,"tipo_tela":"2D","capacidade_total":90}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, caminhoDasSalas, "token-bom", repetida)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, esperava 409 (corpo: %s)", resp.StatusCode, corpo)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "conflito") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPutSalaSubstituiASala(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	novo := `{"numero":9,"tipo_tela":"VIP","capacidade_total":60}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, caminhoDasSalas+"/"+salaID, "token-bom", novo)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	sala := decodificarSala(t, corpo)
	if sala["numero"] != float64(9) || sala["tipo_tela"] != "VIP" {
		t.Fatalf("sala não foi substituída: %v", sala)
	}

	_, corpoBusca := obter(t, amb.servidor, caminhoDasSalas+"/"+salaID)
	if relida := decodificarSala(t, corpoBusca); relida["capacidade_total"] != float64(60) {
		t.Fatalf("a substituição não persistiu: %v", relida)
	}
}

func TestPutSalaPodeManterOProprioNumero(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	mesmoNumero := `{"numero":3,"tipo_tela":"2D","capacidade_total":90}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, caminhoDasSalas+"/"+salaID, "token-bom", mesmoNumero)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("a sala deveria poder manter o próprio número: status %d (corpo: %s)", resp.StatusCode, corpo)
	}
}

func TestPutSalaDeOutroCinemaDevolve404(t *testing.T) {
	deOutroCinema := salaDeTeste()
	deOutroCinema.CinemaID = "b1b2c3d4-0000-4000-8000-000000000999"
	amb := montarComSalas(t, []catalogo.Sala{deOutroCinema})

	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, caminhoDasSalas+"/"+salaID, "token-bom", corpoSalaValido)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404 (corpo: %s)", resp.StatusCode, corpo)
	}
}

func TestDeleteSalaDesativaESomeDaListagem(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	resp, corpo := requisitar(t, amb.servidor, http.MethodDelete, caminhoDasSalas+"/"+salaID, "token-bom", "")

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, esperava 204 (corpo: %s)", resp.StatusCode, corpo)
	}
	if len(corpo) != 0 {
		t.Fatalf("204 não deveria ter corpo, veio %s", corpo)
	}

	_, corpoLista := obter(t, amb.servidor, caminhoDasSalas+"?ativo=true")
	if e := decodificarEnvelope(t, corpoLista); len(e.Itens) != 0 {
		t.Fatalf("a sala desativada deveria sair da listagem de ativas: %v", e.Itens)
	}

	respBusca, corpoBusca := obter(t, amb.servidor, caminhoDasSalas+"/"+salaID)
	if respBusca.StatusCode != http.StatusOK {
		t.Fatalf("a sala desativada deveria seguir legível pelo id: status %d", respBusca.StatusCode)
	}
	if sala := decodificarSala(t, corpoBusca); sala["ativo"] != false {
		t.Fatalf("a sala deveria estar inativa: %v", sala)
	}
}

func TestDeleteSalaInexistenteDevolve404(t *testing.T) {
	amb := montarComSalas(t, nil)
	caminho := caminhoDasSalas + "/c1c2c3c4-0000-4000-8000-000000000999"
	resp, corpo := requisitar(t, amb.servidor, http.MethodDelete, caminho, "token-bom", "")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "sala-nao-encontrada") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}
