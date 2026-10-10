package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type BuscarCinema struct {
	Repo CinemaRepository
}

func (uc BuscarCinema) Executar(ctx context.Context, cinemaID string) (catalogo.Cinema, error) {
	return uc.Repo.BuscarPorID(ctx, cinemaID)
}
