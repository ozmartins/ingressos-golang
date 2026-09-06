package contract

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
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

const (
	filmeDaSessao     = "c394c8b3-76a1-4328-b803-02f5923b7a15"
	corpoSessaoValido = `{"filme_id":"` + filmeDaSessao + `","sala_id":"` + salaID + `",` +
		`"data_hora_inicio":"2026-09-20T19:30:00Z","idioma":"LEGENDADO","preco_base":"42.50"}`
)

func sessaoDeTeste() catalogo.Sessao {
	return catalogo.Sessao{
		ID:             sessaoID,
		FilmeID:        filmeDaSessao,
		SalaID:         salaID,
		DataHoraInicio: agora().Add(24 * time.Hour),
		Idioma:         catalogo.Legendado,
		PrecoBase:      catalogo.DinheiroDeCentavos(4200),
		Status:         catalogo.SessaoAgendada,
	}
}

// A escrita de uma sessão consulta o filme e a sala: as duas precisam existir no
// ambiente para o caminho felizardo passar.
func montarComSessoes(t *testing.T, itens []catalogo.Sessao) *ambiente {
	t.Helper()
	return montar(t, func(a *ambiente) {
		a.sessoes.itens = itens
		a.salas.itens = []catalogo.Sala{salaDeTeste()}
		a.filmes.itens = []catalogo.Filme{{ID: filmeDaSessao, Titulo: "Duna: Parte 2",
			DuracaoMinutos: 166, Status: catalogo.StatusEmCartaz}}
	})
}

func decodificarSessao(t *testing.T, corpo []byte) map[string]any {
	t.Helper()
	var sessao map[string]any
	if err := json.Unmarshal(corpo, &sessao); err != nil {
		t.Fatalf("resposta não é uma sessão JSON: %v (corpo: %s)", err, corpo)
	}
	return sessao
}

func TestGetSessaoPorIDDevolveARepresentacaoGravada(t *testing.T) {
	amb := montarComSessoes(t, []catalogo.Sessao{sessaoDeTeste()})
	resp, corpo := obter(t, amb.servidor, "/api/v1/sessoes/"+sessaoID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	sessao := decodificarSessao(t, corpo)
	for _, campo := range []string{"id", "filme_id", "sala_id", "data_hora_inicio", "idioma", "preco_base", "status"} {
		if _, ok := sessao[campo]; !ok {
			t.Errorf("campo obrigatório %q ausente", campo)
		}
	}
	// A grade resolve o filme e o cinema; o recurso, não.
	if _, presente := sessao["filme_titulo"]; presente {
		t.Error("o recurso não deveria trazer filme_titulo: isso é da grade")
	}
}

func TestGetSessaoInexistenteDevolve404DeSessao(t *testing.T) {
	amb := montarComSessoes(t, []catalogo.Sessao{sessaoDeTeste()})
	resp, corpo := obter(t, amb.servidor, "/api/v1/sessoes/f781a9b2-11e2-4f81-a901-8890bc999999")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "sessao-nao-encontrada") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPostSessaoPublicaEDevolveLocation(t *testing.T) {
	amb := montarComSessoes(t, nil)
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/sessoes", "token-bom", corpoSessaoValido)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	sessao := decodificarSessao(t, corpo)
	id, _ := sessao["id"].(string)
	if id == "" {
		t.Fatal("o serviço deveria gerar o id da sessão")
	}
	if local := resp.Header.Get("Location"); local != "/api/v1/sessoes/"+id {
		t.Fatalf("Location = %q, esperava o caminho da sessão criada", local)
	}
	if sessao["status"] != "AGENDADA" {
		t.Fatalf("sem `status` no corpo, a sessão deveria nascer agendada, veio %v", sessao["status"])
	}
	if sessao["preco_base"] != "42.50" {
		t.Fatalf("preco_base deveria voltar como texto exato, veio %v", sessao["preco_base"])
	}

	respBusca, corpoBusca := obter(t, amb.servidor, "/api/v1/sessoes/"+id)
	if respBusca.StatusCode != http.StatusOK {
		t.Fatalf("a sessão criada deveria ser legível: status %d (corpo: %s)", respBusca.StatusCode, corpoBusca)
	}
}

// Criar a sessão pela API enfileira o anúncio dela, com a planta da sala já
// expandida. A resposta não espera pela publicação — o fato fica na caixa.
func TestPostSessaoEnfileiraOAnuncio(t *testing.T) {
	amb := montarComSessoes(t, nil)
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/sessoes", "token-bom", corpoSessaoValido)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	if len(amb.sessoes.fatos) != 1 {
		t.Fatalf("enfileirou %d fato(s), esperava 1", len(amb.sessoes.fatos))
	}

	fato := amb.sessoes.fatos[0]
	if fato.RoutingKey != usecase.RoutingKeySessaoCriada {
		t.Errorf("routing key = %q", fato.RoutingKey)
	}
	if fato.MessageID != decodificarSessao(t, corpo)["id"] {
		t.Errorf("o message_id deveria ser o id da sessão, veio %q", fato.MessageID)
	}

	var evento usecase.EventoSessaoCriada
	if err := json.Unmarshal(fato.Payload, &evento); err != nil {
		t.Fatalf("o corpo do fato não é JSON válido: %v", err)
	}
	if esperado := salaDeTeste().CapacidadeTotal(); len(evento.Poltronas) != esperado {
		t.Fatalf("anunciou %d poltronas, esperava a capacidade da sala (%d)", len(evento.Poltronas), esperado)
	}
}

// Uma sessão recusada não anuncia nada: nada foi criado.
func TestPostSessaoRecusadaNaoEnfileiraAnuncio(t *testing.T) {
	amb := montar(t, func(a *ambiente) {
		a.salas.itens = []catalogo.Sala{salaDeTeste()}
		a.filmes.itens = []catalogo.Filme{{ID: filmeDaSessao, Titulo: "Duna: Parte 2",
			DuracaoMinutos: 166, Status: catalogo.StatusEmCartaz}}
		a.sessoes.salaOcupada = true
	})
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/sessoes", "token-bom", corpoSessaoValido)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, esperava 409 (corpo: %s)", resp.StatusCode, corpo)
	}
	if len(amb.sessoes.fatos) != 0 {
		t.Fatalf("não deveria anunciar sessão que não existe: %+v", amb.sessoes.fatos)
	}
}

func TestPostSessaoSemTokenDevolve401(t *testing.T) {
	amb := montarComSessoes(t, nil)
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/sessoes", "", corpoSessaoValido)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, esperava 401", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "nao-autenticado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPostSessaoRecusaCorpoInvalido(t *testing.T) {
	amb := montarComSessoes(t, nil)
	casos := map[string]string{
		"sem filme": `{"sala_id":"` + salaID + `","data_hora_inicio":"2026-09-20T19:30:00Z",` +
			`"idioma":"LEGENDADO","preco_base":"42.50"}`,
		"início fora do RFC 3339": `{"filme_id":"` + filmeDaSessao + `","sala_id":"` + salaID + `",` +
			`"data_hora_inicio":"20/09/2026 19:30","idioma":"LEGENDADO","preco_base":"42.50"}`,
		"idioma desconhecido": `{"filme_id":"` + filmeDaSessao + `","sala_id":"` + salaID + `",` +
			`"data_hora_inicio":"2026-09-20T19:30:00Z","idioma":"ORIGINAL","preco_base":"42.50"}`,
		"preço com três casas": `{"filme_id":"` + filmeDaSessao + `","sala_id":"` + salaID + `",` +
			`"data_hora_inicio":"2026-09-20T19:30:00Z","idioma":"LEGENDADO","preco_base":"42.505"}`,
		"preço como número": `{"filme_id":"` + filmeDaSessao + `","sala_id":"` + salaID + `",` +
			`"data_hora_inicio":"2026-09-20T19:30:00Z","idioma":"LEGENDADO","preco_base":42.50}`,
		"campo desconhecido": `{"filme_id":"` + filmeDaSessao + `","sala_id":"` + salaID + `",` +
			`"data_hora_inicio":"2026-09-20T19:30:00Z","idioma":"LEGENDADO","preco_base":"42.50","sala_numero":3}`,
		"json quebrado": `{"filme_id":`,
	}
	for nome, corpoPedido := range casos {
		t.Run(nome, func(t *testing.T) {
			resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/sessoes", "token-bom", corpoPedido)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status %d, esperava 400 (corpo: %s)", resp.StatusCode, corpo)
			}
			if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "corpo-invalido") {
				t.Fatalf("type inesperado: %s", p.Type)
			}
		})
	}
}

func TestPostSessaoDeFilmeInexistenteDevolve404DeFilme(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.salas.itens = []catalogo.Sala{salaDeTeste()} })
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/sessoes", "token-bom", corpoSessaoValido)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404 (corpo: %s)", resp.StatusCode, corpo)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "filme-nao-encontrado") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPostSessaoEmSalaInexistenteDevolve404DeSala(t *testing.T) {
	amb := montar(t, func(a *ambiente) {
		a.filmes.itens = []catalogo.Filme{{ID: filmeDaSessao, DuracaoMinutos: 166}}
	})
	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/sessoes", "token-bom", corpoSessaoValido)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404 (corpo: %s)", resp.StatusCode, corpo)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "sala-nao-encontrada") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPostSessaoEmSalaOcupadaDevolve409(t *testing.T) {
	amb := montarComSessoes(t, nil)
	amb.sessoes.salaOcupada = true

	resp, corpo := requisitar(t, amb.servidor, http.MethodPost, "/api/v1/sessoes", "token-bom", corpoSessaoValido)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, esperava 409 (corpo: %s)", resp.StatusCode, corpo)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "conflito") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestPutSessaoSubstituiASessao(t *testing.T) {
	amb := montarComSessoes(t, []catalogo.Sessao{sessaoDeTeste()})
	novo := `{"filme_id":"` + filmeDaSessao + `","sala_id":"` + salaID + `",` +
		`"data_hora_inicio":"2026-09-21T21:00:00Z","idioma":"DUBLADO","preco_base":"55.00"}`
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, "/api/v1/sessoes/"+sessaoID, "token-bom", novo)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d (corpo: %s)", resp.StatusCode, corpo)
	}
	sessao := decodificarSessao(t, corpo)
	if sessao["idioma"] != "DUBLADO" || sessao["preco_base"] != "55.00" {
		t.Fatalf("sessão não foi substituída: %v", sessao)
	}

	_, corpoBusca := obter(t, amb.servidor, "/api/v1/sessoes/"+sessaoID)
	if relida := decodificarSessao(t, corpoBusca); relida["data_hora_inicio"] != "2026-09-21T21:00:00Z" {
		t.Fatalf("a substituição não persistiu: %v", relida)
	}
}

func TestPutSessaoInexistenteDevolve404(t *testing.T) {
	amb := montarComSessoes(t, nil)
	caminho := "/api/v1/sessoes/f781a9b2-11e2-4f81-a901-8890bc999999"
	resp, corpo := requisitar(t, amb.servidor, http.MethodPut, caminho, "token-bom", corpoSessaoValido)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404 (corpo: %s)", resp.StatusCode, corpo)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "sessao-nao-encontrada") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestDeleteSessaoCancelaESaiDaGrade(t *testing.T) {
	amb := montarComSessoes(t, []catalogo.Sessao{sessaoDeTeste()})
	resp, corpo := requisitar(t, amb.servidor, http.MethodDelete, "/api/v1/sessoes/"+sessaoID, "token-bom", "")

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, esperava 204 (corpo: %s)", resp.StatusCode, corpo)
	}
	if len(corpo) != 0 {
		t.Fatalf("204 não deveria ter corpo, veio %s", corpo)
	}

	respBusca, corpoBusca := obter(t, amb.servidor, "/api/v1/sessoes/"+sessaoID)
	if respBusca.StatusCode != http.StatusOK {
		t.Fatalf("a sessão cancelada deveria seguir legível pelo id: status %d", respBusca.StatusCode)
	}
	if sessao := decodificarSessao(t, corpoBusca); sessao["status"] != "CANCELADA" {
		t.Fatalf("a sessão deveria estar cancelada: %v", sessao)
	}
}

func TestDeleteSessaoInexistenteDevolve404(t *testing.T) {
	amb := montarComSessoes(t, nil)
	caminho := "/api/v1/sessoes/f781a9b2-11e2-4f81-a901-8890bc999999"
	resp, corpo := requisitar(t, amb.servidor, http.MethodDelete, caminho, "token-bom", "")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, esperava 404", resp.StatusCode)
	}
	if p := decodificarProblem(t, resp, corpo); !strings.HasSuffix(p.Type, "sessao-nao-encontrada") {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}
