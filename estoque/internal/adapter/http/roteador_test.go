package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
)

// Teste de caracterização do roteamento (specs/003-roteamento-chi, D8): fixa o
// que um cliente observa em 404, 405, HEAD e subcaminhos de /docs, para que a
// troca do roteador não mude nada sem que um teste avise.

const (
	textoPlano   = "text/plain; charset=utf-8"
	corpo405     = "Method Not Allowed\n"
	corpo404     = "404 page not found\n"
	rotaBloqueio = "/api/v1/sessoes/s1/bloqueios"
	rotaMapa     = "/api/v1/sessoes/s1/poltronas"
)

func TestRoteamentoRespostasDeErroDoRoteador(t *testing.T) {
	casos := []struct {
		nome, metodo, caminho string
		status                int
		allow                 string
	}{
		{"GET em rota só-POST", http.MethodGet, rotaBloqueio, http.StatusMethodNotAllowed, "POST"},
		{"PUT em rota só-POST", http.MethodPut, rotaBloqueio, http.StatusMethodNotAllowed, "POST"},
		{"DELETE em rota GET", http.MethodDelete, rotaMapa, http.StatusMethodNotAllowed, "GET, HEAD"},
		{"OPTIONS em rota GET", http.MethodOptions, rotaMapa, http.StatusMethodNotAllowed, "GET, HEAD"},
		{"POST em /docs", http.MethodPost, "/docs", http.StatusMethodNotAllowed, "GET, HEAD"},
		{"POST em /openapi.yaml", http.MethodPost, "/openapi.yaml", http.StatusMethodNotAllowed, "GET, HEAD"},
		{"caminho inexistente", http.MethodGet, "/nada", http.StatusNotFound, ""},
		{"prefixo de rota sem o restante", http.MethodGet, "/api/v1/sessoes/s1", http.StatusNotFound, ""},
		{"barra final em rota sem barra", http.MethodGet, rotaMapa + "/", http.StatusNotFound, ""},
	}

	api := apiDeTeste(&bloqueioFalso{}, &mapaFalso{})
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			w := executar(t, api, c.metodo, c.caminho, "", "")

			if w.Code != c.status {
				t.Fatalf("status = %d, esperado %d", w.Code, c.status)
			}
			if got := w.Header().Get("Allow"); got != c.allow {
				t.Errorf("Allow = %q, esperado %q", got, c.allow)
			}
			if got := w.Header().Get("Content-Type"); got != textoPlano {
				t.Errorf("Content-Type = %q, esperado %q", got, textoPlano)
			}
			esperado := corpo404
			if c.status == http.StatusMethodNotAllowed {
				esperado = corpo405
			}
			if got := w.Body.String(); got != esperado {
				t.Errorf("corpo = %q, esperado %q", got, esperado)
			}
		})
	}
}

func TestRoteamentoDocumentacaoEmTodosOsCaminhos(t *testing.T) {
	api := apiDeTeste(&bloqueioFalso{}, &mapaFalso{})

	for _, caminho := range []string{"/docs", "/docs/", "/docs/qualquer/coisa", "/openapi.yaml"} {
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			w := executar(t, api, metodo, caminho, "", "")
			if w.Code != http.StatusOK {
				t.Errorf("%s %s = %d, esperado 200", metodo, caminho, w.Code)
			}
		}
	}
}

func TestRoteamentoHeadAtendeRotaGet(t *testing.T) {
	api := apiDeTeste(&bloqueioFalso{}, &mapaFalso{})

	w := executar(t, api, http.MethodHead, rotaMapa, "", tokenValido(t))
	if w.Code != http.StatusOK {
		t.Errorf("HEAD %s = %d, esperado 200", rotaMapa, w.Code)
	}

	w = executar(t, api, http.MethodHead, rotaMapa, "", "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("HEAD %s sem credencial = %d, esperado 401", rotaMapa, w.Code)
	}
}

type mapaQueCapturaSessao struct{ sessao string }

func (m *mapaQueCapturaSessao) Executar(_ context.Context, sessaoID string) ([]poltrona.Poltrona, error) {
	m.sessao = sessaoID
	return nil, nil
}

func TestRoteamentoParametroDeCaminhoChegaDecodificado(t *testing.T) {
	casos := map[string]string{
		"/api/v1/sessoes/abc/poltronas":    "abc",
		"/api/v1/sessoes/x%2Fy/poltronas":  "x/y",
		"/api/v1/sessoes/a%20b/poltronas":  "a b",
		"/api/v1/sessoes/%C3%A7/poltronas": "ç",
	}

	for caminho, esperado := range casos {
		mapa := &mapaQueCapturaSessao{}
		api := apiDeTeste(&bloqueioFalso{}, mapa)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, caminho, nil)
		r.Header.Set("Authorization", "Bearer "+tokenValido(t))
		api.Rotas().ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, esperado 200. corpo: %s", caminho, w.Code, w.Body.String())
		}
		if mapa.sessao != esperado {
			t.Errorf("GET %s: sessao_id = %q, esperado %q", caminho, mapa.sessao, esperado)
		}
	}
}

// Diferença aceita pelo mantenedor (specs/003-roteamento-chi, research D7): o
// ServeMux redirecionava (307) caminhos não canônicos ao caminho limpo; o chi
// responde 404. Nenhum cliente do repositório gera esses caminhos.
func TestRoteamentoCaminhoNaoCanonicoNaoRedireciona(t *testing.T) {
	api := apiDeTeste(&bloqueioFalso{}, &mapaFalso{})

	for _, caminho := range []string{"//docs", "/api/v1/../docs", "/docs/../openapi.yaml"} {
		w := executar(t, api, http.MethodGet, caminho, "", "")
		if w.Code == http.StatusTemporaryRedirect {
			t.Errorf("GET %s redirecionou; a decisão D7 é não redirecionar", caminho)
		}
		if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, esperado 404 (ou 200 se o chi casar o caminho)", caminho, w.Code)
		}
	}
}
