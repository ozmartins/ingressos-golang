package http

import (
	"net/http"
	"strings"
	"testing"
)

func (a *ambiente) metodo(t *testing.T, metodo, caminho string) (resposta, string) {
	t.Helper()
	req, err := http.NewRequest(metodo, a.srv.URL+caminho, nil)
	if err != nil {
		t.Fatalf("montar requisição: %v", err)
	}
	res, corpo := a.enviar(t, req, nil)
	return res, string(corpo)
}

func TestCaminhoInexistenteResponde404(t *testing.T) {
	a := montarAmbiente(t)
	res, corpo := a.metodo(t, http.MethodGet, "/inexistente")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, queria 404", res.StatusCode)
	}
	if strings.TrimSpace(corpo) != "404 page not found" {
		t.Errorf("corpo = %q", corpo)
	}
}

func TestMetodoNaoPermitidoResponde405ComAllow(t *testing.T) {
	a := montarAmbiente(t)
	casos := []struct{ metodo, caminho, allow string }{
		{http.MethodGet, "/api/v1/ingressos/validar", "POST"},
		{http.MethodPost, "/health/live", "GET, HEAD"},
		{http.MethodPost, "/api/v1/ingressos/meus-ingressos", "GET, HEAD"},
		{http.MethodDelete, "/docs", "GET, HEAD"},
	}
	for _, c := range casos {
		res, corpo := a.metodo(t, c.metodo, c.caminho)
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status = %d, queria 405", c.metodo, c.caminho, res.StatusCode)
			continue
		}
		if got := res.Header.Get("Allow"); got != c.allow {
			t.Errorf("%s %s: Allow = %q, queria %q", c.metodo, c.caminho, got, c.allow)
		}
		if strings.TrimSpace(corpo) != "Method Not Allowed" {
			t.Errorf("%s %s: corpo = %q", c.metodo, c.caminho, corpo)
		}
	}
}

func TestHeadEmRotaGetResponde200SemCorpo(t *testing.T) {
	a := montarAmbiente(t)
	res, corpo := a.metodo(t, http.MethodHead, "/health/live")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, queria 200", res.StatusCode)
	}
	if corpo != "" {
		t.Errorf("HEAD devolveu corpo: %q", corpo)
	}
}

func TestRotasPublicasNaoExigemCredencial(t *testing.T) {
	a := montarAmbiente(t)
	for _, caminho := range []string{"/health/live", "/health/ready", "/openapi.yaml", "/docs", "/docs/", "/docs/qualquer"} {
		res, _ := a.metodo(t, http.MethodGet, caminho)
		if res.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status = %d, queria 200", caminho, res.StatusCode)
		}
	}
}

func TestRotasProtegidasContinuamExigindoCredencial(t *testing.T) {
	a := montarAmbiente(t)
	res, _ := a.metodo(t, http.MethodGet, "/api/v1/ingressos/meus-ingressos")
	if res.StatusCode != http.StatusUnauthorized || res.Header.Get("Content-Type") != "application/problem+json" {
		t.Errorf("meus-ingressos: %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	semChave, _ := a.postValidar(t, `{"codigo_qr":"x"}`, nil)
	if semChave.StatusCode != http.StatusUnauthorized {
		t.Errorf("validar sem chave: %d", semChave.StatusCode)
	}
}
