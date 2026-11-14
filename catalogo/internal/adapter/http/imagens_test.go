package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestImagensServeArquivoEHideListagem(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "filmes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "filmes", "a.jpg"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	router := NovoRouter(Dependencias{Imagens: HandlerImagens(dir)})

	casos := map[string]int{
		"/imagens/filmes/a.jpg":   http.StatusOK,
		"/imagens/filmes/nao.jpg": http.StatusNotFound,
		"/imagens/filmes/":        http.StatusNotFound,
		"/imagens/filmes":         http.StatusNotFound,
		"/imagens/":               http.StatusNotFound,
		"/imagens/../etc/passwd":  http.StatusNotFound,
	}
	for caminho, esperado := range casos {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caminho, nil))
		if w.Code != esperado {
			t.Errorf("%s: status %d, esperado %d", caminho, w.Code, esperado)
		}
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/imagens/filmes/a.jpg", nil))
	if w.Body.String() != "img" {
		t.Errorf("corpo = %q", w.Body.String())
	}
}
