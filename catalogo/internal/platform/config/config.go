// Package config lê e valida a configuração de ambiente uma única vez, na
// inicialização. Constituição, princípio II: nada embutido no artefato, e o
// processo recusa subir com configuração incompleta ou malformada.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// TimeoutEstoqueMaximo é o teto absoluto imposto pela especificação (FR-027).
// Configuração que peça mais que isso é recusada.
const TimeoutEstoqueMaximo = 2 * time.Second

type Config struct {
	HTTPPort string

	DatabaseURL string

	KeycloakIssuerURL string
	KeycloakAudience  string

	EstoqueGRPCAddr string
	EstoqueTimeout  time.Duration

	BreakerFalhasConsecutivas uint32
	BreakerIntervaloAberto    time.Duration

	PaginacaoTamanhoPadrao int
	PaginacaoTamanhoMaximo int

	OTLPEndpoint string
	LogLevel     string
}

type erroCampo struct {
	campo  string
	motivo string
}

// Carregar lê o ambiente e devolve a configuração validada. O erro agrega todos
// os campos problemáticos de uma vez: descobrir um por reinicialização é ruim
// para quem opera.
func Carregar() (Config, error) {
	var falhas []erroCampo

	c := Config{
		HTTPPort:          comPadrao("HTTP_PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		KeycloakIssuerURL: os.Getenv("KEYCLOAK_ISSUER_URL"),
		KeycloakAudience:  os.Getenv("KEYCLOAK_AUDIENCE"),
		EstoqueGRPCAddr:   os.Getenv("ESTOQUE_GRPC_ADDR"),
		OTLPEndpoint:      os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		LogLevel:          comPadrao("LOG_LEVEL", "info"),
	}

	for _, obrigatorio := range []struct {
		campo string
		valor string
	}{
		{"DATABASE_URL", c.DatabaseURL},
		{"KEYCLOAK_ISSUER_URL", c.KeycloakIssuerURL},
		{"KEYCLOAK_AUDIENCE", c.KeycloakAudience},
		{"ESTOQUE_GRPC_ADDR", c.EstoqueGRPCAddr},
	} {
		if strings.TrimSpace(obrigatorio.valor) == "" {
			falhas = append(falhas, erroCampo{obrigatorio.campo, "obrigatória e ausente"})
		}
	}

	timeout, err := duracao("ESTOQUE_TIMEOUT", 2*time.Second)
	switch {
	case err != nil:
		falhas = append(falhas, erroCampo{"ESTOQUE_TIMEOUT", err.Error()})
	case timeout <= 0:
		falhas = append(falhas, erroCampo{"ESTOQUE_TIMEOUT", "deve ser maior que zero"})
	case timeout > TimeoutEstoqueMaximo:
		falhas = append(falhas, erroCampo{"ESTOQUE_TIMEOUT", fmt.Sprintf("acima do teto de %s exigido pela especificação", TimeoutEstoqueMaximo)})
	default:
		c.EstoqueTimeout = timeout
	}

	if v, err := inteiro("BREAKER_FALHAS_CONSECUTIVAS", 5); err != nil {
		falhas = append(falhas, erroCampo{"BREAKER_FALHAS_CONSECUTIVAS", err.Error()})
	} else if v < 1 {
		falhas = append(falhas, erroCampo{"BREAKER_FALHAS_CONSECUTIVAS", "deve ser maior ou igual a 1"})
	} else {
		c.BreakerFalhasConsecutivas = uint32(v)
	}

	if v, err := duracao("BREAKER_INTERVALO_ABERTO", 30*time.Second); err != nil {
		falhas = append(falhas, erroCampo{"BREAKER_INTERVALO_ABERTO", err.Error()})
	} else if v <= 0 {
		falhas = append(falhas, erroCampo{"BREAKER_INTERVALO_ABERTO", "deve ser maior que zero"})
	} else {
		c.BreakerIntervaloAberto = v
	}

	padrao, errPadrao := inteiro("PAGINACAO_TAMANHO_PADRAO", 20)
	if errPadrao != nil {
		falhas = append(falhas, erroCampo{"PAGINACAO_TAMANHO_PADRAO", errPadrao.Error()})
	}
	maximo, errMaximo := inteiro("PAGINACAO_TAMANHO_MAXIMO", 100)
	if errMaximo != nil {
		falhas = append(falhas, erroCampo{"PAGINACAO_TAMANHO_MAXIMO", errMaximo.Error()})
	}
	if errPadrao == nil && errMaximo == nil {
		switch {
		case padrao < 1:
			falhas = append(falhas, erroCampo{"PAGINACAO_TAMANHO_PADRAO", "deve ser maior ou igual a 1"})
		case maximo < 1:
			falhas = append(falhas, erroCampo{"PAGINACAO_TAMANHO_MAXIMO", "deve ser maior ou igual a 1"})
		case padrao > maximo:
			falhas = append(falhas, erroCampo{"PAGINACAO_TAMANHO_PADRAO", "não pode ser maior que PAGINACAO_TAMANHO_MAXIMO"})
		default:
			c.PaginacaoTamanhoPadrao, c.PaginacaoTamanhoMaximo = padrao, maximo
		}
	}

	if len(falhas) > 0 {
		var b strings.Builder
		b.WriteString("configuração inválida:")
		for _, f := range falhas {
			fmt.Fprintf(&b, "\n  - %s: %s", f.campo, f.motivo)
		}
		return Config{}, errors.New(b.String())
	}
	return c, nil
}

func comPadrao(chave, padrao string) string {
	if v := strings.TrimSpace(os.Getenv(chave)); v != "" {
		return v
	}
	return padrao
}

func duracao(chave string, padrao time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(chave))
	if v == "" {
		return padrao, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("valor %q não é uma duração válida (ex.: 2s, 500ms)", v)
	}
	return d, nil
}

func inteiro(chave string, padrao int) (int, error) {
	v := strings.TrimSpace(os.Getenv(chave))
	if v == "" {
		return padrao, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("valor %q não é um número inteiro", v)
	}
	return n, nil
}
