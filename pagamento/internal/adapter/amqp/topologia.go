// Package amqp é o adaptador de mensageria: topologia, consumo e publicação.
package amqp

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Topologia é a declaração dos recursos AMQP (contracts/eventos.md §5).
type Topologia struct {
	Exchange       string
	ExchangeDLX    string
	Fila           string
	FilaDLQ        string
	Binding        string
	LimiteEntregas int
}

// Declarar cria exchanges e filas de forma idempotente. Chamada na largada: o
// processo recusa subir se a topologia não puder ser garantida, porque consumir
// de uma fila sem fila morta é perder mensagem em silêncio.
func (t Topologia) Declarar(ch *amqp.Channel) error {
	for _, ex := range []string{t.Exchange, t.ExchangeDLX} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("declarar exchange %s: %w", ex, err)
		}
	}

	// Fila morta primeiro: a fila principal aponta para ela.
	if _, err := ch.QueueDeclare(t.FilaDLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declarar fila morta %s: %w", t.FilaDLQ, err)
	}
	if err := ch.QueueBind(t.FilaDLQ, t.Fila, t.ExchangeDLX, false, nil); err != nil {
		return fmt.Errorf("vincular fila morta: %w", err)
	}

	// Fila quórum: x-delivery-limit é o que conta as entregas sem estado do lado
	// da aplicação (research.md D5). Esgotado o limite, o broker encaminha
	// sozinho para a fila morta.
	//
	// Tradução de vocabulário, medida contra o broker e não suposta: a FR-021
	// fala em TENTATIVAS (padrão 3); o RabbitMQ trata x-delivery-limit como
	// número de REENTREGAS, entregando limite+1 vezes (x-delivery-count vai de 0
	// a limite). Subtrai-se 1 para que o teto configurado signifique o que a spec
	// diz. Verificado em test/integration/quarentena_test.go.
	reentregas := t.LimiteEntregas - 1
	if reentregas < 0 {
		reentregas = 0
	}
	args := amqp.Table{
		"x-queue-type":              "quorum",
		"x-delivery-limit":          int32(reentregas),
		"x-dead-letter-exchange":    t.ExchangeDLX,
		"x-dead-letter-routing-key": t.Fila,
	}
	if _, err := ch.QueueDeclare(t.Fila, true, false, false, false, args); err != nil {
		return fmt.Errorf("declarar fila %s: %w", t.Fila, err)
	}
	if err := ch.QueueBind(t.Fila, t.Binding, t.Exchange, false, nil); err != nil {
		return fmt.Errorf("vincular fila %s: %w", t.Fila, err)
	}
	return nil
}
