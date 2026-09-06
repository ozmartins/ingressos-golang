package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type BuscarSala struct {
	Salas SalaRepository
}

// Diferente da listagem, a busca por identificador não aplica recorte público:
// quem tem o id de uma sala desativada consegue vê-la.
func (uc BuscarSala) Executar(ctx context.Context, cinemaID, salaID string) (catalogo.Sala, error) {
	sala, err := uc.Salas.BuscarPorID(ctx, salaID)
	if err != nil {
		return catalogo.Sala{}, err
	}
	if err := conferirPosse(sala, cinemaID); err != nil {
		return catalogo.Sala{}, err
	}
	return sala, nil
}

// A sala vive dentro do cinema no caminho. Pedir uma sala pelo cinema errado é
// pedir algo que não existe ali — 404, não 403: o caminho inteiro é o endereço.
func conferirPosse(sala catalogo.Sala, cinemaID string) error {
	if sala.CinemaID != cinemaID {
		return shared.NaoEncontrado("sala", sala.ID)
	}
	return nil
}
