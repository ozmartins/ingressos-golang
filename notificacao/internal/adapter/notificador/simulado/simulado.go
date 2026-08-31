// Package simulado é o único adaptador de aviso desta entrega. A integração com
// provedor real de e-mail, push ou SMS está fora do escopo (clarificação 1):
// o que a feature exige é que a tentativa aconteça, que o desfecho seja
// observado e que fique registrado.
//
// O modo de falha existe para que o teste da FR-025 possa exercitar o caminho
// de erro sem depender de terceiro.
package simulado

import (
	"context"
	"errors"
	"log/slog"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

// ErrCanalIndisponivel é o que o modo de falha devolve.
var ErrCanalIndisponivel = errors.New("simulado: canal de aviso indisponível")

// Notificador finge o envio e registra o que teria saído.
type Notificador struct {
	Falhar bool
	Log    *slog.Logger
}

func (n Notificador) Canal() aviso.Canal { return aviso.Email }

// Avisar nunca toca no código de acesso ao registrar (FR-021).
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
