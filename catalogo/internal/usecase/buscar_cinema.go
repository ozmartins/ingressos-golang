package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type BuscarCinema struct {
	Repo CinemaRepository
}

// Diferente da listagem, a busca por identificador não aplica recorte público:
// quem tem o id de um cinema desativado consegue vê-lo.
func (uc BuscarCinema) Executar(ctx context.Context, cinemaID string) (catalogo.Cinema, error) {
	return uc.Repo.BuscarPorID(ctx, cinemaID)
}
