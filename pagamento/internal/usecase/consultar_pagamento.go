package usecase

import (
	"context"
	"errors"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

type ConsultarPagamento struct{ Repo Repositorio }

func (uc ConsultarPagamento) Executar(ctx context.Context, reservaID, usuarioID string) (transacao.Transacao, error) {
	t, err := uc.Repo.BuscarPorReserva(ctx, reservaID)
	if err != nil {
		return transacao.Transacao{}, err
	}
	if t.UsuarioID != usuarioID {
		return transacao.Transacao{}, ErrNaoEncontrada
	}
	return t, nil
}

func NaoEncontrada(err error) bool { return errors.Is(err, ErrNaoEncontrada) }
