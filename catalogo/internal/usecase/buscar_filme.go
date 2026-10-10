package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type BuscarFilme struct {
	Repo FilmeRepository
}

func (uc BuscarFilme) Executar(ctx context.Context, filmeID string) (catalogo.Filme, error) {
	return uc.Repo.BuscarPorID(ctx, filmeID)
}
