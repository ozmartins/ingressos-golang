package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type BuscarSala struct {
	Salas SalaRepository
}

func (uc BuscarSala) Executar(ctx context.Context, salaID string) (catalogo.Sala, error) {
	return uc.Salas.BuscarPorID(ctx, salaID)
}
