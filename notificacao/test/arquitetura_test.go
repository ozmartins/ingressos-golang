package test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

const modulo = "github.com/oseias/ingressos-golang/notificacao"

type pacote struct {
	ImportPath  string
	Imports     []string
	TestImports []string
}

// O núcleo (domínio e casos de uso) não conhece adaptadores, plataforma nem
// infraestrutura: persistência, mensageria e transporte são detalhe de borda.
func TestNucleoNaoImportaInfraestrutura(t *testing.T) {
	saida, err := exec.Command("go", "list", "-json", modulo+"/internal/domain/...", modulo+"/internal/usecase/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}

	proibidos := []string{
		modulo + "/internal/adapter",
		modulo + "/internal/platform",
		"gorm.io",
		"github.com/jackc/pgx",
		"github.com/rabbitmq/amqp091-go",
		"go.opentelemetry.io/otel",
		"net/http",
	}

	decodificador := json.NewDecoder(strings.NewReader(string(saida)))
	verificados := 0
	for decodificador.More() {
		var p pacote
		if err := decodificador.Decode(&p); err != nil {
			t.Fatalf("decodificar go list: %v", err)
		}
		verificados++
		for _, imp := range append(append([]string{}, p.Imports...), p.TestImports...) {
			for _, proibido := range proibidos {
				if imp == proibido || strings.HasPrefix(imp, proibido+"/") {
					t.Errorf("%s importa %s: o núcleo não pode depender de infraestrutura", p.ImportPath, imp)
				}
			}
		}
	}
	if verificados == 0 {
		t.Fatal("nenhum pacote do núcleo foi verificado")
	}
}
