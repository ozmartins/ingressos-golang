package usecase

import (
	"context"
	"fmt"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type ListarSalas struct {
	Cinemas CinemaRepository
	Salas   SalaRepository
}

func (uc ListarSalas) Executar(ctx context.Context, cinemaID string, req shared.PageRequest) (shared.Page[catalogo.Sala], error) {
	existe, err := uc.Cinemas.Existe(ctx, cinemaID)
	if err != nil {
		return shared.Page[catalogo.Sala]{}, err
	}
	if !existe {
		return shared.Page[catalogo.Sala]{}, fmt.Errorf("%w: cinema %s", shared.ErrNaoEncontrado, cinemaID)
	}
	return uc.Salas.ListarPorCinema(ctx, cinemaID, req)
}
