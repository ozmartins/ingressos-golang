//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	for _, esperado := range []string{"DATABASE_URL", "RABBITMQ_URL", "JWKS_URL", "JWT_ISSUER", "JWT_AUDIENCE"} {
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

	// A API REST carrega o conjunto de chaves na largada, como as demais APIs
	// do sistema. Aqui basta um conjunto vazio: o que este teste verifica é que
	// o processo sobe com o ambiente apontando para fora, não a validação de
	// token — essa vive nos testes do adaptador HTTP.
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer jwks.Close()

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
		"HTTP_ADDR=127.0.0.1:18085",
		"JWKS_URL=" + jwks.URL,
		"JWT_ISSUER=http://keycloak.test/realms/cinema",
		"JWT_AUDIENCE=cinema-app",
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
