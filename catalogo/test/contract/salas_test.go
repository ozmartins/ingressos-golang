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
	outroCinemaID   = "b1b2c3d4-0000-4000-8000-000000000999"
	caminhoDasSalas = "/api/v1/salas"
	// Três fileiras somando 26 lugares: é essa soma que a resposta precisa
	// devolver em `capacidade_total`, já que o corpo não a informa.
	fileirasValidas = `"fileiras":[{"fileira":"A","assentos":12},` +
		`{"fileira":"F","assentos":8,"tipo":"PCD"},` +
		`{"fileira":"J","assentos":6,"tipo":"NAMORADEIRA"}]`
	corpoSalaValido = `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX",` + fileirasValidas + `}`
)

func salaDeTeste() catalogo.Sala {
	return catalogo.Sala{ID: salaID, CinemaID: cinemaID, Numero: 3,
		TipoTela: catalogo.TelaIMAX, Ativo: true,
		Layout: catalogo.LayoutSala{Fileiras: []catalogo.Fileira{
			{Letra: "A", Assentos: 60, Tipo: catalogo.PoltronaNormal},
			{Letra: "B", Assentos: 60, Tipo: catalogo.PoltronaNormal},
		}}}
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

func TestGetSalasFiltraPorCinema(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	resp, corpo := obter(t, amb.servidor, caminhoDasSalas+"?cinema_id="+cinemaID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	for _, campo := range []string{"id", "cinema_id", "numero", "tipo_tela", "fileiras", "capacidade_total", "ativo"} {
		if _, ok := e.Itens[0][campo]; !ok {
			t.Errorf("campo obrigatório %q ausente", campo)
		}
	}
}

// Sem `cinema_id` a listagem é da rede inteira: a sala deixou de viver dentro
// do caminho do cinema, e o filtro passou a ser opcional como o das sessões.
func TestGetSalasSemCinemaIDListaTodaARede(t *testing.T) {
	deOutroCinema := salaDeTeste()
	deOutroCinema.ID, deOutroCinema.CinemaID = "c1c2c3c4-0000-4000-8000-000000000002", outroCinemaID
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste(), deOutroCinema})

	_, corpo := obter(t, amb.servidor, caminhoDasSalas)
	if e := decodificarEnvelope(t, corpo); len(e.Itens) != 2 {
		t.Fatalf("sem cinema_id, esperava as salas dos dois cinemas, obteve %d", len(e.Itens))
	}

	_, corpoDoCinema := obter(t, amb.servidor, caminhoDasSalas+"?cinema_id="+cinemaID)
	if e := decodificarEnvelope(t, corpoDoCinema); len(e.Itens) != 1 {
		t.Fatalf("com cinema_id, esperava só a sala daquele cinema, obteve %d", len(e.Itens))
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

func TestGetSalasDeCinemaInexistenteDevolve404DeCinema(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.cinemas.existe = false })
	resp, corpo := obter(t, amb.servidor, caminhoDasSalas+"?cinema_id="+cinemaID)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "cinema-nao-encontrado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestGetSalasDeCinemaSemSalasDevolve200Vazio(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.cinemas.existe = true })
	resp, corpo := obter(t, amb.servidor, caminhoDasSalas+"?cinema_id="+cinemaID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	if len(e.Itens) != 0 || e.Pagina.Total != 0 {
		t.Fatalf("esperava página vazia, obteve %+v", e)
	}
}

func TestGetSalasComCinemaIDMalformadoDevolve400(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	resp, corpo := obter(t, amb.servidor, caminhoDasSalas+"?cinema_id=nao-e-uuid")

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, esperava 400", resp.StatusCode)
	}
	decodificarProblem(t, resp, corpo)
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

func TestGetSalaInexistenteDevolve404DeSala(t *testing.T) {
	amb := montarComSalas(t, nil)
	resp, corpo := obter(t, amb.servidor, caminhoDasSalas+"/"+salaID)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "sala-nao-encontrada") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestGetSalaComIDMalformadoDevolve400(t *testing.T) {
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
		t.Fatalf("cinema_id = %v, esperava o do corpo", sala["cinema_id"])
	}
	if sala["ativo"] != true {
		t.Fatalf("sem `ativo` no corpo, a sala deveria nascer ativa, veio %v", sala["ativo"])
	}

	respBusca, corpoBusca := obter(t, amb.servidor, caminhoDasSalas+"/"+id)
	if respBusca.StatusCode != http.StatusOK {
		t.Fatalf("a sala criada deveria ser legível: status %d (corpo: %s)", respBusca.StatusCode, corpoBusca)
	}
}

// A capacidade não é digitada: ela sai da planta. E a planta volta ordenada por
// fileira, independentemente da ordem em que o corpo a listou.
func TestPostSalaDerivaACapacidadeDoLayout(t *testing.T) {
	amb := montarComSalas(t, nil)
	foraDeOrdem := `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX",` +
		`"fileiras":[{"fileira":"J","assentos":6,"tipo":"NAMORADEIRA"},` +
		`{"fileira":"A","assentos":12},{"fileira":"F","assentos":8,"tipo":"PCD"}]}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, caminhoDasSalas, "token-bom", foraDeOrdem)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	sala := decodificarSala(t, corpo)
	if sala["capacidade_total"] != float64(26) {
		t.Fatalf("capacidade_total = %v, esperava a soma 26", sala["capacidade_total"])
	}

	fileiras, ok := sala["fileiras"].([]any)
	if !ok || len(fileiras) != 3 {
		t.Fatalf("fileiras = %v, esperava as três da planta", sala["fileiras"])
	}
	for i, esperada := range []string{"A", "F", "J"} {
		if fileiras[i].(map[string]any)["fileira"] != esperada {
			t.Errorf("fileira %d = %v, esperava %q", i, fileiras[i], esperada)
		}
	}
	// Sem `tipo` no corpo, a fileira nasce normal — o mesmo padrão de `ativo`.
	if fileiras[0].(map[string]any)["tipo"] != "NORMAL" {
		t.Errorf("fileira sem tipo deveria nascer NORMAL: %v", fileiras[0])
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
		"sem cinema_id":   `{"numero":7,"tipo_tela":"IMAX",` + fileirasValidas + `}`,
		"cinema_id vazio": `{"cinema_id":"","numero":7,"tipo_tela":"IMAX",` + fileirasValidas + `}`,
		"cinema_id malformado": `{"cinema_id":"nao-e-uuid","numero":7,"tipo_tela":"IMAX",` +
			fileirasValidas + `}`,
		"sem numero":        `{"cinema_id":"` + cinemaID + `","tipo_tela":"IMAX",` + fileirasValidas + `}`,
		"numero zero":       `{"cinema_id":"` + cinemaID + `","numero":0,"tipo_tela":"IMAX",` + fileirasValidas + `}`,
		"tela desconhecida": `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"4DX",` + fileirasValidas + `}`,
		"sem fileiras":      `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX"}`,
		"fileiras vazias":   `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX","fileiras":[]}`,
		"fileira repetida": `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX",` +
			`"fileiras":[{"fileira":"A","assentos":10},{"fileira":"a","assentos":8}]}`,
		"fileira com dígito": `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX",` +
			`"fileiras":[{"fileira":"A1","assentos":10}]}`,
		"assentos zero": `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX",` +
			`"fileiras":[{"fileira":"A","assentos":0}]}`,
		"tipo de poltrona desconhecido": `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX",` +
			`"fileiras":[{"fileira":"A","assentos":10,"tipo":"PUFE"}]}`,
		// A capacidade é derivada: quem ainda a mandar leva 400, porque os
		// handlers recusam campo desconhecido.
		"capacidade_total no corpo": `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX",` +
			fileirasValidas + `,"capacidade_total":180}`,
		"campo desconhecido": `{"cinema_id":"` + cinemaID + `","numero":7,"tipo_tela":"IMAX",` +
			fileirasValidas + `,"cor":"azul"}`,
		"json quebrado": `{"numero":`,
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
	repetida := `{"cinema_id":"` + cinemaID + `","numero":3,"tipo_tela":"2D",` + fileirasValidas + `}`
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
	novo := `{"cinema_id":"` + cinemaID + `","numero":9,"tipo_tela":"VIP",` + fileirasValidas + `}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, caminhoDasSalas+"/"+salaID, "token-bom", novo)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	sala := decodificarSala(t, corpo)
	if sala["numero"] != float64(9) || sala["tipo_tela"] != "VIP" {
		t.Fatalf("sala não foi substituída: %v", sala)
	}

	_, corpoBusca := obter(t, amb.servidor, caminhoDasSalas+"/"+salaID)
	if relida := decodificarSala(t, corpoBusca); relida["capacidade_total"] != float64(26) {
		t.Fatalf("a substituição não persistiu: %v", relida)
	}
}

func TestPutSalaPodeManterOProprioNumero(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	mesmoNumero := `{"cinema_id":"` + cinemaID + `","numero":3,"tipo_tela":"2D",` + fileirasValidas + `}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, caminhoDasSalas+"/"+salaID, "token-bom", mesmoNumero)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("a sala deveria poder manter o próprio número: status %d (corpo: %s)", resp.StatusCode, corpo)
	}
}

func TestPutSalaSemCinemaIDMantemOCinema(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	semCinema := `{"numero":9,"tipo_tela":"VIP",` + fileirasValidas + `}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, caminhoDasSalas+"/"+salaID, "token-bom", semCinema)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, esperava 200 (corpo: %s)", resp.StatusCode, corpo)
	}
	if sala := decodificarSala(t, corpo); sala["cinema_id"] != cinemaID {
		t.Fatalf("a sala deveria continuar no cinema atual: %v", sala)
	}
}

// O vínculo com o cinema é do cadastro, não do corpo do PUT: a sala não migra.
func TestPutSalaComOutroCinemaDevolve409(t *testing.T) {
	amb := montarComSalas(t, []catalogo.Sala{salaDeTeste()})
	migrando := `{"cinema_id":"` + outroCinemaID + `","numero":3,"tipo_tela":"2D",` + fileirasValidas + `}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, caminhoDasSalas+"/"+salaID, "token-bom", migrando)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, esperava 409 (corpo: %s)", resp.StatusCode, corpo)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "conflito") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPutSalaInexistenteDevolve404(t *testing.T) {
	amb := montarComSalas(t, nil)
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
