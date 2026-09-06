package usecase

import "context"

type RemoverFilme struct {
	Repo FilmeRepository
}

// A remoção é lógica: as sessões referenciam o filme, então tirá-lo do catálogo
// é marcá-lo FORA_DE_CARTAZ — some da listagem pública e a grade continua íntegra.
func (uc RemoverFilme) Executar(ctx context.Context, filmeID string) error {
	return uc.Repo.MarcarForaDeCartaz(ctx, filmeID)
}
