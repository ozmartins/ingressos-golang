package amqp

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// O exchange é compartilhado pelos quatro serviços. Este declara só ele: fila é
// de quem consome, e criar a fila do vizinho é decidir por ele quando ela existe
// e como ela é. A declaração é idempotente — quem subir primeiro cria.
const Exchange = "cinema.eventos"

// A conexão se refaz sozinha. O fato precisa ser reenviado até o broker aceitá-lo,
// e um broker que reinicia derruba a conexão: sem reabri-la, a caixa de saída
// nunca esvaziaria e só um restart do processo destravaria a fila.
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

// canalDePublicacao devolve um canal vivo e o fluxo de confirmações dele,
// reabrindo a conexão quando ela ou o canal caíram. Os dois andam juntos porque
// as confirmações são por canal: reabrir um sem o outro escutaria no vazio.
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

// Exige c.mu.
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
	// Sem confirmação do broker não há como saber que o fato saiu, e marcá-lo
	// como publicado às cegas é perdê-lo.
	if err := canal.Confirm(false); err != nil {
		conn.Close()
		return fmt.Errorf("habilitar confirmações de publicação: %w", err)
	}

	c.conn = conn
	c.canal = canal
	c.confirmacoes = canal.NotifyPublish(make(chan amqp.Confirmation, 1))
	return nil
}

// Exige c.mu.
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
