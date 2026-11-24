package obs

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var prefixosSilenciosos = []string{"/health", "/api/v1/health", "/docs", "/openapi.yaml"}

func silenciosa(caminho string) bool {
	for _, p := range prefixosSilenciosos {
		if strings.HasPrefix(caminho, p) {
			return true
		}
	}
	return false
}

type chaveRota struct{}

// MiddlewareHTTP instrumenta o servidor: extrai o traceparent recebido e abre o
// span da requisição (otelhttp), conta e cronometra com o padrão de rota do chi
// e escreve uma linha de log com trace_id.
//
// Deve ser registrado com r.Use(...) DENTRO do roteador chi: o padrão da rota
// só é conhecido por quem roda sob ele, e é ele (não a URL) que vira rótulo.
func MiddlewareHTTP(next http.Handler) http.Handler {
	interno := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		emVoo.Inc()
		defer emVoo.Dec()
		inicio := time.Now()

		next.ServeHTTP(ww, r)

		rota := "desconhecida"
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			rota = rc.RoutePattern()
		}
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		dur := time.Since(inicio)

		requisicoes.WithLabelValues(r.Method, rota, strconv.Itoa(status)).Inc()
		duracaoHTTP.WithLabelValues(r.Method, rota).Observe(dur.Seconds())

		if h, ok := r.Context().Value(chaveRota{}).(*string); ok {
			*h = rota
		}
		trace.SpanFromContext(r.Context()).SetAttributes(
			attribute.String("http.route", rota), attribute.String("request.id", RequestID(r.Context())))

		nivel := slog.LevelInfo
		switch {
		case status >= 500:
			nivel = slog.LevelError
		case silenciosa(r.URL.Path):
			nivel = slog.LevelDebug
		}
		slog.LogAttrs(r.Context(), nivel, "requisição concluída",
			slog.String("method", r.Method), slog.String("route", rota),
			slog.Int("status", status), slog.Float64("duration_ms", float64(dur.Microseconds())/1000),
			slog.Int("bytes", ww.BytesWritten()))
	})

	// O otelhttp dá nome ao span de novo depois de o handler terminar, então o
	// nome com o padrão da rota (e não com a URL, que carrega ids) precisa vir
	// do formatador, lendo o que o handler anotou neste ponteiro.
	traced := otelhttp.NewHandler(interno, "http.server",
		otelhttp.WithFilter(func(r *http.Request) bool { return !silenciosa(r.URL.Path) }),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			if h, ok := r.Context().Value(chaveRota{}).(*string); ok && *h != "" {
				return r.Method + " " + *h
			}
			return r.Method
		}),
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Request id na borda: aceita o do cliente se for seguro, senão gera um;
		// devolve no cabeçalho de resposta e propaga pelo contexto.
		id := RequestIDAceito(r.Header.Get(HeaderRequestID))
		w.Header().Set(HeaderRequestID, id)
		ctx := ComRequestID(context.WithValue(r.Context(), chaveRota{}, new(string)), id)
		traced.ServeHTTP(w, r.WithContext(ctx))
	})
}
