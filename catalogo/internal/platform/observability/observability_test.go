package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
)

func TestIniciarSemColetorConfigurado(t *testing.T) {
	metricas, encerrar, err := Iniciar(context.Background(), "")
	if err != nil {
		t.Fatalf("Iniciar falhou sem coletor configurado: %v", err)
	}
	defer func() {
		if err := encerrar(context.Background()); err != nil {
			t.Errorf("encerrar: %v", err)
		}
	}()

	if metricas == nil || metricas.HTTPDuracao == nil {
		t.Fatal("Iniciar devolveu instrumentos incompletos")
	}
}

func TestPropagacaoW3CEstaInstaladaSempre(t *testing.T) {
	p := otel.GetTextMapPropagator()
	if p == nil {
		t.Fatal("nenhum propagador instalado")
	}
	campos := p.Fields()
	var temTraceparent bool
	for _, c := range campos {
		if c == "traceparent" {
			temTraceparent = true
		}
	}
	if !temTraceparent {
		t.Fatalf("o propagador não trata traceparent (campos: %v)", campos)
	}
}
