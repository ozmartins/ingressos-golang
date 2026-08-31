// Package simulado é o único adaptador de adquirente desta entrega (clarificação
// Q1, research.md D7). O comportamento observável — aprovação, recusa, demora e
// indisponibilidade — é controlado por regras determinísticas sobre o valor, para
// que o roteiro manual consiga exercitar cada desfecho sem tocar no código.
//
// Trocar isto por um adquirente real não altera domínio, caso de uso nem banco.
package simulado

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

// ErrIndisponivel simula falha de infraestrutura do meio de pagamento — é erro,
// não desfecho: leva a mensagem de volta para a fila.
var ErrIndisponivel = errors.New("simulado: meio de pagamento indisponível")

// Adquirente implementa usecase.Adquirente com regras por centavos do valor:
//
//	.13 → recusado por cartão recusado
//	.66 → recusado por saldo insuficiente
//	.99 → demora além do prazo (leva a PENDENTE_VERIFICACAO)
//	.00 e demais → aprovado
//
// A regra vive nos centavos porque é o que o publicador manual consegue variar
// sem parâmetro novo, e porque mantém o roteiro de quickstart.md legível.
type Adquirente struct {
	// Demora é quanto o desfecho ".99" espera. Deve ser maior que o prazo
	// configurado do adquirente para produzir o caso indeterminado.
	Demora time.Duration
	// Indisponivel força erro de infraestrutura em toda cobrança, para o teste
	// de resiliência.
	Indisponivel bool
}

func (a Adquirente) Cobrar(ctx context.Context, c usecase.Cobranca) (usecase.ResultadoCobranca, error) {
	if a.Indisponivel {
		return usecase.ResultadoCobranca{}, ErrIndisponivel
	}

	switch centavos(c.ValorTotal) {
	case "13":
		return usecase.ResultadoCobranca{Desfecho: usecase.Recusada, Motivo: transacao.MotivoCartaoRecusado}, nil
	case "66":
		return usecase.ResultadoCobranca{Desfecho: usecase.Recusada, Motivo: transacao.MotivoSaldoInsuficiente}, nil
	case "99":
		demora := a.Demora
		if demora == 0 {
			demora = time.Hour
		}
		select {
		case <-time.After(demora):
			// Se o prazo do chamador for maior que a demora, a cobrança passa.
			return usecase.ResultadoCobranca{Desfecho: usecase.Aprovada, Codigo: "sim-" + uuid.NewString()}, nil
		case <-ctx.Done():
			// Prazo estourado: o chamador não sabe se cobramos. É exatamente o
			// desfecho indeterminado, e por isso ele é um caso do contrato.
			return usecase.ResultadoCobranca{Desfecho: usecase.Indeterminada}, nil
		}
	default:
		return usecase.ResultadoCobranca{Desfecho: usecase.Aprovada, Codigo: "sim-" + uuid.NewString()}, nil
	}
}

func centavos(valor string) string {
	i := strings.LastIndex(valor, ".")
	if i < 0 || len(valor)-i != 3 {
		return ""
	}
	return valor[i+1:]
}
