package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	PortaHTTP string

	DatabaseURL string

	AMQPURL            string
	AMQPExchange       string
	AMQPExchangeDLX    string
	AMQPFilaReserva    string
	AMQPFilaReservaDLQ string
	AMQPPrefetch       int
	AMQPLimiteEntregas int

	AdquirenteTimeout time.Duration

	JWKSURL   string
	JWTIssuer string
	JWTAud    string

	OTLPEndpoint string
	NivelLog     string
}

type faltando []string

func Carregar() (Config, error) {
	var f faltando
	c := Config{
		PortaHTTP:          comPadrao("PORTA_HTTP", "8080"),
		AMQPExchange:       comPadrao("AMQP_EXCHANGE", "cinema.eventos"),
		AMQPExchangeDLX:    comPadrao("AMQP_EXCHANGE_DLX", "cinema.eventos.dlx"),
		AMQPFilaReserva:    comPadrao("AMQP_FILA_RESERVA_CRIADA", "pagamento.reserva-criada"),
		AMQPFilaReservaDLQ: comPadrao("AMQP_FILA_RESERVA_CRIADA_DLQ", "pagamento.reserva-criada.dlq"),
		OTLPEndpoint:       os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		NivelLog:           comPadrao("NIVEL_LOG", "info"),
	}

	c.DatabaseURL = obrigatoria("DATABASE_URL", &f)
	c.AMQPURL = obrigatoria("AMQP_URL", &f)
	c.JWKSURL = obrigatoria("JWKS_URL", &f)
	c.JWTIssuer = obrigatoria("JWT_ISSUER", &f)
	c.JWTAud = obrigatoria("JWT_AUDIENCE", &f)

	c.AMQPPrefetch = inteiro("AMQP_PREFETCH", 10, &f)
	c.AMQPLimiteEntregas = inteiro("AMQP_LIMITE_ENTREGAS", 3, &f)
	c.AdquirenteTimeout = duracao("ADQUIRENTE_TIMEOUT", 10*time.Second, &f)

	if c.AMQPPrefetch <= 0 {
		f = append(f, "AMQP_PREFETCH deve ser maior que zero")
	}
	if c.AMQPLimiteEntregas <= 0 {
		f = append(f, "AMQP_LIMITE_ENTREGAS deve ser maior que zero")
	}
	if c.AdquirenteTimeout <= 0 {
		f = append(f, "ADQUIRENTE_TIMEOUT deve ser maior que zero")
	}

	if len(f) > 0 {
		return Config{}, fmt.Errorf("configuração inválida:\n  - %s", strings.Join(f, "\n  - "))
	}
	return c, nil
}

func comPadrao(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

func obrigatoria(chave string, f *faltando) string {
	v := os.Getenv(chave)
	if v == "" {
		*f = append(*f, chave+" é obrigatória e está ausente")
	}
	return v
}

func inteiro(chave string, padrao int, f *faltando) int {
	v := os.Getenv(chave)
	if v == "" {
		return padrao
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*f = append(*f, chave+" deve ser um inteiro, veio "+strconv.Quote(v))
		return padrao
	}
	return n
}

func duracao(chave string, padrao time.Duration, f *faltando) time.Duration {
	v := os.Getenv(chave)
	if v == "" {
		return padrao
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		*f = append(*f, chave+" deve ser uma duração (ex.: 10s), veio "+strconv.Quote(v))
		return padrao
	}
	return d
}
