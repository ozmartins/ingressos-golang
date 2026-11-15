package obs

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func iniciarParaTeste(t *testing.T) {
	t.Helper()
	o, err := Iniciar(context.Background(), "teste")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = o.Desligar(context.Background()) })
}

func TestTraceAtravessaOBrokerPelosHeaders(t *testing.T) {
	iniciarParaTeste(t)

	ctxPub, spanPub, headers := IniciarPublicacao(context.Background(), "cinema", "reserva.criada")
	spanPub.End()

	if headers["traceparent"] == "" {
		t.Fatal("o publicador deveria gravar o traceparent nos headers")
	}

	_, spanCons := IniciarConsumo(context.Background(), "fila", amqp.Delivery{Headers: headers})
	defer spanCons.End()

	if got, want := spanCons.SpanContext().TraceID(), spanPub.SpanContext().TraceID(); got != want {
		t.Fatalf("o consumidor deveria continuar o trace do publicador: %s != %s", got, want)
	}
	_ = ctxPub
}

func TestTraceparentSobreviveAoRoundTripDaCaixaDeSaida(t *testing.T) {
	iniciarParaTeste(t)

	ctx, span := tracer().Start(context.Background(), "requisicao")
	defer span.End()

	tp := Traceparent(ctx)
	if !strings.HasPrefix(tp, "00-"+span.SpanContext().TraceID().String()) {
		t.Fatalf("traceparent inesperado: %q", tp)
	}

	_, filho := tracer().Start(ComTraceparent(context.Background(), tp), "publicador")
	defer filho.End()
	if filho.SpanContext().TraceID() != span.SpanContext().TraceID() {
		t.Fatal("o span do publicador deveria herdar o trace gravado na caixa")
	}
}

func TestMensagemSemHeaderComecaTraceNovo(t *testing.T) {
	iniciarParaTeste(t)
	_, span := IniciarConsumo(context.Background(), "fila", amqp.Delivery{})
	defer span.End()
	if !span.SpanContext().IsValid() {
		t.Fatal("sem traceparent deve começar um trace novo, não falhar")
	}
}

func TestLogCarregaTraceIDQuandoHaSpan(t *testing.T) {
	iniciarParaTeste(t)
	var buf bytes.Buffer
	l := slog.New(manipuladorDeTrace{slog.NewJSONHandler(&buf, nil)})

	ctx, span := tracer().Start(context.Background(), "x")
	defer span.End()
	l.InfoContext(ctx, "com span")
	l.Info("sem span")

	linhas := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if !strings.Contains(linhas[0], `"trace_id":"`+span.SpanContext().TraceID().String()+`"`) {
		t.Errorf("log com span deveria ter trace_id: %s", linhas[0])
	}
	if strings.Contains(linhas[1], "trace_id") {
		t.Errorf("log sem span não deveria ter trace_id: %s", linhas[1])
	}
}

func TestMiddlewareRotulaPeloPadraoDaRotaENuncaPelaURL(t *testing.T) {
	iniciarParaTeste(t)
	r := chi.NewRouter()
	r.Use(MiddlewareHTTP)
	r.Get("/sessoes/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })

	antes := testutil.ToFloat64(requisicoes.WithLabelValues("GET", "/sessoes/{id}", "418"))
	for _, id := range []string{"a", "b", "c"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/sessoes/"+id, nil))
	}
	if got := testutil.ToFloat64(requisicoes.WithLabelValues("GET", "/sessoes/{id}", "418")) - antes; got != 3 {
		t.Fatalf("três ids diferentes deveriam cair na mesma série: contou %v", got)
	}

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nao-existe", nil))
	if testutil.ToFloat64(requisicoes.WithLabelValues("GET", "desconhecida", "404")) < 1 {
		t.Fatal("rota inexistente deveria usar o rótulo fixo 'desconhecida'")
	}
}

func TestSpanDoServidorTemONomeDoPadraoDaRota(t *testing.T) {
	iniciarParaTeste(t)
	gravador := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(gravador)))

	r := chi.NewRouter()
	r.Use(MiddlewareHTTP)
	r.Get("/sessoes/{id}", func(http.ResponseWriter, *http.Request) {})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/sessoes/uuid-qualquer", nil))

	spans := gravador.Ended()
	if len(spans) != 1 || spans[0].Name() != "GET /sessoes/{id}" {
		t.Fatalf("o nome do span deve usar o padrão da rota, não a URL com o id: %+v", spans)
	}
}

func TestAmostradorDescartaConsultaSemPaiMasGravaComPai(t *testing.T) {
	gravador := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(amostrador()), sdktrace.WithSpanProcessor(gravador))
	tr := tp.Tracer("t")

	// Polling da caixa de saída: sem pai, não vira trace.
	_, solta := tr.Start(context.Background(), "gorm.Query")
	solta.End()
	_, redisSolto := tr.Start(context.Background(), "get", trace.WithAttributes(attribute.String("db.system", "redis")))
	redisSolto.End()
	if n := len(gravador.Ended()); n != 0 {
		t.Fatalf("consultas sem pai deveriam ser descartadas, gravou %d", n)
	}

	// Dentro de uma requisição, a mesma consulta é gravada.
	ctx, req := tr.Start(context.Background(), "POST /x")
	_, consulta := tr.Start(ctx, "gorm.Query")
	consulta.End()
	req.End()
	if n := len(gravador.Ended()); n != 2 {
		t.Fatalf("a consulta com pai deveria ser gravada junto da requisição, gravou %d spans", n)
	}
}

func TestBreakerExpoeUmEstadoPorVez(t *testing.T) {
	EstadoDoBreaker("catalogo.sessao", "open")
	if testutil.ToFloat64(estadoBreaker.WithLabelValues("catalogo.sessao", "open")) != 1 ||
		testutil.ToFloat64(estadoBreaker.WithLabelValues("catalogo.sessao", "closed")) != 0 {
		t.Fatal("só o estado atual deve valer 1")
	}
}
