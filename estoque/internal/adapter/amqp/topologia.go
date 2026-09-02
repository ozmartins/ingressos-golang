package amqp

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	Exchange    = "cinema.eventos"
	ExchangeDLX = "cinema.eventos.dlx"

	FilaPagamentoSucesso = "estoque.pagamento-sucesso"
	FilaPagamentoFalhou  = "estoque.pagamento-falhou"
	FilaSessaoCriada     = "estoque.sessao-criada"

	BindingPagamentoSucesso = "pagamento.sucesso"
	BindingPagamentoFalhou  = "pagamento.falhou"
	BindingSessaoCriada     = "sessao.criada"
)

var filas = map[string]string{
	FilaPagamentoSucesso: BindingPagamentoSucesso,
	FilaPagamentoFalhou:  BindingPagamentoFalhou,
	FilaSessaoCriada:     BindingSessaoCriada,
}

func NomeDLQ(fila string) string { return fila + ".dlq" }

type Conexao struct {
	conn         *amqp.Connection
	canalPublica *amqp.Channel
	confirmacoes chan amqp.Confirmation
	url          string
}

func Conectar(url string) (*Conexao, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("conectar ao broker: %w", err)
	}

	canal, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("abrir canal: %w", err)
	}
	if err := declararTopologia(canal); err != nil {
		conn.Close()
		return nil, err
	}

	if err := canal.Confirm(false); err != nil {
		conn.Close()
		return nil, fmt.Errorf("habilitar confirmações de publicação: %w", err)
	}

	return &Conexao{
		conn:         conn,
		canalPublica: canal,
		confirmacoes: canal.NotifyPublish(make(chan amqp.Confirmation, 1)),
		url:          url,
	}, nil
}

func declararTopologia(canal *amqp.Channel) error {
	for _, ex := range []string{Exchange, ExchangeDLX} {
		if err := canal.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("declarar exchange %s: %w", ex, err)
		}
	}

	for fila, binding := range filas {
		dlq := NomeDLQ(fila)
		if _, err := canal.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declarar %s: %w", dlq, err)
		}
		if err := canal.QueueBind(dlq, dlq, ExchangeDLX, false, nil); err != nil {
			return fmt.Errorf("vincular %s: %w", dlq, err)
		}

		if _, err := canal.QueueDeclare(fila, true, false, false, false, amqp.Table{
			"x-dead-letter-exchange":    ExchangeDLX,
			"x-dead-letter-routing-key": dlq,
		}); err != nil {
			return fmt.Errorf("declarar %s: %w", fila, err)
		}
		if err := canal.QueueBind(fila, binding, Exchange, false, nil); err != nil {
			return fmt.Errorf("vincular %s a %s: %w", fila, binding, err)
		}
	}
	return nil
}

func (c *Conexao) Verificar() error {
	if c.conn == nil || c.conn.IsClosed() {
		return fmt.Errorf("conexão com o broker fechada")
	}
	return nil
}

func (c *Conexao) Fechar() {
	if c.conn != nil && !c.conn.IsClosed() {
		_ = c.conn.Close()
	}
}

func (c *Conexao) Canal() (*amqp.Channel, error) { return c.conn.Channel() }
