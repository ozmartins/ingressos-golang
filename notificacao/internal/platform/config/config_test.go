package config

import (
	"strings"
	"testing"
)

// obrigatorias é o conjunto que o processo exige para subir (contracts/erros.md §5).
var obrigatorias = map[string]string{
	"DATABASE_URL":        "postgres://u:p@localhost:5432/n?sslmode=disable",
	"AMQP_URL":            "amqp://guest:guest@localhost:5672/",
	"JWKS_URL":            "http://localhost:8081/certs",
	"JWT_ISSUER":          "http://localhost:8081/realms/cinema",
	"JWT_AUDIENCE":        "conta-cinema",
	"INGRESSO_QR_SEGREDO": "segredo",
	"PORTARIA_API_KEY":    "chave",
}

func ambienteCompleto(t *testing.T) {
	t.Helper()
	for k, v := range obrigatorias {
		t.Setenv(k, v)
	}
}

func TestCarregarComAmbienteCompleto(t *testing.T) {
	ambienteCompleto(t)
	c, err := Carregar()
	if err != nil {
		t.Fatalf("Carregar devolveu erro: %v", err)
	}
	// Padrões documentados no plano.
	if c.PortaHTTP != "8080" || c.AMQPExchange != "cinema.eventos" ||
		c.AMQPFila != "notificacao.pagamento-sucesso" ||
		c.AMQPPrefetch != 10 || c.AMQPLimiteEntregas != 3 ||
		c.NotificadorModo != NotificarEnviar {
		t.Errorf("padrões inesperados: %+v", c)
	}
	if c.AMQPFilaDLQ != "notificacao.pagamento-sucesso.dlq" {
		t.Errorf("fila morta = %q", c.AMQPFilaDLQ)
	}
}

// A ausência do segredo do QR sozinha já impede o processo de subir (D11).
func TestSegredoDoQRAusenteImpedeSubir(t *testing.T) {
	ambienteCompleto(t)
	t.Setenv("INGRESSO_QR_SEGREDO", "")

	_, err := Carregar()
	if err == nil {
		t.Fatal("Carregar aceitou ambiente sem INGRESSO_QR_SEGREDO")
	}
	if !strings.Contains(err.Error(), "INGRESSO_QR_SEGREDO") {
		t.Errorf("o erro não nomeia a chave faltante: %v", err)
	}
}

// O erro lista TODAS as chaves de uma vez, e não a primeira que falhou.
func TestErroListaTodasAsChavesFaltantes(t *testing.T) {
	for k := range obrigatorias {
		t.Setenv(k, "")
	}
	_, err := Carregar()
	if err == nil {
		t.Fatal("Carregar aceitou ambiente vazio")
	}
	for k := range obrigatorias {
		if !strings.Contains(err.Error(), k) {
			t.Errorf("o erro não menciona %s: %v", k, err)
		}
	}
}

func TestValorMalformadoNaoCaiNoPadrao(t *testing.T) {
	casos := map[string]string{
		"AMQP_PREFETCH":        "abc",
		"AMQP_LIMITE_ENTREGAS": "3.5",
		"NOTIFICADOR_MODO":     "talvez",
	}
	for chave, valor := range casos {
		t.Run(chave, func(t *testing.T) {
			ambienteCompleto(t)
			t.Setenv(chave, valor)
			_, err := Carregar()
			if err == nil {
				t.Fatalf("Carregar aceitou %s=%q em vez de falhar na largada", chave, valor)
			}
			if !strings.Contains(err.Error(), chave) {
				t.Errorf("o erro não nomeia %s: %v", chave, err)
			}
		})
	}
}

func TestValorNaoPositivoRecusado(t *testing.T) {
	for _, chave := range []string{"AMQP_PREFETCH", "AMQP_LIMITE_ENTREGAS"} {
		t.Run(chave, func(t *testing.T) {
			ambienteCompleto(t)
			t.Setenv(chave, "0")
			if _, err := Carregar(); err == nil {
				t.Errorf("Carregar aceitou %s=0", chave)
			}
		})
	}
}
