package simulado

import (
	"context"
	"errors"
	"log/slog"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

var ErrCanalIndisponivel = errors.New("simulado: canal de aviso indisponível")

type Notificador struct {
	Falhar bool
	Log    *slog.Logger
}

func (n Notificador) Canal() aviso.Canal { return aviso.Email }

func (n Notificador) Avisar(_ context.Context, ing ingresso.Ingresso) error {
	if n.Falhar {
		return ErrCanalIndisponivel
	}
	if n.Log != nil {
		n.Log.Info("aviso de confirmação simulado",
			"ingresso_id", ing.ID, "usuario_id", ing.UsuarioID, "canal", string(aviso.Email))
	}
	return nil
}
