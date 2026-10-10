package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func executar(t *testing.T, s *Servico, metodo, caminho string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(metodo, caminho, nil))
	return w
}

func ok(context.Context) error    { return nil }
func falha(context.Context) error { return errors.New("fora do ar") }

func TestVivoResponde200(t *testing.T) {
	w := executar(t, Novo(), http.MethodGet, "/health/live")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200", w.Code)
	}
	if got := w.Body.String(); got != `{"status":"vivo"}` {
		t.Errorf("corpo = %q", got)
	}
}

func TestProntidaoPorEstadoDasDependencias(t *testing.T) {
	casos := []struct {
		nome         string
		verificacoes []Verificacao
		status       int
		contem       []string
	}{
		{"sem dependências", nil, http.StatusOK, []string{`"status":"pronto"`}},
		{"todas ok", []Verificacao{{Nome: "postgres", Essencial: true, Checar: ok}}, http.StatusOK,
			[]string{`"status":"pronto"`, `"postgres":"ok"`}},
		{"essencial falhando", []Verificacao{{Nome: "postgres", Essencial: true, Checar: falha}},
			http.StatusServiceUnavailable, []string{`"status":"indisponivel"`, `"postgres":"indisponivel"`}},
		{"não essencial falhando", []Verificacao{{Nome: "redis", Essencial: false, Checar: falha}},
			http.StatusOK, []string{`"status":"degradado"`, `"redis":"degradado"`}},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			w := executar(t, Novo(c.verificacoes...), http.MethodGet, "/health/ready")

			if w.Code != c.status {
				t.Fatalf("status = %d, esperado %d", w.Code, c.status)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, esperado application/json", got)
			}
			for _, trecho := range c.contem {
				if !strings.Contains(w.Body.String(), trecho) {
					t.Errorf("corpo %q não contém %q", w.Body.String(), trecho)
				}
			}
		})
	}
}

func TestRoteamentoDaSaude(t *testing.T) {
	casos := []struct {
		nome, metodo, caminho string
		status                int
		allow, corpo          string
	}{
		{"HEAD em live", http.MethodHead, "/health/live", http.StatusOK, "", ""},
		{"HEAD em ready", http.MethodHead, "/health/ready", http.StatusOK, "", ""},
		{"POST em live", http.MethodPost, "/health/live", http.StatusMethodNotAllowed, "GET, HEAD", "Method Not Allowed\n"},
		{"DELETE em ready", http.MethodDelete, "/health/ready", http.StatusMethodNotAllowed, "GET, HEAD", "Method Not Allowed\n"},
		{"OPTIONS em live", http.MethodOptions, "/health/live", http.StatusMethodNotAllowed, "GET, HEAD", "Method Not Allowed\n"},
		{"raiz", http.MethodGet, "/", http.StatusNotFound, "", "404 page not found\n"},
		{"/health sem sufixo", http.MethodGet, "/health", http.StatusNotFound, "", "404 page not found\n"},
		{"barra final", http.MethodGet, "/health/live/", http.StatusNotFound, "", "404 page not found\n"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			w := executar(t, Novo(), c.metodo, c.caminho)

			if w.Code != c.status {
				t.Fatalf("status = %d, esperado %d", w.Code, c.status)
			}
			if got := w.Header().Get("Allow"); got != c.allow {
				t.Errorf("Allow = %q, esperado %q", got, c.allow)
			}
			if c.metodo != http.MethodHead && w.Body.String() != c.corpo && c.status != http.StatusOK {
				t.Errorf("corpo = %q, esperado %q", w.Body.String(), c.corpo)
			}
		})
	}
}
