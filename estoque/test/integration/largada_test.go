//go:build integration

package integration

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestProcessoRecusaSubirSemConfiguracao cobre a segunda metade de SC-010: o
// artefato entregue não carrega configuração embutida, e o processo falha alto
// na largada em vez de descobrir o problema na primeira requisição do dia.
func TestProcessoRecusaSubirSemConfiguracao(t *testing.T) {
	binario := compilarBinario(t)

	cmd := exec.Command(binario)
	// Ambiente deliberadamente vazio.
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	saida, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("o processo subiu sem configuração obrigatória")
	}
	texto := string(saida)
	for _, esperado := range []string{"DATABASE_URL", "RABBITMQ_URL"} {
		if !strings.Contains(texto, esperado) {
			t.Errorf("a recusa não menciona %s:\n%s", esperado, texto)
		}
	}
}

// TestProcessoSobeApenasComConfiguracaoExterna cobre a primeira metade de
// SC-010: a mesma imagem sobe e atende com o ambiente apontando para as
// dependências, sem alteração no artefato.
func TestProcessoSobeApenasComConfiguracaoExterna(t *testing.T) {
	binario := compilarBinario(t)

	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()

	cmd := exec.CommandContext(ctx, binario)
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"DATABASE_URL=" + ambiente.DatabaseURL,
		"REDIS_URL=" + ambiente.RedisURL,
		"RABBITMQ_URL=" + ambiente.RabbitURL,
		"GRPC_ADDR=127.0.0.1:15051",
		"ADMIN_ADDR=127.0.0.1:18090",
		"TLS_CLIENT_AUTH=off",
		"LOG_LEVEL=error",
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("iniciar processo: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	// A instância precisa se declarar apta a receber tráfego.
	var ultimoStatus int
	prazo := time.Now().Add(30 * time.Second)
	for time.Now().Before(prazo) {
		resp, err := http.Get("http://127.0.0.1:18090/health/ready")
		if err == nil {
			ultimoStatus = resp.StatusCode
			resp.Body.Close()
			if ultimoStatus == http.StatusOK {
				return
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("instância não ficou pronta (último status: %d)", ultimoStatus)
}

func compilarBinario(t *testing.T) string {
	t.Helper()
	caminho := t.TempDir() + "/estoque"

	cmd := exec.Command("go", "build", "-o", caminho, "../../cmd/estoque")
	if saida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilar binário: %v\n%s", err, saida)
	}
	return caminho
}
