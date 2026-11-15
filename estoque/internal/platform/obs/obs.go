// Package obs reúne a observabilidade do serviço: log estruturado (log/slog em
// JSON), métricas Prometheus e tracing distribuído OpenTelemetry (W3C Trace
// Context). Fica em platform porque é infraestrutura: o núcleo (domain e
// usecase) nunca a importa.
package obs

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Observabilidade é a alça devolvida por Iniciar; Desligar descarrega os spans.
type Observabilidade struct {
	servico string
	tp      *sdktrace.TracerProvider
}

// Iniciar configura, para o processo inteiro: o logger padrão (JSON, com
// trace_id/span_id quando há span no contexto), o propagador W3C e o
// TracerProvider.
//
// Os spans só saem do processo se OTEL_EXPORTER_OTLP_ENDPOINT (ou a variante
// _TRACES_) estiver definida. Sem ela o provedor continua gerando ids, para que
// os logs sempre carreguem trace_id, mas nada é exportado.
func Iniciar(ctx context.Context, servico string) (*Observabilidade, error) {
	configurarLog(servico)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		semconv.ServiceName(servico),
	))
	if err != nil {
		return nil, err
	}

	opcoes := []sdktrace.TracerProviderOption{sdktrace.WithResource(res), sdktrace.WithSampler(amostrador())}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != "" {
		exp, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, err
		}
		opcoes = append(opcoes, sdktrace.WithBatcher(exp))
	}

	tp := sdktrace.NewTracerProvider(opcoes...)
	otel.SetTracerProvider(tp)
	return &Observabilidade{servico: servico, tp: tp}, nil
}

// Desligar descarrega os spans pendentes.
func (o *Observabilidade) Desligar(ctx context.Context) error {
	if o == nil || o.tp == nil {
		return nil
	}
	return o.tp.Shutdown(ctx)
}

// ServirMetricas expõe GET /metrics (formato Prometheus) em addr, numa porta
// própria: assim a rota não passa pela autenticação nem aparece no OpenAPI.
// Devolve uma função que encerra o servidor.
func ServirMetricas(addr string) (parar func(context.Context) error) {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(Registro, promhttp.HandlerOpts{}))
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erroDeLog("servidor de métricas encerrou", err)
		}
	}()
	return srv.Shutdown
}

// EnderecoMetricas lê OBS_ADDR; o padrão é :9090.
func EnderecoMetricas() string {
	if v := os.Getenv("OBS_ADDR"); v != "" {
		return v
	}
	return ":9090"
}

// amostrador decide quais traces são gravados. Segue a decisão do pai; para
// spans sem pai, descarta os de acesso a dados (banco, Redis): o publicador da
// caixa de saída e as varreduras consultam o banco a cada segundo e, sem isto,
// cada consulta viraria um trace de um span só, afogando os traces reais. Uma
// consulta feita dentro de uma requisição ou de uma mensagem tem pai e é gravada.
func amostrador() sdktrace.Sampler { return sdktrace.ParentBased(raizSemDados{}) }

type raizSemDados struct{}

func (raizSemDados) Description() string { return "raiz sem acesso a dados" }

func (raizSemDados) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	estado := trace.SpanContextFromContext(p.ParentContext).TraceState()
	if acessoADados(p) {
		return sdktrace.SamplingResult{Decision: sdktrace.Drop, Tracestate: estado}
	}
	return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample, Tracestate: estado}
}

func acessoADados(p sdktrace.SamplingParameters) bool {
	if strings.HasPrefix(p.Name, "gorm.") {
		return true
	}
	for _, a := range p.Attributes {
		if k := string(a.Key); k == "db.system" || k == "db.system.name" {
			return true
		}
	}
	return false
}
