package simulado

import (
	"context"
	"errors"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

var ErrCanalIndisponivel = errors.New("simulado: canal de aviso indisponível")

type Notificador struct {
	Falhar bool
}

func (n Notificador) Canal() aviso.Canal { return aviso.Email }

func (n Notificador) Avisar(_ context.Context, ing ingresso.Ingresso) error {
	if n.Falhar {
		return ErrCanalIndisponivel
	}

	return nil
}
