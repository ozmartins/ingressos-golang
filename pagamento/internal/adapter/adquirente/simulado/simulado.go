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

var ErrIndisponivel = errors.New("simulado: meio de pagamento indisponível")

type Adquirente struct {
	Demora       time.Duration
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
			return usecase.ResultadoCobranca{Desfecho: usecase.Aprovada, Codigo: "sim-" + uuid.NewString()}, nil
		case <-ctx.Done():
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
