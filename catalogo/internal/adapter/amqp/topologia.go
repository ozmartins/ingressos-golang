package amqp

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "cinema.eventos"

type Conexao struct {
	url string

	mu           sync.Mutex
	conn         *amqp.Connection
	canal        *amqp.Channel
	confirmacoes chan amqp.Confirmation
}

func Conectar(url string) (*Conexao, error) {
	c := &Conexao{url: url}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.abrir(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conexao) canalDePublicacao() (*amqp.Channel, chan amqp.Confirmation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.conn.IsClosed() || c.canal == nil || c.canal.IsClosed() {
		c.fechar()
		if err := c.abrir(); err != nil {
			return nil, nil, err
		}
	}
	return c.canal, c.confirmacoes, nil
}

func (c *Conexao) abrir() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("conectar ao broker: %w", err)
	}

	canal, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("abrir canal: %w", err)
	}
	if err := canal.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return fmt.Errorf("declarar exchange %s: %w", Exchange, err)
	}
	if err := canal.Confirm(false); err != nil {
		conn.Close()
		return fmt.Errorf("habilitar confirmações de publicação: %w", err)
	}

	c.conn = conn
	c.canal = canal
	c.confirmacoes = canal.NotifyPublish(make(chan amqp.Confirmation, 1))
	return nil
}

func (c *Conexao) fechar() {
	if c.conn != nil && !c.conn.IsClosed() {
		_ = c.conn.Close()
	}
	c.conn, c.canal, c.confirmacoes = nil, nil, nil
}

func (c *Conexao) Fechar() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fechar()
}
