package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type BuscarSala struct {
	Salas SalaRepository
}

// Diferente da listagem, a busca por identificador não aplica recorte público:
// quem tem o id de uma sala desativada consegue vê-la.
func (uc BuscarSala) Executar(ctx context.Context, salaID string) (catalogo.Sala, error) {
	return uc.Salas.BuscarPorID(ctx, salaID)
}
