package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type AtualizarCinema struct {
	Repo CinemaRepository
}

// A atualização substitui o cinema inteiro: o corpo descreve o estado final, e
// campo omitido volta a ser ausente.
func (uc AtualizarCinema) Executar(ctx context.Context, cinemaID string, dados catalogo.DadosCinema) (catalogo.Cinema, error) {
	c, err := catalogo.NovoCinema(cinemaID, dados)
	if err != nil {
		return catalogo.Cinema{}, err
	}
	if err := uc.Repo.Atualizar(ctx, c); err != nil {
		return catalogo.Cinema{}, err
	}
	return c, nil
}
