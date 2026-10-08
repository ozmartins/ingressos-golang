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
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"
)

type Dependencias struct {
	Handlers    Handlers
	Saude       http.HandlerFunc
	Verificador middleware.VerificadorDeCredencial
	Metricas    *observability.Metricas
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
	{Metodo: "POST", Caminho: "/api/v1/sessoes/{id}/reservar", Documentada: true, Protegida: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.PostReservar })},
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

// novoRouter recebe a tabela para que os testes registrem handlers falsos sob
// os mesmos métodos e caminhos de produção.
//
// Os middlewares globais entram por r.Use, e não por fora do roteador: o chi
// grava r.Pattern numa cópia da requisição, e um middleware externo leria o
// padrão vazio — o rótulo "rota" de log e métrica viraria o caminho com o
// identificador.
func novoRouter(tabela []Rota, d Dependencias) http.Handler {
	r := chi.NewRouter()
	r.Use(
		middleware.Telemetria,
		middleware.Recuperacao,
		middleware.Log(d.Metricas),
		chimw.GetHead, // o ServeMux atende HEAD pela rota GET; o chi responderia 405
	)
	r.MethodNotAllowed(metodoNaoPermitido)

	protegida := middleware.Autenticacao(d.Verificador, func(w http.ResponseWriter, r *http.Request, detalhe string) {
		EscreverProblem(w, r, catNaoAutenticado, detalhe)
	})

	for _, rota := range tabela {
		h := rota.handler(d)
		if rota.Protegida {
			h = protegida(h)
		}
		r.Method(rota.Metodo, rota.Caminho, comPadrao(rota, h))
	}

	return r
}

// comPadrao devolve ao r.Pattern o formato "MÉTODO /caminho" do ServeMux; o
// chi o preenche só com o caminho.
func comPadrao(rota Rota, h http.Handler) http.Handler {
	padrao := rota.Metodo + " " + rota.Caminho
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Pattern = padrao
		h.ServeHTTP(w, r)
	})
}

// metodoNaoPermitido reproduz o 405 do ServeMux: Allow completo (o chi o
// preenche de forma incompleta) e corpo em texto simples. Os métodos são
// sondados no próprio roteador para não haver uma segunda fonte de verdade.
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
