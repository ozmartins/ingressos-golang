package http

import (
	"net/http"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/http/middleware"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"
)

// Dependencias reúne o que o roteador precisa montar.
type Dependencias struct {
	Handlers    Handlers
	Saude       http.HandlerFunc
	Verificador middleware.VerificadorDeCredencial
	Metricas    *observability.Metricas
}

// NovoRouter monta as rotas do contrato.
//
// A autenticação é aplicada só à rota de reserva: o catálogo é público por
// especificação, e envolver tudo no mesmo middleware exigiria uma lista de
// exceções — que é onde erros de autorização costumam nascer.
func NovoRouter(d Dependencias) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/filmes", d.Handlers.GetFilmes)
	mux.HandleFunc("GET /api/v1/cinemas", d.Handlers.GetCinemas)
	mux.HandleFunc("GET /api/v1/cinemas/{id}/salas", d.Handlers.GetSalasDoCinema)
	mux.HandleFunc("GET /api/v1/sessoes", d.Handlers.GetSessoes)
	mux.Handle("GET /health", d.Saude)

	protegida := middleware.Autenticacao(d.Verificador, func(w http.ResponseWriter, r *http.Request, detalhe string) {
		EscreverProblem(w, r, catNaoAutenticado, detalhe)
	})
	mux.Handle("POST /api/v1/sessoes/{id}/reservar",
		protegida(http.HandlerFunc(d.Handlers.PostReservar)))

	// Ordem das camadas, de fora para dentro: telemetria abre o span (para que
	// o log já saia correlacionado), recuperação protege o resto, log mede o
	// que efetivamente aconteceu.
	return middleware.Encadear(mux,
		middleware.Telemetria,
		middleware.Recuperacao,
		middleware.Log(d.Metricas),
	)
}
