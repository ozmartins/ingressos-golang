package config

import (
	"strings"
	"testing"
)

var obrigatorias = map[string]string{
	"DATABASE_URL": "postgres://u:p@localhost:5432/n?sslmode=disable",
	"RABBITMQ_URL": "amqp://guest:guest@localhost:5672/",
	"JWKS_URL":     "http://localhost:8081/certs",
	"JWT_ISSUER":   "http://localhost:8081/realms/cinema",
	"JWT_AUDIENCE": "cinema-app",
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
	if c.PortaHTTP != "8080" || c.AMQPExchange != "cinema.eventos" ||
		c.AMQPFilaReserva != "pagamento.reserva-criada" ||
		c.AMQPPrefetch != 10 || c.AMQPLimiteEntregas != 3 || c.VarreduraLote != 50 {
		t.Errorf("padrões inesperados: %+v", c)
	}
}

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
		"AMQP_PREFETCH":       "abc",
		"VARREDURA_LOTE":      "3.5",
		"ADQUIRENTE_TIMEOUT":  "rápido",
		"VARREDURA_INTERVALO": "2",
	}
	for chave, valor := range casos {
		t.Run(chave, func(t *testing.T) {
			ambienteCompleto(t)
			t.Setenv(chave, valor)
			_, err := Carregar()
			if err == nil {
				t.Fatalf("Carregar aceitou %s=%q", chave, valor)
			}
			if !strings.Contains(err.Error(), chave) {
				t.Errorf("o erro não nomeia %s: %v", chave, err)
			}
		})
	}
}

func TestValorNaoPositivoRecusado(t *testing.T) {
	casos := map[string]string{
		"AMQP_PREFETCH":        "0",
		"AMQP_LIMITE_ENTREGAS": "0",
		"ADQUIRENTE_TIMEOUT":   "0s",
	}
	for chave, valor := range casos {
		t.Run(chave, func(t *testing.T) {
			ambienteCompleto(t)
			t.Setenv(chave, valor)
			if _, err := Carregar(); err == nil {
				t.Errorf("Carregar aceitou %s=%s", chave, valor)
			}
		})
	}
}
