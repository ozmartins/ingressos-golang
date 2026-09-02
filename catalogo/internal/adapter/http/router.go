package http

import (
	"net/http"

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
	{Metodo: "GET", Caminho: "/api/v1/cinemas", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetCinemas })},
	{Metodo: "GET", Caminho: "/api/v1/cinemas/{id}/salas", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetSalasDoCinema })},
	{Metodo: "GET", Caminho: "/api/v1/sessoes", Documentada: true,
		handler: simples(func(d Dependencias) http.HandlerFunc { return d.Handlers.GetSessoes })},
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

func NovoRouter(d Dependencias) http.Handler {
	mux := http.NewServeMux()

	protegida := middleware.Autenticacao(d.Verificador, func(w http.ResponseWriter, r *http.Request, detalhe string) {
		EscreverProblem(w, r, catNaoAutenticado, detalhe)
	})

	for _, rota := range rotas {
		h := rota.handler(d)
		if rota.Protegida {
			h = protegida(h)
		}
		mux.Handle(rota.Metodo+" "+rota.Caminho, h)
	}

	return middleware.Encadear(mux,
		middleware.Telemetria,
		middleware.Recuperacao,
		middleware.Log(d.Metricas),
	)
}
