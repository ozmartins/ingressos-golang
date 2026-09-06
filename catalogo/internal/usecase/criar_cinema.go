package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type CriarCinema struct {
	Repo    CinemaRepository
	GerarID func() string
}

func (uc CriarCinema) Executar(ctx context.Context, dados catalogo.DadosCinema) (catalogo.Cinema, error) {
	c, err := catalogo.NovoCinema(uc.GerarID(), dados)
	if err != nil {
		return catalogo.Cinema{}, err
	}
	if err := uc.Repo.Criar(ctx, c); err != nil {
		return catalogo.Cinema{}, err
	}
	return c, nil
}
