package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"
)

func Telemetria(prox http.Handler) http.Handler {
	return otelhttp.NewHandler(prox, "http.server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			if p := r.Pattern; p != "" {
				return p
			}
			return r.Method + " " + r.URL.Path
		}),
	)
}

type gravadorComStatus struct {
	http.ResponseWriter
	status int
}

func (g *gravadorComStatus) WriteHeader(codigo int) {
	g.status = codigo
	g.ResponseWriter.WriteHeader(codigo)
}

func (g *gravadorComStatus) Write(b []byte) (int, error) {
	if g.status == 0 {
		g.status = http.StatusOK
	}
	return g.ResponseWriter.Write(b)
}

func Log(metricas *observability.Metricas) func(http.Handler) http.Handler {
	return func(prox http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inicio := time.Now()
			g := &gravadorComStatus{ResponseWriter: w}

			prox.ServeHTTP(g, r)

			duracao := time.Since(inicio)
			rota := r.Pattern
			if rota == "" {
				rota = r.URL.Path
			}

			slog.InfoContext(r.Context(), "requisição atendida",
				slog.String("metodo", r.Method),
				slog.String("rota", rota),
				slog.Int("status", g.status),
				slog.Duration("duracao", duracao),
			)

			if metricas != nil {
				metricas.HTTPDuracao.Record(r.Context(), duracao.Seconds(), metric.WithAttributes(
					attribute.String("rota", rota),
					attribute.Int("status", g.status),
				))
			}
		})
	}
}

func Recuperacao(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.ErrorContext(r.Context(), "pânico atendendo requisição", slog.Any("panico", p))
				w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"type":"https://cinema.example/errors/erro-interno","title":"Erro interno","status":500,"detail":"Erro interno. Consulte o suporte informando o identificador desta requisição."}`))
			}
		}()
		prox.ServeHTTP(w, r)
	})
}

func Encadear(h http.Handler, camadas ...func(http.Handler) http.Handler) http.Handler {
	for i := len(camadas) - 1; i >= 0; i-- {
		h = camadas[i](h)
	}
	return h
}
