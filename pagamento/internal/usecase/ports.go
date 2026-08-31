// Package usecase orquestra o domínio. Depende apenas de interfaces declaradas
// aqui — nunca de adaptador, driver ou biblioteca de transporte.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

var (
	// ErrNaoEncontrada distingue ausência de erro de infraestrutura (FR-018).
	ErrNaoEncontrada = errors.New("usecase: transação não encontrada")
	// ErrJaFinalizada indica que outra execução finalizou a transação antes desta.
	ErrJaFinalizada = errors.New("usecase: transação já finalizada por outra execução")
)

// Relogio existe para que expiração e instante de pagamento sejam testáveis sem
// espera real (research.md D9).
type Relogio interface{ Agora() time.Time }

// GeradorID produz a identidade da transação.
type GeradorID interface{ Novo() string }

// Repositorio é a porta de persistência (data-model.md §4).
type Repositorio interface {
	// CriarSeAusente insere a transação, ou devolve a já existente para a mesma
	// reserva. É a porta de entrada da idempotência: criada=false significa que
	// outra entrega chegou primeiro e esta NÃO deve cobrar (FR-006).
	CriarSeAusente(ctx context.Context, t transacao.Transacao) (criada bool, atual transacao.Transacao, err error)

	// BuscarPorReserva devolve ErrNaoEncontrada quando não há transação.
	BuscarPorReserva(ctx context.Context, reservaID string) (transacao.Transacao, error)

	// Finalizar grava o estado final. A escrita é condicionada a PROCESSANDO:
	// zero linhas afetadas significa que outro já finalizou.
	Finalizar(ctx context.Context, t transacao.Transacao) error

	// ReivindicarCobranca concede o direito exclusivo de chamar o adquirente por
	// esta transação. Devolve false quando outra execução já o detém. É o que
	// impede duas entregas simultâneas de cobrarem a mesma reserva.
	ReivindicarCobranca(ctx context.Context, id string, agora time.Time) (bool, error)

	// LiberarCobranca devolve o direito de cobrar, e só deve ser chamada quando
	// o adquirente devolveu erro — ou seja, quando nada foi emitido.
	LiberarCobranca(ctx context.Context, id string, agora time.Time) error

	// MarcarAnunciado registra que o resultado foi publicado (FR-014).
	MarcarAnunciado(ctx context.Context, id string, agora time.Time) error
}

// Cobranca é o que se pede ao meio de pagamento.
type Cobranca struct {
	TransacaoID    string
	ReservaID      string
	ValorTotal     string
	FormaPagamento transacao.FormaPagamento
}

// DesfechoCobranca é o resultado de uma tentativa. São três, e o terceiro é o
// que sustenta a decisão D4: não saber é diferente de ter sido recusado.
type DesfechoCobranca int

const (
	// Aprovada — o meio de pagamento confirmou e devolveu uma referência.
	Aprovada DesfechoCobranca = iota
	// Recusada — o meio de pagamento negou, com motivo.
	Recusada
	// Indeterminada — não houve resposta no prazo. Não se sabe se a cobrança foi
	// efetivada; nada pode ser anunciado e nada pode ser recobrado (FR-008, FR-022).
	Indeterminada
)

// ResultadoCobranca é o que a porta do adquirente devolve.
type ResultadoCobranca struct {
	Desfecho DesfechoCobranca
	Codigo   string           // preenchido quando Aprovada
	Motivo   transacao.Motivo // preenchido quando Recusada
}

// Adquirente é a porta do meio de pagamento (research.md D7). O único adaptador
// desta entrega é o simulado; trocar por um real não altera nada acima daqui.
type Adquirente interface {
	Cobrar(ctx context.Context, c Cobranca) (ResultadoCobranca, error)
}

// Fato é um anúncio pronto para publicação.
type Fato struct {
	RoutingKey string
	MessageID  string
	Payload    []byte
}

// Publicador entrega fatos ao barramento. A implementação MUST confirmar a
// entrega com o broker antes de retornar sem erro (FR-014).
type Publicador interface {
	Publicar(ctx context.Context, f Fato) error
}
