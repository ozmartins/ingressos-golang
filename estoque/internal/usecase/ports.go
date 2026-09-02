package usecase

import (
	"context"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
)

type ResultadoTransicao int

const (
	TransicaoAplicada ResultadoTransicao = iota
	TransicaoIgnoradaDuplicata
	TransicaoIgnoradaEstadoFinal
	TransicaoIgnoradaInexistente
)

func (r ResultadoTransicao) String() string {
	switch r {
	case TransicaoAplicada:
		return "aplicada"
	case TransicaoIgnoradaDuplicata:
		return "ignorada-duplicata"
	case TransicaoIgnoradaEstadoFinal:
		return "ignorada-estado-final"
	default:
		return "ignorada-inexistente"
	}
}

type FatoPendente struct {
	MessageID    string
	RoutingKey   string
	Payload      []byte
	TraceContext map[string]string
}

type Concessao struct {
	Reserva reserva.Reserva
}

type RepositorioReservas interface {
	Conceder(ctx context.Context, sol reserva.Solicitacao, r reserva.Reserva, fato FatoPendente) error

	Confirmar(ctx context.Context, fila, messageID, reservaID string, agora time.Time) (ResultadoTransicao, error)

	Cancelar(ctx context.Context, fila, messageID, reservaID string, agora time.Time) (ResultadoTransicao, error)

	ExpirarVencidas(ctx context.Context, agora time.Time, limite int) ([]string, error)

	ExpirarUma(ctx context.Context, reservaID string, agora time.Time) (ResultadoTransicao, error)
}

type RepositorioPoltronas interface {
	MapaDaSessao(ctx context.Context, sessaoID string) ([]poltrona.Poltrona, error)

	ProvisionarMatriz(ctx context.Context, fila, messageID, sessaoID string, poltronas []poltrona.Poltrona) (ResultadoTransicao, error)
}

type IndiceDePrazo interface {
	Marcar(ctx context.Context, reservaID string, expiraEm time.Time) error
	Liberar(ctx context.Context, reservaID string) error
}

type Relogio interface {
	Agora() time.Time
}

type Registrador interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
