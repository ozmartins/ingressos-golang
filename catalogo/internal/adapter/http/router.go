package http

import (
	"net/http"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/http/middleware"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/http/openapi"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"
)

// Dependencias reúne o que o roteador precisa montar.
type Dependencias struct {
	Handlers    Handlers
	Saude       http.HandlerFunc
	Verificador middleware.VerificadorDeCredencial
	Metricas    *observability.Metricas
}

// Rota descreve uma entrada do roteador.
//
// A tabela existe para ser lida por dois consumidores: NovoRouter, que a
// registra, e o teste de paridade, que confronta o que está registrado com o
// que o contrato publica. Sem essa forma declarativa, a paridade seria uma
// promessa de revisão humana em vez de uma verificação de build.
type Rota struct {
	Metodo string
	// Caminho é o padrão do ServeMux, com os placeholders na sintaxe do Go
	// ({id}) — que por acaso coincide com a do OpenAPI.
	Caminho string
	// Documentada distingue a superfície da API da superfície de documentação:
	// /docs e /openapi.yaml servem o contrato, não fazem parte dele.
	Documentada bool
	// Protegida marca as rotas que passam pelo middleware de autenticação.
	Protegida bool
	handler   func(Dependencias) http.Handler
}

func simples(escolher func(Dependencias) http.HandlerFunc) func(Dependencias) http.Handler {
	return func(d Dependencias) http.Handler { return escolher(d) }
}

// rotas é o contrato de roteamento do serviço, em um lugar só.
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

	// Documentação: serve o contrato, não pertence a ele.
	{Metodo: "GET", Caminho: "/openapi.yaml",
		handler: simples(func(Dependencias) http.HandlerFunc { return openapi.HandlerEspecificacao() })},
	{Metodo: "GET", Caminho: "/docs",
		handler: simples(func(Dependencias) http.HandlerFunc { return openapi.HandlerUI("/openapi.yaml") })},
	// A variante com barra evita um 404 confuso para quem digita a URL à mão.
	{Metodo: "GET", Caminho: "/docs/",
		handler: simples(func(Dependencias) http.HandlerFunc { return openapi.HandlerUI("/openapi.yaml") })},
}

// Rotas devolve a tabela de roteamento para inspeção.
func Rotas() []Rota { return append([]Rota(nil), rotas...) }

// NovoRouter monta as rotas do contrato.
//
// A autenticação é aplicada só à rota de reserva: o catálogo é público por
// especificação, e envolver tudo no mesmo middleware exigiria uma lista de
// exceções — que é onde erros de autorização costumam nascer.
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

	// Ordem das camadas, de fora para dentro: telemetria abre o span (para que
	// o log já saia correlacionado), recuperação protege o resto, log mede o
	// que efetivamente aconteceu.
	return middleware.Encadear(mux,
		middleware.Telemetria,
		middleware.Recuperacao,
		middleware.Log(d.Metricas),
	)
}
