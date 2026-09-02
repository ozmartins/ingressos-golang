package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

const (
	emissorTeste  = "http://keycloak.test/realms/cinema"
	audienciaTest = "cinema-app"
	segredoTeste  = "segredo-de-teste-apenas"
	subTeste      = "11111111-1111-4111-8111-111111111111"
)

type bloqueioFalso struct {
	resultado    usecase.ResultadoBloqueio
	err          error
	usuarioVisto string
	rotulosVisto []string
}

func (b *bloqueioFalso) Executar(_ context.Context, _, usuarioID string, rotulos []string) (usecase.ResultadoBloqueio, error) {
	b.usuarioVisto = usuarioID
	b.rotulosVisto = rotulos
	return b.resultado, b.err
}

type mapaFalso struct {
	poltronas []poltrona.Poltrona
	err       error
}

func (m *mapaFalso) Executar(context.Context, string) ([]poltrona.Poltrona, error) {
	return m.poltronas, m.err
}

func apiDeTeste(b CasoDeUsoBloqueio, m CasoDeUsoMapa) *API {
	chave := func(*jwt.Token) (any, error) { return []byte(segredoTeste), nil }
	return &API{
		Bloqueio: b,
		Mapa:     m,
		Auth:     NovoAutenticadorComChave(chave, emissorTeste, audienciaTest),
		Limite:   10,
	}
}

func tokenValido(t *testing.T) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": emissorTeste,
		"aud": audienciaTest,
		"sub": subTeste,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	assinado, err := tok.SignedString([]byte(segredoTeste))
	if err != nil {
		t.Fatalf("assinando o token de teste: %v", err)
	}
	return assinado
}

func executar(t *testing.T, api *API, metodo, caminho, corpo, token string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	api.Rotas().ServeHTTP(w, r)
	return w
}

func TestBloqueioUsaOSubDoTokenComoUsuario(t *testing.T) {
	b := &bloqueioFalso{resultado: usecase.ResultadoBloqueio{
		Concedido: true,
		Reserva: reserva.Reserva{
			ID:       "22222222-2222-4222-8222-222222222222",
			ExpiraEm: time.Now().Add(10 * time.Minute),
		},
		Mensagem: "reserva criada",
	}}
	api := apiDeTeste(b, &mapaFalso{})

	corpo := `{"poltronas_ids":["A1","A2"],"usuario_id":"outra-pessoa"}`
	w := executar(t, api, http.MethodPost, "/api/v1/sessoes/s1/bloqueios", corpo, tokenValido(t))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, esperado 201. corpo: %s", w.Code, w.Body.String())
	}
	if b.usuarioVisto != subTeste {
		t.Errorf("usuario_id = %q, esperado o sub do token (%q)", b.usuarioVisto, subTeste)
	}
	var resp respostaBloqueio
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decodificando a resposta: %v", err)
	}
	if resp.ReservaID != b.resultado.Reserva.ID {
		t.Errorf("reserva_id = %q, esperado %q", resp.ReservaID, b.resultado.Reserva.ID)
	}
	if _, err := time.Parse(time.RFC3339, resp.ExpiraEm); err != nil {
		t.Errorf("expira_em = %q, não é RFC 3339: %v", resp.ExpiraEm, err)
	}
}

func TestBloqueioSemTokenRecusa401(t *testing.T) {
	b := &bloqueioFalso{}
	api := apiDeTeste(b, &mapaFalso{})

	w := executar(t, api, http.MethodPost, "/api/v1/sessoes/s1/bloqueios", `{"poltronas_ids":["A1"]}`, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperado 401", w.Code)
	}
	if b.usuarioVisto != "" {
		t.Error("o caso de uso foi chamado apesar da credencial ausente")
	}
}

func TestPoltronasIndisponiveisRespondem409(t *testing.T) {
	b := &bloqueioFalso{resultado: usecase.ResultadoBloqueio{
		Concedido: false,
		Mensagem:  "poltronas indisponíveis",
	}}
	api := apiDeTeste(b, &mapaFalso{})

	w := executar(t, api, http.MethodPost, "/api/v1/sessoes/s1/bloqueios", `{"poltronas_ids":["A1"]}`, tokenValido(t))

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, esperado 409", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("Content-Type = %q, esperado application/problem+json", ct)
	}
}

func TestErrosDeDominioViramOStatusEquivalente(t *testing.T) {
	casos := []struct {
		nome     string
		err      error
		esperado int
		tipo     string
	}{
		{"solicitação inválida", shared.ErrSolicitacaoInvalida, http.StatusBadRequest, tipoSolicitacaoInvalida},
		{"limite excedido", shared.ErrLimiteExcedido, http.StatusBadRequest, tipoLimiteExcedido},
		{"sessão não provisionada", shared.ErrSessaoNaoProvisionada, http.StatusUnprocessableEntity, tipoSessaoNaoProvisionada},
		{"poltrona inexistente", shared.ErrPoltronaInexistente, http.StatusUnprocessableEntity, tipoPoltronaInexistente},
		{"sessão desconhecida", shared.ErrSessaoDesconhecida, http.StatusNotFound, tipoSessaoDesconhecida},
		{"dependência indisponível", shared.ErrDependenciaIndisponivel, http.StatusServiceUnavailable, tipoDependencia},
		{"falha não prevista", errors.New("qualquer outra"), http.StatusInternalServerError, tipoInterno},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			api := apiDeTeste(&bloqueioFalso{err: c.err}, &mapaFalso{})
			w := executar(t, api, http.MethodPost, "/api/v1/sessoes/s1/bloqueios", `{"poltronas_ids":["A1"]}`, tokenValido(t))

			if w.Code != c.esperado {
				t.Fatalf("status = %d, esperado %d. corpo: %s", w.Code, c.esperado, w.Body.String())
			}
			var p problem
			if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
				t.Fatalf("decodificando o problem: %v", err)
			}
			if p.Type != prefixoTipo+c.tipo {
				t.Errorf("type = %q, esperado %q", p.Type, prefixoTipo+c.tipo)
			}
		})
	}
}

func TestErroInternoNaoVazaDetalhe(t *testing.T) {
	api := apiDeTeste(&bloqueioFalso{err: errors.New("pq: connection refused em 10.0.0.7:5432")}, &mapaFalso{})

	w := executar(t, api, http.MethodPost, "/api/v1/sessoes/s1/bloqueios", `{"poltronas_ids":["A1"]}`, tokenValido(t))

	if strings.Contains(w.Body.String(), "10.0.0.7") || strings.Contains(w.Body.String(), "pq:") {
		t.Errorf("a resposta vazou detalhe interno: %s", w.Body.String())
	}
}

func TestLimiteExcedidoInformaOTeto(t *testing.T) {
	api := apiDeTeste(&bloqueioFalso{err: shared.ErrLimiteExcedido}, &mapaFalso{})

	w := executar(t, api, http.MethodPost, "/api/v1/sessoes/s1/bloqueios", `{"poltronas_ids":["A1"]}`, tokenValido(t))

	var p problem
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("decodificando o problem: %v", err)
	}
	if p.Metadata["limite"] != "10" {
		t.Errorf("metadata.limite = %q, esperado \"10\"", p.Metadata["limite"])
	}
}

func TestCorpoNaoJSONRecusa400(t *testing.T) {
	api := apiDeTeste(&bloqueioFalso{}, &mapaFalso{})

	w := executar(t, api, http.MethodPost, "/api/v1/sessoes/s1/bloqueios", `nada disso`, tokenValido(t))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, esperado 400", w.Code)
	}
}

func TestMapaDevolveOsMesmosEstadosDoDominio(t *testing.T) {
	m := &mapaFalso{poltronas: []poltrona.Poltrona{
		{Rotulo: "A1", Fileira: "A", Numero: 1, Tipo: poltrona.Normal, Status: poltrona.Livre},
		{Rotulo: "A2", Fileira: "A", Numero: 2, Tipo: poltrona.PCD, Status: poltrona.Reservada},
	}}
	api := apiDeTeste(&bloqueioFalso{}, m)

	w := executar(t, api, http.MethodGet, "/api/v1/sessoes/s1/poltronas", "", tokenValido(t))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200. corpo: %s", w.Code, w.Body.String())
	}
	var resp respostaMapa
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decodificando a resposta: %v", err)
	}
	if resp.SessaoID != "s1" {
		t.Errorf("sessao_id = %q, esperado \"s1\"", resp.SessaoID)
	}
	if len(resp.Poltronas) != 2 {
		t.Fatalf("poltronas = %d, esperado 2", len(resp.Poltronas))
	}
	if resp.Poltronas[1].Tipo != "PCD" || resp.Poltronas[1].Status != "RESERVADA" {
		t.Errorf("segunda poltrona = %+v, esperado tipo PCD e status RESERVADA", resp.Poltronas[1])
	}
}

func TestMapaSemTokenRecusa401(t *testing.T) {
	api := apiDeTeste(&bloqueioFalso{}, &mapaFalso{})

	w := executar(t, api, http.MethodGet, "/api/v1/sessoes/s1/poltronas", "", "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperado 401", w.Code)
	}
}

func TestDocumentacaoDispensaCredencial(t *testing.T) {
	api := apiDeTeste(&bloqueioFalso{}, &mapaFalso{})

	for _, caminho := range []string{"/docs", "/docs/", "/openapi.yaml"} {
		w := executar(t, api, http.MethodGet, caminho, "", "")
		if w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, esperado 200", caminho, w.Code)
		}
	}
}
