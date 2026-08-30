// Package usecase orquestra as regras de domínio. Não conhece banco, broker nem
// transporte: tudo que vem de fora entra por uma porta declarada aqui
// (constituição, princípio I).
package usecase

import (
	"context"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
)

// ResultadoTransicao diz o que aconteceu com um desfecho aplicado a uma reserva.
// Distinguir "aplicado" de "ignorado" é o que permite registrar divergência sem
// tratar reentrega como erro (FR-021, FR-022).
type ResultadoTransicao int

const (
	// TransicaoAplicada: a reserva estava pendente e mudou de estado.
	TransicaoAplicada ResultadoTransicao = iota
	// TransicaoIgnoradaDuplicata: a mensagem já havia sido processada.
	TransicaoIgnoradaDuplicata
	// TransicaoIgnoradaEstadoFinal: a reserva já estava finalizada.
	TransicaoIgnoradaEstadoFinal
	// TransicaoIgnoradaInexistente: não existe reserva com esse identificador.
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

// FatoPendente é o que a caixa de saída guarda para publicar depois. O contexto
// de rastreamento vai junto porque o publicador roda fora da requisição que
// produziu o fato (FR-044, SC-009).
type FatoPendente struct {
	MessageID    string
	RoutingKey   string
	Payload      []byte
	TraceContext map[string]string
}

// Concessao é o resultado de um bloqueio concedido.
type Concessao struct {
	Reserva reserva.Reserva
}

// RepositorioReservas concentra as operações atômicas sobre reserva e poltronas.
// Toda transição muda reserva e poltronas na mesma unidade indivisível (FR-015).
type RepositorioReservas interface {
	// Conceder aplica o protocolo de bloqueio: trava as poltronas em ordem
	// determinística, verifica disponibilidade e grava reserva, vínculos,
	// estado das poltronas e o fato — tudo ou nada.
	//
	// Devolve ErrPoltronasIndisponiveis quando alguma poltrona não está livre,
	// ErrPoltronaInexistente para rótulo desconhecido na sessão e
	// ErrSessaoNaoProvisionada quando a sessão não tem matriz.
	Conceder(ctx context.Context, sol reserva.Solicitacao, r reserva.Reserva, fato FatoPendente) error

	// Confirmar leva a reserva de PENDENTE para CONFIRMADA e as poltronas para
	// OCUPADA, registrando a idempotência na mesma transação.
	Confirmar(ctx context.Context, fila, messageID, reservaID string, agora time.Time) (ResultadoTransicao, error)

	// Cancelar leva a reserva para CANCELADA e as poltronas de volta para LIVRE.
	Cancelar(ctx context.Context, fila, messageID, reservaID string, agora time.Time) (ResultadoTransicao, error)

	// ExpirarVencidas invalida em lote as reservas cujo prazo venceu, devolvendo
	// os identificadores afetados. É a fonte autoritativa da expiração (D4).
	ExpirarVencidas(ctx context.Context, agora time.Time, limite int) ([]string, error)

	// ExpirarUma invalida uma reserva específica, se ainda estiver pendente e
	// vencida. É o que a notificação de prazo do Redis dispara.
	ExpirarUma(ctx context.Context, reservaID string, agora time.Time) (ResultadoTransicao, error)
}

// RepositorioPoltronas cuida da matriz de poltronas de cada sessão.
type RepositorioPoltronas interface {
	// MapaDaSessao devolve todas as poltronas da sessão, ordenadas por fileira
	// e número. Lista vazia significa sessão desconhecida.
	MapaDaSessao(ctx context.Context, sessaoID string) ([]poltrona.Poltrona, error)

	// ProvisionarMatriz cria as poltronas da sessão de forma indivisível e
	// idempotente: reprocessar o mesmo fato não duplica nem reinicia estados.
	ProvisionarMatriz(ctx context.Context, fila, messageID, sessaoID string, poltronas []poltrona.Poltrona) (ResultadoTransicao, error)
}

// IndiceDePrazo é o gatilho pronto de expiração. Não participa da correção do
// bloqueio: perdê-lo atrasa a liberação, nunca permite venda dupla (D2).
type IndiceDePrazo interface {
	Marcar(ctx context.Context, reservaID string, expiraEm time.Time) error
	Liberar(ctx context.Context, reservaID string) error
}

// Relogio é a porta de tempo, para que expiração seja testável sem espera real.
type Relogio interface {
	Agora() time.Time
}

// Registrador é o mínimo de log que os casos de uso precisam. Mantido aqui para
// que o núcleo não dependa da camada de plataforma.
type Registrador interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
