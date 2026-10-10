package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type AtualizarSala struct {
	Cinemas CinemaRepository
	Salas   SalaRepository
}

func (uc AtualizarSala) Executar(ctx context.Context, salaID string, dados catalogo.DadosSala) (catalogo.Sala, error) {
	atual, err := uc.Salas.BuscarPorID(ctx, salaID)
	if err != nil {
		return catalogo.Sala{}, err
	}
	if dados.CinemaID == "" {
		dados.CinemaID = atual.CinemaID
	}
	if dados.CinemaID != atual.CinemaID {
		return catalogo.Sala{}, errConflito("a sala pertence ao cinema %s e não pode mudar de cinema", atual.CinemaID)
	}
	if dados.Fileiras == nil {
		dados.Fileiras = atual.Layout.Dados()
	}

	sala, err := catalogo.NovaSala(salaID, dados)
	if err != nil {
		return catalogo.Sala{}, err
	}
	if !sala.Layout.Igual(atual.Layout) {
		return catalogo.Sala{}, errConflito(
			"a planta da sala não pode ser redesenhada; desative esta sala e cadastre outra")
	}
	if err := conferirCinemaELiberdadeDoNumero(ctx, uc.Cinemas, uc.Salas, sala, salaID); err != nil {
		return catalogo.Sala{}, err
	}
	if err := uc.Salas.Atualizar(ctx, sala); err != nil {
		return catalogo.Sala{}, err
	}
	return sala, nil
}
