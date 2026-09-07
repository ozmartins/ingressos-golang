package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const TimeoutEstoqueMaximo = 2 * time.Second

type Config struct {
	HTTPPort string

	DatabaseURL string

	KeycloakIssuerURL string
	KeycloakAudience  string

	EstoqueGRPCAddr string
	EstoqueTimeout  time.Duration

	// O canal com o estoque é mTLS: ele exige certificado de cliente, e é por
	// ele que identifica quem chama.
	EstoqueTLSCAFile   string
	EstoqueTLSCertFile string
	EstoqueTLSKeyFile  string

	RabbitMQURL     string
	OutboxIntervalo time.Duration
	OutboxLote      int

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

func Carregar() (Config, error) {
	var falhas []erroCampo

	c := Config{
		HTTPPort:          comPadrao("HTTP_PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		KeycloakIssuerURL: os.Getenv("KEYCLOAK_ISSUER_URL"),
		KeycloakAudience:  os.Getenv("KEYCLOAK_AUDIENCE"),
		EstoqueGRPCAddr:   os.Getenv("ESTOQUE_GRPC_ADDR"),
		RabbitMQURL:       os.Getenv("RABBITMQ_URL"),

		EstoqueTLSCAFile:   os.Getenv("ESTOQUE_TLS_CA_FILE"),
		EstoqueTLSCertFile: os.Getenv("ESTOQUE_TLS_CERT_FILE"),
		EstoqueTLSKeyFile:  os.Getenv("ESTOQUE_TLS_KEY_FILE"),
		OTLPEndpoint:       os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		LogLevel:           comPadrao("LOG_LEVEL", "info"),
	}

	for _, obrigatorio := range []struct {
		campo string
		valor string
	}{
		{"DATABASE_URL", c.DatabaseURL},
		{"KEYCLOAK_ISSUER_URL", c.KeycloakIssuerURL},
		{"KEYCLOAK_AUDIENCE", c.KeycloakAudience},
		{"ESTOQUE_GRPC_ADDR", c.EstoqueGRPCAddr},
		{"RABBITMQ_URL", c.RabbitMQURL},
		{"ESTOQUE_TLS_CA_FILE", c.EstoqueTLSCAFile},
		{"ESTOQUE_TLS_CERT_FILE", c.EstoqueTLSCertFile},
		{"ESTOQUE_TLS_KEY_FILE", c.EstoqueTLSKeyFile},
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

	if v, err := duracao("OUTBOX_INTERVALO", time.Second); err != nil {
		falhas = append(falhas, erroCampo{"OUTBOX_INTERVALO", err.Error()})
	} else if v <= 0 {
		falhas = append(falhas, erroCampo{"OUTBOX_INTERVALO", "deve ser maior que zero"})
	} else {
		c.OutboxIntervalo = v
	}

	if v, err := inteiro("OUTBOX_LOTE", 100); err != nil {
		falhas = append(falhas, erroCampo{"OUTBOX_LOTE", err.Error()})
	} else if v < 1 {
		falhas = append(falhas, erroCampo{"OUTBOX_LOTE", "deve ser maior ou igual a 1"})
	} else {
		c.OutboxLote = v
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
