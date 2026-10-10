package usecase

import "context"

type RemoverSala struct {
	Salas SalaRepository
}

func (uc RemoverSala) Executar(ctx context.Context, salaID string) error {
	return uc.Salas.Desativar(ctx, salaID)
}
