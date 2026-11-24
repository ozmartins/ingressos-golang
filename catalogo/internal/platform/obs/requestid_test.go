package obs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDAceito(t *testing.T) {
	if got := RequestIDAceito("abc-123_X.y"); got != "abc-123_X.y" {
		t.Errorf("id válido deveria ser mantido, veio %q", got)
	}
	for _, ruim := range []string{"", "com espaço", "linha\nnova", strings.Repeat("a", 65), "açaí"} {
		got := RequestIDAceito(ruim)
		if got == ruim || len(got) != 32 {
			t.Errorf("id %q deveria ser substituído por um novo, veio %q", ruim, got)
		}
	}
}

func TestRequestIDNoContexto(t *testing.T) {
	if RequestID(context.Background()) != "" {
		t.Fatal("contexto vazio não deveria ter request id")
	}
	if got := RequestID(ComRequestID(context.Background(), "r1")); got != "r1" {
		t.Errorf("veio %q", got)
	}
}

func TestMiddlewareHTTPRequestID(t *testing.T) {
	var visto string
	h := MiddlewareHTTP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { visto = RequestID(r.Context()) }))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, "cliente-123")
	h.ServeHTTP(rec, req)
	if visto != "cliente-123" || rec.Header().Get(HeaderRequestID) != "cliente-123" {
		t.Errorf("deveria aceitar o id do cliente: contexto=%q resposta=%q", visto, rec.Header().Get(HeaderRequestID))
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if len(visto) != 32 || rec.Header().Get(HeaderRequestID) != visto {
		t.Errorf("deveria gerar um id na borda: contexto=%q resposta=%q", visto, rec.Header().Get(HeaderRequestID))
	}
}
