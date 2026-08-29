package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
)

// Este teste existe por causa de um defeito real: a versão do semconv divergia
// da do SDK, resource.Merge recusava juntar os esquemas, e o processo morria na
// inicialização. Nada acusava, porque nenhum teste chamava Iniciar — só a
// execução do binário revelava. Agora acusa.
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

	if metricas == nil || metricas.EstoqueTotal == nil || metricas.EstoqueDuracao == nil ||
		metricas.BreakerEstado == nil || metricas.HTTPDuracao == nil {
		t.Fatal("Iniciar devolveu instrumentos incompletos")
	}
}

// A ausência de coletor não pode desligar a propagação: repassar o contexto
// recebido é obrigação do serviço, não função do observador.
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

// Os quatro desfechos que FR-035 exige, mais o de negócio, precisam existir como
// rótulos distintos — T048 depende deles para provar que uma recusa local não
// contata o estoque.
func TestRotulosDeDesfechoSaoDistintos(t *testing.T) {
	vistos := map[string]bool{}
	for _, d := range []string{
		DesfechoSucesso, DesfechoIndisponivel, DesfechoTimeout,
		DesfechoRecusaRapida, DesfechoPoltronasIndisponiveis,
	} {
		if d == "" {
			t.Error("rótulo vazio")
		}
		if vistos[d] {
			t.Errorf("rótulo duplicado: %s", d)
		}
		vistos[d] = true
	}
	if len(vistos) != 5 {
		t.Fatalf("esperava 5 rótulos distintos, obteve %d", len(vistos))
	}
}
