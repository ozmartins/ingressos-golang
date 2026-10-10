package usecase

import "context"

type RemoverCinema struct {
	Repo CinemaRepository
}

func (uc RemoverCinema) Executar(ctx context.Context, cinemaID string) error {
	return uc.Repo.Desativar(ctx, cinemaID)
}
