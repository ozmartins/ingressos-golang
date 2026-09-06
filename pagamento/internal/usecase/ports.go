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

	// Grava a forma escolhida e a passagem para PROCESSANDO. Condicionada ao
	// estado de origem: duas escolhas simultâneas, só uma vale.
	RegistrarEscolha(ctx context.Context, t transacao.Transacao) error

	// As que esperam cobrança: PROCESSANDO sem cobrança emitida. É o que o
	// varredor consome a cada tique.
	AguardandoCobranca(ctx context.Context, limite int) ([]transacao.Transacao, error)

	// Cancela em bloco as que esperam forma e já venceram — a reserva do outro
	// lado já liberou as poltronas, e ninguém mais vai pagar por elas.
	CancelarEsperasVencidas(ctx context.Context, agora time.Time, limite int) ([]transacao.Transacao, error)

	// As que já têm desfecho durável mas cujo anúncio não saiu. Um anúncio que
	// falha depois de o estado estar gravado não pode se perder: é o que a
	// entrega ao menos uma vez exige de quem publica.
	AnunciosPendentes(ctx context.Context, limite int) ([]transacao.Transacao, error)

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
