package test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

const modulo = "github.com/oseias/ingressos-golang/estoque"

type pacote struct {
	ImportPath  string
	Imports     []string
	TestImports []string
}

func TestNucleoNaoImportaInfraestrutura(t *testing.T) {
	saida, err := exec.Command("go", "list", "-json", modulo+"/internal/domain/...", modulo+"/internal/usecase/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}

	proibidos := []string{
		modulo + "/internal/adapter",
		modulo + "/internal/platform",
		modulo + "/gen/pb",
		"github.com/jackc/pgx",
		"github.com/rabbitmq/amqp091-go",
		"github.com/redis/go-redis",
		"google.golang.org/grpc",
		"go.opentelemetry.io/otel",
		"net/http",
	}

	decodificador := json.NewDecoder(strings.NewReader(string(saida)))
	pacotesVerificados := 0

	for decodificador.More() {
		var p pacote
		if err := decodificador.Decode(&p); err != nil {
			t.Fatalf("decodificar go list: %v", err)
		}
		pacotesVerificados++

		for _, importado := range p.Imports {
			for _, proibido := range proibidos {
				if strings.HasPrefix(importado, proibido) {
					t.Errorf("%s importa %s — o núcleo não pode depender de infraestrutura (constituição, princípio I)",
						p.ImportPath, importado)
				}
			}
		}
	}

	if pacotesVerificados == 0 {
		t.Fatal("nenhum pacote do núcleo foi verificado — o teste passaria em vazio")
	}
	t.Logf("pacotes do núcleo verificados: %d", pacotesVerificados)
}
