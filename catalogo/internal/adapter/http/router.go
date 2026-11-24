package http

import (
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/http/middleware"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/http/openapi"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/obs"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/ratelimit"
)

type Dependencias struct {
	Handlers    Handlers
	Saude       http.HandlerFunc
	Verificador middleware.VerificadorDeCredencial
	// Taxa limita as rotas /api/ por IP de origem (nil = sem limite).
	Taxa *ratelimit.Limitador
}

type Rota struct {
	Metodo      string
	Caminho     string
	Documentada bool
	Protegida   bool
	handler     func(Dependencias) http.Handler
}

func simples(escolher func(Dependencias) http.HandlerFunc) func(Dependencias) http.Handler {
	return func(d Dependencias) http.Handler { return escolher(d) }
}

var rotas = []Rota{
	{Metodo: "GET", Caminho: "/api/v1/filmes", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetFilmes })},
	{Metodo: "GET", Caminho: "/api/v1/filmes/{id}", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetFilme })},
	{Metodo: "POST", Caminho: "/api/v1/filmes", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PostFilme })},
	{Metodo: "PUT", Caminho: "/api/v1/filmes/{id}", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PutFilme })},
	{Metodo: "DELETE", Caminho: "/api/v1/filmes/{id}", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.DeleteFilme })},
	{Metodo: "GET", Caminho: "/api/v1/cinemas", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetCinemas })},
	{Metodo: "GET", Caminho: "/api/v1/cinemas/{id}", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetCinema })},
	{Metodo: "POST", Caminho: "/api/v1/cinemas", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PostCinema })},
	{Metodo: "PUT", Caminho: "/api/v1/cinemas/{id}", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PutCinema })},
	{Metodo: "DELETE", Caminho: "/api/v1/cinemas/{id}", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.DeleteCinema })},
	{Metodo: "GET", Caminho: "/api/v1/salas", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetSalas })},
	{Metodo: "POST", Caminho: "/api/v1/salas", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PostSala })},
	{Metodo: "GET", Caminho: "/api/v1/salas/{id}", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetSala })},
	{Metodo: "PUT", Caminho: "/api/v1/salas/{id}", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PutSala })},
	{Metodo: "DELETE", Caminho: "/api/v1/salas/{id}", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.DeleteSala })},
	{Metodo: "GET", Caminho: "/api/v1/sessoes", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetSessoes })},
	{Metodo: "POST", Caminho: "/api/v1/sessoes", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PostSessao })},
	{Metodo: "GET", Caminho: "/api/v1/sessoes/{id}", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetSessao })},
	{Metodo: "PUT", Caminho: "/api/v1/sessoes/{id}", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PutSessao })},
	{Metodo: "DELETE", Caminho: "/api/v1/sessoes/{id}", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.DeleteSessao })},
	{Metodo: "GET", Caminho: "/health", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Saude })},

	{Metodo: "GET", Caminho: "/openapi.yaml",
		handler: simples(func(Dependencias) http.HandlerFunc { return openapi.HandlerEspecificacao() })},
	{Metodo: "GET", Caminho: "/docs",
		handler: simples(func(Dependencias) http.HandlerFunc { return openapi.HandlerUI("/openapi.yaml") })},
	{Metodo: "GET", Caminho: "/docs/",
		handler: simples(func(Dependencias) http.HandlerFunc { return openapi.HandlerUI("/openapi.yaml") })},
}

func Rotas() []Rota { return append([]Rota(nil), rotas...) }

func NovoRouter(d Dependencias) http.Handler { return novoRouter(rotas, d) }

func novoRouter(tabela []Rota, d Dependencias) http.Handler {
	r := chi.NewRouter()
	r.Use(
		obs.MiddlewareHTTP,
		middleware.Recuperacao,
		chimw.GetHead,
	)
	r.MethodNotAllowed(metodoNaoPermitido)

	protegida := middleware.Autenticacao(d.Verificador, func(w http.ResponseWriter, r *http.Request, detalhe string) {
		EscreverProblem(w, r, catNaoAutenticado, detalhe)
	})

	// Rota protegida = autenticada (401) e com permissão de escrita (403).
	exigirEscrita := middleware.Autorizacao(identidade.PermissaoEscrita, func(w http.ResponseWriter, r *http.Request, detalhe string) {
		EscreverProblem(w, r, catAcessoNegado, detalhe)
	})

	for _, rota := range tabela {
		h := rota.handler(d)
		if rota.Protegida {
			h = protegida(exigirEscrita(h))
		}
		if strings.HasPrefix(rota.Caminho, "/api/") {
			h = limitarPorIP(d.Taxa, h)
		}
		r.Method(rota.Metodo, rota.Caminho, comPadrao(rota, h))
	}

	return r
}

// limitarPorIP devolve 429 com Retry-After quando o IP de origem excede a taxa.
// Não confia em X-Forwarded-For (forjável); atrás de um proxy, todos os
// clientes compartilham o IP do proxy, por isso o padrão é generoso.
func limitarPorIP(taxa *ratelimit.Limitador, h http.Handler) http.Handler {
	if taxa == nil {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !taxa.Permitir(ratelimit.IPDe(r)) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func comPadrao(rota Rota, h http.Handler) http.Handler {
	padrao := rota.Metodo + " " + rota.Caminho
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Pattern = padrao
		h.ServeHTTP(w, r)
	})
}

func metodoNaoPermitido(w http.ResponseWriter, r *http.Request) {
	var permitidos []string
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		if chi.RouteContext(r.Context()).Routes.Match(chi.NewRouteContext(), m, r.URL.Path) {
			permitidos = append(permitidos, m)
		}
	}
	if slices.Contains(permitidos, http.MethodGet) {
		permitidos = append(permitidos, http.MethodHead)
	}
	sort.Strings(permitidos)
	w.Header().Set("Allow", strings.Join(permitidos, ", "))
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}
