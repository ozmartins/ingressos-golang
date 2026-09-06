package usecase

import "context"

type RemoverCinema struct {
	Repo CinemaRepository
}

// A remoção é lógica: as salas referenciam o cinema, e as sessões referenciam as
// salas, então tirá-lo da rede é desativá-lo — some da listagem pública e a
// grade já gravada continua íntegra.
func (uc RemoverCinema) Executar(ctx context.Context, cinemaID string) error {
	return uc.Repo.Desativar(ctx, cinemaID)
}
