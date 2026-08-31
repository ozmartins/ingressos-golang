// Package config lê e valida a configuração do ambiente uma única vez, na
// largada. Variável ausente ou malformada impede o processo de subir: subir sem
// o segredo do QR emitiria ingressos que a portaria nunca validaria — dano
// silencioso e difícil de desfazer (research.md D11).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	PortaHTTP string

	DatabaseURL string

	AMQPURL            string
	AMQPExchange       string
	AMQPExchangeDLX    string
	AMQPFila           string
	AMQPFilaDLQ        string
	AMQPPrefetch       int
	AMQPLimiteEntregas int

	IngressoQRSegredo string
	PortariaAPIKey    string

	JWKSURL   string
	JWTIssuer string
	JWTAud    string

	NotificadorModo string

	OTLPEndpoint string
	NivelLog     string
}

// Modos do notificador simulado (research.md D6).
const (
	NotificarEnviar = "enviar"
	NotificarFalhar = "falhar"
)

type faltando []string

// Carregar lê o ambiente e devolve erro listando TODAS as chaves problemáticas
// de uma vez — descobrir uma por vez, reiniciando o processo, é desperdício.
func Carregar() (Config, error) {
	var f faltando
	c := Config{
		PortaHTTP:       comPadrao("PORTA_HTTP", "8080"),
		AMQPExchange:    comPadrao("AMQP_EXCHANGE", "cinema.eventos"),
		AMQPExchangeDLX: comPadrao("AMQP_EXCHANGE_DLX", "cinema.eventos.dlx"),
		AMQPFila:        comPadrao("AMQP_FILA_PAGAMENTO_SUCESSO", "notificacao.pagamento-sucesso"),
		OTLPEndpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		NivelLog:        comPadrao("NIVEL_LOG", "info"),
	}
	c.AMQPFilaDLQ = comPadrao("AMQP_FILA_PAGAMENTO_SUCESSO_DLQ", c.AMQPFila+".dlq")

	c.DatabaseURL = obrigatoria("DATABASE_URL", &f)
	c.AMQPURL = obrigatoria("AMQP_URL", &f)
	c.JWKSURL = obrigatoria("JWKS_URL", &f)
	c.JWTIssuer = obrigatoria("JWT_ISSUER", &f)
	c.JWTAud = obrigatoria("JWT_AUDIENCE", &f)
	c.IngressoQRSegredo = obrigatoria("INGRESSO_QR_SEGREDO", &f)
	c.PortariaAPIKey = obrigatoria("PORTARIA_API_KEY", &f)

	c.AMQPPrefetch = inteiro("AMQP_PREFETCH", 10, &f)
	c.AMQPLimiteEntregas = inteiro("AMQP_LIMITE_ENTREGAS", 3, &f)
	c.NotificadorModo = enumerada("NOTIFICADOR_MODO", NotificarEnviar,
		[]string{NotificarEnviar, NotificarFalhar}, &f)

	if c.AMQPPrefetch <= 0 {
		f = append(f, "AMQP_PREFETCH deve ser maior que zero")
	}
	if c.AMQPLimiteEntregas <= 0 {
		f = append(f, "AMQP_LIMITE_ENTREGAS deve ser maior que zero")
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

// inteiro trata valor malformado como erro de largada, e não como queda
// silenciosa no padrão: AMQP_LIMITE_ENTREGAS=abc virando 3 sem ninguém saber é
// pior do que não subir.
func inteiro(chave string, padrao int, f *faltando) int {
	v := os.Getenv(chave)
	if v == "" {
		return padrao
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*f = append(*f, chave+" deve ser um número inteiro, e veio "+strconv.Quote(v))
		return padrao
	}
	return n
}

func enumerada(chave, padrao string, aceitos []string, f *faltando) string {
	v := os.Getenv(chave)
	if v == "" {
		return padrao
	}
	for _, a := range aceitos {
		if v == a {
			return v
		}
	}
	*f = append(*f, chave+" deve ser um de ["+strings.Join(aceitos, ", ")+"], e veio "+strconv.Quote(v))
	return padrao
}
