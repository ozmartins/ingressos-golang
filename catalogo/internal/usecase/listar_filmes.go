package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type ListarFilmes struct {
	Repo FilmeRepository
}

func (uc ListarFilmes) Executar(ctx context.Context, filtro FiltroFilmes, req shared.PageRequest) (shared.Page[catalogo.Filme], error) {
	return uc.Repo.Listar(ctx, filtro, catalogo.StatusPublicos, req)
}
