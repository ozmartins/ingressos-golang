package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Valores medidos no http.ServeMux antes da troca pelo chi; a troca não pode
// mudá-los (FR-005, FR-006).
func TestRoteamentoPreservaRespostasDoServeMux(t *testing.T) {
	api := apiCom(repoStub{})
	const (
		texto404 = "404 page not found\n"
		texto405 = "Method Not Allowed\n"
		textoCT  = "text/plain; charset=utf-8"
		htmlCT   = "text/html; charset=utf-8"
	)
	casos := []struct {
		metodo, caminho string
		status          int
		contentType     string
		allow           string
		corpo           string // vazio = não confere
	}{
		{"GET", "/nao-existe", 404, textoCT, "", texto404},
		{"GET", "/api/v1/health/live/", 404, textoCT, "", texto404},
		{"GET", "/api/v1/pagamentos/reserva/x/", 404, textoCT, "", texto404},
		{"DELETE", "/api/v1/pagamentos/reserva/x", 405, textoCT, "GET, HEAD, POST", texto405},
		{"PUT", "/docs", 405, textoCT, "GET, HEAD", texto405},
		{"OPTIONS", "/docs", 405, textoCT, "GET, HEAD", texto405},
		{"POST", "/api/v1/health/live", 405, textoCT, "GET, HEAD", texto405},
		{"HEAD", "/api/v1/health/live", 200, "application/json", "", ""},
		{"HEAD", "/docs", 200, htmlCT, "", ""},
		{"GET", "/docs", 200, htmlCT, "", ""},
		{"GET", "/docs/", 200, htmlCT, "", ""},
		{"GET", "/docs/qualquer", 200, htmlCT, "", ""},
	}
	for _, c := range casos {
		t.Run(c.metodo+" "+c.caminho, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.Rotas().ServeHTTP(w, httptest.NewRequest(c.metodo, c.caminho, nil))
			if w.Code != c.status {
				t.Errorf("status = %d, esperado %d", w.Code, c.status)
			}
			if got := w.Header().Get("Content-Type"); got != c.contentType {
				t.Errorf("Content-Type = %q, esperado %q", got, c.contentType)
			}
			if got := w.Header().Get("Allow"); got != c.allow {
				t.Errorf("Allow = %q, esperado %q", got, c.allow)
			}
			if c.corpo != "" && w.Body.String() != c.corpo {
				t.Errorf("corpo = %q, esperado %q", w.Body.String(), c.corpo)
			}
		})
	}
}

func TestReservaIDChegaAoTratamentoSemAlteracao(t *testing.T) {
	api := apiCom(repoStub{})
	w := chamar(t, api, "nao-e-uuid", token(t, dona, nil))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), CodReservaIDInvalido) {
		t.Fatalf("esperado 400 %s, veio %d %s", CodReservaIDInvalido, w.Code, w.Body.String())
	}
}
