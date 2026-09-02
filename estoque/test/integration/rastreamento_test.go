//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	adaptadoramqp "github.com/oseias/ingressos-golang/estoque/internal/adapter/amqp"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
)

func TestCorrelacaoPontaAPontaAtravessaOBroker(t *testing.T) {
	gravador := tracetest.NewSpanRecorder()
	provedor := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(gravador))
	anterior := otel.GetTracerProvider()
	otel.SetTracerProvider(provedor)
	t.Cleanup(func() { otel.SetTracerProvider(anterior) })

	obs, err := observability.Iniciar(context.Background(), "error", "")
	if err != nil {
		t.Fatalf("observabilidade: %v", err)
	}

	c := montarCenario(t, false)
	ctxRaiz, cancelar := context.WithCancel(context.Background())
	defer cancelar()

	conexao := conectarBroker(t)
	if err := adaptadoramqp.ConsumirPagamentoSucesso(ctxRaiz, conexao, 8, obs, c.Confirmar); err != nil {
		t.Fatalf("consumidor: %v", err)
	}
	publicador := &adaptadoramqp.Publicador{Conexao: conexao, Banco: c.Banco, Obs: obs}

	ctxSolicitacao, span := obs.Tracer.Start(context.Background(), "solicitacao-do-catalogo")
	traceEsperado := span.SpanContext().TraceID().String()

	sessao := c.novaSessao(t, []string{"A"}, 3)
	resultado, err := c.Bloquear.Executar(ctxSolicitacao, sessao, usuario, []string{"A1"})
	if err != nil || !resultado.Concedido {
		t.Fatalf("bloqueio: %v", err)
	}
	span.End()

	if _, err := publicador.Drenar(context.Background(), 200); err != nil {
		t.Fatalf("drenar: %v", err)
	}

	publicarComTrace(t, "pagamento.sucesso", resultado.Reserva.ID, traceEsperado)

	aguardar(t, 20*time.Second, func() bool {
		return c.statusReserva(t, resultado.Reserva.ID) == "CONFIRMADA"
	}, "consumo não confirmou a reserva")

	aguardar(t, 10*time.Second, func() bool {
		for _, s := range gravador.Ended() {
			if s.Name() == "consumir "+adaptadoramqp.FilaPagamentoSucesso &&
				s.SpanContext().TraceID().String() == traceEsperado {
				return true
			}
		}
		return false
	}, "o span de consumo nasceu em outro trace — o rastro se perde no broker")
}

func publicarComTrace(t *testing.T, routingKey, reservaID, traceID string) {
	t.Helper()

	if _, err := trace.TraceIDFromHex(traceID); err != nil {
		t.Fatalf("trace id inválido: %v", err)
	}
	publicarBrutoComHeader(t, routingKey, reservaID,
		"00-"+traceID+"-00f067aa0ba902b7-01")
}
