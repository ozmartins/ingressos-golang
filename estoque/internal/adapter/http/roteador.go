package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var metodosSondados = []string{
	http.MethodDelete, http.MethodGet, http.MethodOptions,
	http.MethodPatch, http.MethodPost, http.MethodPut,
}

func NovoRoteador() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.GetHead)
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		if allow := metodosAceitos(r, req); allow != "" {
			w.Header().Set("Allow", allow)
		}
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})
	return r
}

func metodosAceitos(r *chi.Mux, req *http.Request) string {
	caminho := req.URL.RawPath
	if caminho == "" {
		caminho = req.URL.Path
	}

	var aceitos []string
	for _, m := range metodosSondados {
		if !r.Match(chi.NewRouteContext(), m, caminho) {
			continue
		}
		aceitos = append(aceitos, m)
		if m == http.MethodGet {
			aceitos = append(aceitos, http.MethodHead)
		}
	}
	return strings.Join(aceitos, ", ")
}
