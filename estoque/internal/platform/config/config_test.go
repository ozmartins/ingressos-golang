package config

import (
	"strings"
	"testing"
	"time"
)

func ambienteValido(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/estoque")
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	t.Setenv("TLS_CLIENT_AUTH", "off")
	t.Setenv("JWKS_URL", "http://keycloak:8081/realms/cinema/protocol/openid-connect/certs")
	t.Setenv("JWT_ISSUER", "http://keycloak:8081/realms/cinema")
	t.Setenv("JWT_AUDIENCE", "cinema-app")
}

func TestCarregarAplicaPadroes(t *testing.T) {
	ambienteValido(t)

	cfg, err := Carregar()
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if cfg.ReservaTTL != 10*time.Minute {
		t.Errorf("ReservaTTL = %v, esperado 10m", cfg.ReservaTTL)
	}
	if cfg.PoltronasMaxPorBloqueio != 10 {
		t.Errorf("PoltronasMaxPorBloqueio = %d, esperado 10", cfg.PoltronasMaxPorBloqueio)
	}
	if cfg.VarreduraExpiracaoInterval != 10*time.Second {
		t.Errorf("VarreduraExpiracaoInterval = %v, esperado 10s", cfg.VarreduraExpiracaoInterval)
	}
	if cfg.GRPCAddr != ":50051" {
		t.Errorf("GRPCAddr = %q, esperado :50051", cfg.GRPCAddr)
	}
	if cfg.HTTPAddr != ":8085" {
		t.Errorf("HTTPAddr = %q, esperado :8085", cfg.HTTPAddr)
	}
}

func TestCarregarRecusaVariavelObrigatoriaAusente(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("TLS_CLIENT_AUTH", "off")

	_, err := Carregar()
	if err == nil {
		t.Fatal("esperava recusa por variável obrigatória ausente")
	}
	for _, esperado := range []string{"DATABASE_URL", "RABBITMQ_URL"} {
		if !strings.Contains(err.Error(), esperado) {
			t.Errorf("erro não menciona %s: %v", esperado, err)
		}
	}
}

func TestCarregarRecusaValorMalformado(t *testing.T) {
	casos := map[string]struct{ chave, valor string }{
		"duração":      {"RESERVA_TTL", "dez minutos"},
		"inteiro":      {"POLTRONAS_MAX_POR_BLOQUEIO", "muitas"},
		"inteiro zero": {"AMQP_PREFETCH", "0"},
		"modo tls":     {"TLS_CLIENT_AUTH", "talvez"},
		"duração zero": {"VARREDURA_EXPIRACAO_INTERVALO", "0s"},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			ambienteValido(t)
			t.Setenv(caso.chave, caso.valor)

			if _, err := Carregar(); err == nil {
				t.Fatalf("esperava recusa para %s=%q", caso.chave, caso.valor)
			} else if !strings.Contains(err.Error(), caso.chave) {
				t.Errorf("erro não menciona %s: %v", caso.chave, err)
			}
		})
	}
}

func TestCarregarExigeMaterialTLSQuandoAutenticacaoObrigatoria(t *testing.T) {
	ambienteValido(t)
	t.Setenv("TLS_CLIENT_AUTH", "require")

	_, err := Carregar()
	if err == nil {
		t.Fatal("esperava recusa: mTLS exigido sem material de identidade")
	}
	for _, esperado := range []string{"TLS_CERT_FILE", "TLS_KEY_FILE", "TLS_CLIENT_CA_FILE"} {
		if !strings.Contains(err.Error(), esperado) {
			t.Errorf("erro não menciona %s: %v", esperado, err)
		}
	}
}
