package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

var (
	ErrNaoEncontrada = errors.New("usecase: transação não encontrada")
	ErrJaFinalizada  = errors.New("usecase: transação já finalizada por outra execução")
)

type Relogio interface{ Agora() time.Time }

type GeradorID interface{ Novo() string }

type Repositorio interface {
	CriarSeAusente(ctx context.Context, t transacao.Transacao) (criada bool, atual transacao.Transacao, err error)

	BuscarPorReserva(ctx context.Context, reservaID string) (transacao.Transacao, error)

	Finalizar(ctx context.Context, t transacao.Transacao) error

	ReivindicarCobranca(ctx context.Context, id string, agora time.Time) (bool, error)

	LiberarCobranca(ctx context.Context, id string, agora time.Time) error

	MarcarAnunciado(ctx context.Context, id string, agora time.Time) error
}

type Cobranca struct {
	TransacaoID    string
	ReservaID      string
	ValorTotal     string
	FormaPagamento transacao.FormaPagamento
}

type DesfechoCobranca int

const (
	Aprovada DesfechoCobranca = iota
	Recusada
	Indeterminada
)

type ResultadoCobranca struct {
	Desfecho DesfechoCobranca
	Codigo   string
	Motivo   transacao.Motivo
}

type Adquirente interface {
	Cobrar(ctx context.Context, c Cobranca) (ResultadoCobranca, error)
}

type Fato struct {
	RoutingKey string
	MessageID  string
	Payload    []byte
}

type Publicador interface {
	Publicar(ctx context.Context, f Fato) error
}
