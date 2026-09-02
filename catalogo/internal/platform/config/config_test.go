package config

import (
	"strings"
	"testing"
	"time"
)

func ambienteMinimo(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/catalogo")
	t.Setenv("KEYCLOAK_ISSUER_URL", "http://localhost:8081/realms/cinema")
	t.Setenv("KEYCLOAK_AUDIENCE", "cinema-app")
	t.Setenv("ESTOQUE_GRPC_ADDR", "localhost:50051")
}

func TestCarregarAplicaPadroes(t *testing.T) {
	ambienteMinimo(t)
	c, err := Carregar()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if c.HTTPPort != "8080" || c.LogLevel != "info" {
		t.Fatalf("padrões não aplicados: porta=%s log=%s", c.HTTPPort, c.LogLevel)
	}
	if c.EstoqueTimeout != 2*time.Second {
		t.Fatalf("timeout padrão esperado 2s, obteve %s", c.EstoqueTimeout)
	}
	if c.BreakerFalhasConsecutivas != 5 || c.BreakerIntervaloAberto != 30*time.Second {
		t.Fatalf("padrões de recusa rápida errados: %d/%s", c.BreakerFalhasConsecutivas, c.BreakerIntervaloAberto)
	}
	if c.PaginacaoTamanhoPadrao != 20 || c.PaginacaoTamanhoMaximo != 100 {
		t.Fatalf("padrões de paginação errados: %d/%d", c.PaginacaoTamanhoPadrao, c.PaginacaoTamanhoMaximo)
	}
}

func TestCarregarRecusaVariavelObrigatoriaAusente(t *testing.T) {
	ambienteMinimo(t)
	t.Setenv("ESTOQUE_GRPC_ADDR", "")
	_, err := Carregar()
	if err == nil {
		t.Fatal("esperava erro com variável obrigatória ausente")
	}
	if !strings.Contains(err.Error(), "ESTOQUE_GRPC_ADDR") {
		t.Fatalf("erro deveria nomear a variável faltante, obteve: %v", err)
	}
}

func TestCarregarAgregaTodasAsFalhas(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("KEYCLOAK_ISSUER_URL", "")
	t.Setenv("KEYCLOAK_AUDIENCE", "")
	t.Setenv("ESTOQUE_GRPC_ADDR", "")
	_, err := Carregar()
	if err == nil {
		t.Fatal("esperava erro")
	}
	for _, campo := range []string{"DATABASE_URL", "KEYCLOAK_ISSUER_URL", "KEYCLOAK_AUDIENCE", "ESTOQUE_GRPC_ADDR"} {
		if !strings.Contains(err.Error(), campo) {
			t.Errorf("erro não menciona %s: %v", campo, err)
		}
	}
}

func TestCarregarRecusaValorMalformado(t *testing.T) {
	ambienteMinimo(t)
	t.Setenv("ESTOQUE_TIMEOUT", "dois segundos")
	if _, err := Carregar(); err == nil || !strings.Contains(err.Error(), "ESTOQUE_TIMEOUT") {
		t.Fatalf("esperava recusa de duração malformada, obteve %v", err)
	}
}

func TestCarregarRecusaTimeoutAcimaDoTetoDaEspecificacao(t *testing.T) {
	ambienteMinimo(t)
	t.Setenv("ESTOQUE_TIMEOUT", "10s")
	if _, err := Carregar(); err == nil || !strings.Contains(err.Error(), "ESTOQUE_TIMEOUT") {
		t.Fatalf("esperava recusa de timeout acima do teto, obteve %v", err)
	}
}

func TestCarregarRecusaPaginacaoIncoerente(t *testing.T) {
	ambienteMinimo(t)
	t.Setenv("PAGINACAO_TAMANHO_PADRAO", "500")
	t.Setenv("PAGINACAO_TAMANHO_MAXIMO", "100")
	if _, err := Carregar(); err == nil || !strings.Contains(err.Error(), "PAGINACAO_TAMANHO_PADRAO") {
		t.Fatalf("esperava recusa de padrão acima do máximo, obteve %v", err)
	}
}
