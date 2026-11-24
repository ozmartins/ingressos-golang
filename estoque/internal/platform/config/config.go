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
	DatabaseURL string
	RedisURL    string
	RabbitMQURL string

	AdminAddr string
	HTTPAddr  string

	JWKSURL     string
	JWTIssuer   string
	JWTAudience string

	// CatalogoGRPCAddr é o endereço gRPC do catálogo (host:porta).
	CatalogoGRPCAddr string
	CatalogoTimeout  time.Duration
	// TLS do gRPC para o catálogo: CA que assina o servidor e, opcional, o par
	// certificado/chave do cliente (mTLS). CA vazia = sem TLS.
	CatalogoTLSCA   string
	CatalogoTLSCert string
	CatalogoTLSKey  string
	// Identidade do estoque junto ao catálogo (client credentials no Keycloak).
	KeycloakTokenURL      string
	CatalogoClientID      string
	CatalogoClientSegredo string
	// Validade da sessão em cache usada como fallback quando o catálogo cai.
	CatalogoCacheTTL time.Duration
	// Limite de taxa das reservas, por usuário: fichas repostas por segundo e rajada.
	RateLimitRPS   int
	RateLimitBurst int
	// Retry da consulta ao catálogo: total de tentativas e espera base do backoff.
	CatalogoTentativas int
	CatalogoEsperaBase time.Duration
	BreakerFalhas      int
	BreakerIntervalo   time.Duration

	ReservaTTL                 time.Duration
	PoltronasMaxPorBloqueio    int
	VarreduraExpiracaoInterval time.Duration
	AMQPPrefetch               int
	RetencaoMensagens          time.Duration
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

		AdminAddr: c.comPadrao("ADMIN_ADDR", ":8090"),
		HTTPAddr:  c.comPadrao("HTTP_ADDR", ":8085"),

		JWKSURL:     c.obrigatoria("JWKS_URL"),
		JWTIssuer:   c.obrigatoria("JWT_ISSUER"),
		JWTAudience: c.obrigatoria("JWT_AUDIENCE"),

		KeycloakTokenURL:      c.obrigatoria("KEYCLOAK_TOKEN_URL"),
		CatalogoClientID:      c.comPadrao("CATALOGO_CLIENT_ID", "estoque-svc"),
		CatalogoClientSegredo: c.obrigatoria("CATALOGO_CLIENT_SECRET"),

		CatalogoCacheTTL: c.duracao("CATALOGO_CACHE_TTL", "5m"),

		RateLimitRPS:   c.inteiro("RATE_LIMIT_RPS", "5"),
		RateLimitBurst: c.inteiro("RATE_LIMIT_BURST", "10"),

		CatalogoGRPCAddr:   c.obrigatoria("CATALOGO_GRPC_ADDR"),
		CatalogoTLSCA:      c.comPadrao("CATALOGO_GRPC_TLS_CA", ""),
		CatalogoTLSCert:    c.comPadrao("CATALOGO_GRPC_TLS_CERT", ""),
		CatalogoTLSKey:     c.comPadrao("CATALOGO_GRPC_TLS_KEY", ""),
		CatalogoTimeout:    c.duracao("CATALOGO_TIMEOUT", "2s"),
		CatalogoTentativas: c.inteiro("CATALOGO_TENTATIVAS", "3"),
		CatalogoEsperaBase: c.duracao("CATALOGO_ESPERA_BASE", "100ms"),
		BreakerFalhas:      c.inteiro("BREAKER_FALHAS_CONSECUTIVAS", "5"),
		BreakerIntervalo:   c.duracao("BREAKER_INTERVALO_ABERTO", "30s"),

		ReservaTTL:                 c.duracao("RESERVA_TTL", "10m"),
		PoltronasMaxPorBloqueio:    c.inteiro("POLTRONAS_MAX_POR_BLOQUEIO", "10"),
		VarreduraExpiracaoInterval: c.duracao("VARREDURA_EXPIRACAO_INTERVALO", "10s"),
		AMQPPrefetch:               c.inteiro("AMQP_PREFETCH", "32"),
		RetencaoMensagens:          c.duracao("RETENCAO_MENSAGENS_PROCESSADAS", "720h"),
	}

	if len(c.erros) > 0 {
		return nil, fmt.Errorf("configuração inválida:\n  - %s", strings.Join(c.erros, "\n  - "))
	}
	return cfg, nil
}

var ErrRedisAusente = errors.New("REDIS_URL não configurado; expiração fica só com a varredura")
