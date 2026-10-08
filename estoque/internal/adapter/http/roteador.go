package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// metodosSondados está em ordem alfabética: é a ordem em que o ServeMux da
// biblioteca padrão listava o cabeçalho Allow.
var metodosSondados = []string{
	http.MethodDelete, http.MethodGet, http.MethodOptions,
	http.MethodPatch, http.MethodPost, http.MethodPut,
}

// NovoRoteador devolve um roteador chi que responde a HEAD e a 405 como o
// ServeMux anterior: HEAD atende toda rota GET, e o 405 leva o corpo em texto
// plano e o cabeçalho Allow. O chi não entrega a um handler de 405 os métodos
// aceitos, então Allow é obtido sondando o próprio roteador.
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
