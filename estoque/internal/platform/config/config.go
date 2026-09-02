package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type ModoTLS string

const (
	TLSExigido   ModoTLS = "require"
	TLSDesligado ModoTLS = "off"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	RabbitMQURL string

	GRPCAddr  string
	AdminAddr string
	HTTPAddr  string

	JWKSURL     string
	JWTIssuer   string
	JWTAudience string

	TLSCertFile     string
	TLSKeyFile      string
	TLSClientCAFile string
	TLSClientAuth   ModoTLS

	ReservaTTL                 time.Duration
	PoltronasMaxPorBloqueio    int
	VarreduraExpiracaoInterval time.Duration
	AMQPPrefetch               int
	RetencaoMensagens          time.Duration

	OTLPEndpoint string
	LogLevel     string
}

type coletor struct{ erros []string }

func (c *coletor) falta(chave, motivo string) {
	c.erros = append(c.erros, fmt.Sprintf("%s: %s", chave, motivo))
}

func (c *coletor) obrigatoria(chave string) string {
	v := strings.TrimSpace(os.Getenv(chave))
	if v == "" {
		c.falta(chave, "variável obrigatória ausente")
	}
	return v
}

func (c *coletor) comPadrao(chave, padrao string) string {
	if v := strings.TrimSpace(os.Getenv(chave)); v != "" {
		return v
	}
	return padrao
}

func (c *coletor) duracao(chave, padrao string) time.Duration {
	bruto := c.comPadrao(chave, padrao)
	d, err := time.ParseDuration(bruto)
	if err != nil {
		c.falta(chave, fmt.Sprintf("duração malformada (%q)", bruto))
		return 0
	}
	if d <= 0 {
		c.falta(chave, "duração deve ser positiva")
	}
	return d
}

func (c *coletor) inteiro(chave, padrao string) int {
	bruto := c.comPadrao(chave, padrao)
	n, err := strconv.Atoi(bruto)
	if err != nil {
		c.falta(chave, fmt.Sprintf("inteiro malformado (%q)", bruto))
		return 0
	}
	if n <= 0 {
		c.falta(chave, "deve ser positivo")
	}
	return n
}

func Carregar() (*Config, error) {
	c := &coletor{}

	cfg := &Config{
		DatabaseURL: c.obrigatoria("DATABASE_URL"),
		RedisURL:    c.comPadrao("REDIS_URL", ""),
		RabbitMQURL: c.obrigatoria("RABBITMQ_URL"),

		GRPCAddr:  c.comPadrao("GRPC_ADDR", ":50051"),
		AdminAddr: c.comPadrao("ADMIN_ADDR", ":8090"),
		HTTPAddr:  c.comPadrao("HTTP_ADDR", ":8085"),

		JWKSURL:     c.obrigatoria("JWKS_URL"),
		JWTIssuer:   c.obrigatoria("JWT_ISSUER"),
		JWTAudience: c.obrigatoria("JWT_AUDIENCE"),

		TLSCertFile:     c.comPadrao("TLS_CERT_FILE", ""),
		TLSKeyFile:      c.comPadrao("TLS_KEY_FILE", ""),
		TLSClientCAFile: c.comPadrao("TLS_CLIENT_CA_FILE", ""),
		TLSClientAuth:   ModoTLS(c.comPadrao("TLS_CLIENT_AUTH", string(TLSExigido))),

		ReservaTTL:                 c.duracao("RESERVA_TTL", "10m"),
		PoltronasMaxPorBloqueio:    c.inteiro("POLTRONAS_MAX_POR_BLOQUEIO", "10"),
		VarreduraExpiracaoInterval: c.duracao("VARREDURA_EXPIRACAO_INTERVALO", "10s"),
		AMQPPrefetch:               c.inteiro("AMQP_PREFETCH", "32"),
		RetencaoMensagens:          c.duracao("RETENCAO_MENSAGENS_PROCESSADAS", "720h"),

		OTLPEndpoint: c.comPadrao("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		LogLevel:     c.comPadrao("LOG_LEVEL", "info"),
	}

	switch cfg.TLSClientAuth {
	case TLSExigido:
		if cfg.TLSCertFile == "" {
			c.falta("TLS_CERT_FILE", "obrigatório quando TLS_CLIENT_AUTH=require")
		}
		if cfg.TLSKeyFile == "" {
			c.falta("TLS_KEY_FILE", "obrigatório quando TLS_CLIENT_AUTH=require")
		}
		if cfg.TLSClientCAFile == "" {
			c.falta("TLS_CLIENT_CA_FILE", "obrigatório quando TLS_CLIENT_AUTH=require")
		}
	case TLSDesligado:
	default:
		c.falta("TLS_CLIENT_AUTH", fmt.Sprintf("valor não reconhecido (%q); use require ou off", cfg.TLSClientAuth))
	}

	if len(c.erros) > 0 {
		return nil, fmt.Errorf("configuração inválida:\n  - %s", strings.Join(c.erros, "\n  - "))
	}
	return cfg, nil
}

var ErrRedisAusente = errors.New("REDIS_URL não configurado; expiração fica só com a varredura")
