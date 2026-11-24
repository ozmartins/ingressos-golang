package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort string
	// GRPCAddr é onde o servidor gRPC escuta (consumido pelo estoque).
	GRPCAddr string
	// TLS do gRPC: certificado e chave do servidor (ambos ou nenhum) e, opcional,
	// a CA que assina os certificados de cliente (liga mTLS). Vazio = sem TLS.
	GRPCTLSCert     string
	GRPCTLSKey      string
	GRPCTLSClientCA string

	DatabaseURL string

	KeycloakIssuerURL string
	KeycloakAudience  string

	RabbitMQURL     string
	OutboxIntervalo time.Duration
	OutboxLote      int

	PaginacaoTamanhoPadrao int
	PaginacaoTamanhoMaximo int
}

type erroCampo struct {
	campo  string
	motivo string
}

func Carregar() (Config, error) {
	var falhas []erroCampo

	c := Config{
		HTTPPort:          comPadrao("HTTP_PORT", "8080"),
		GRPCAddr:          comPadrao("GRPC_ADDR", ":9091"),
		GRPCTLSCert:       strings.TrimSpace(os.Getenv("GRPC_TLS_CERT")),
		GRPCTLSKey:        strings.TrimSpace(os.Getenv("GRPC_TLS_KEY")),
		GRPCTLSClientCA:   strings.TrimSpace(os.Getenv("GRPC_TLS_CLIENT_CA")),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		KeycloakIssuerURL: os.Getenv("KEYCLOAK_ISSUER_URL"),
		KeycloakAudience:  os.Getenv("KEYCLOAK_AUDIENCE"),
		RabbitMQURL:       os.Getenv("RABBITMQ_URL"),
	}

	for _, obrigatorio := range []struct {
		campo string
		valor string
	}{
		{"DATABASE_URL", c.DatabaseURL},
		{"KEYCLOAK_ISSUER_URL", c.KeycloakIssuerURL},
		{"KEYCLOAK_AUDIENCE", c.KeycloakAudience},
		{"RABBITMQ_URL", c.RabbitMQURL},
	} {
		if strings.TrimSpace(obrigatorio.valor) == "" {
			falhas = append(falhas, erroCampo{obrigatorio.campo, "obrigatória e ausente"})
		}
	}

	if (c.GRPCTLSCert == "") != (c.GRPCTLSKey == "") {
		falhas = append(falhas, erroCampo{"GRPC_TLS_CERT/GRPC_TLS_KEY", "informe os dois ou nenhum"})
	}
	if c.GRPCTLSClientCA != "" && c.GRPCTLSCert == "" {
		falhas = append(falhas, erroCampo{"GRPC_TLS_CLIENT_CA", "exige GRPC_TLS_CERT e GRPC_TLS_KEY"})
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
