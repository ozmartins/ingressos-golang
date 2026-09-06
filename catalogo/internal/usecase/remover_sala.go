package usecase

import "context"

type RemoverSala struct {
	Salas SalaRepository
}

// A remoção é lógica, como a do cinema: as sessões referenciam a sala, então
// tirá-la do cinema é desativá-la — some da listagem e a grade já gravada
// continua íntegra. O número volta a ficar livre para a sala que a substituir.
func (uc RemoverSala) Executar(ctx context.Context, salaID string) error {
	return uc.Salas.Desativar(ctx, salaID)
}
