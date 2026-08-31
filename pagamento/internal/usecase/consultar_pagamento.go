package usecase

import (
	"context"
	"errors"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

// ConsultarPagamento devolve o andamento do pagamento de uma reserva (FR-015).
type ConsultarPagamento struct{ Repo Repositorio }

// Executar aplica a guarda de dono: reserva de terceiro devolve exatamente o
// mesmo ErrNaoEncontrada de reserva inexistente. Responder algo distinguível
// confirmaria a existência da reserva alheia, o que a FR-017 proíbe.
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

// NaoEncontrada é o predicado que a camada HTTP usa para escolher o 404.
func NaoEncontrada(err error) bool { return errors.Is(err, ErrNaoEncontrada) }
