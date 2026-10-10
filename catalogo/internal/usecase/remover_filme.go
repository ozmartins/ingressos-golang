package usecase

import "context"

type RemoverFilme struct {
	Repo FilmeRepository
}

func (uc RemoverFilme) Executar(ctx context.Context, filmeID string) error {
	return uc.Repo.MarcarForaDeCartaz(ctx, filmeID)
}
