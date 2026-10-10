package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type ListarCinemas struct {
	Repo CinemaRepository
}

func (uc ListarCinemas) Executar(ctx context.Context, filtro FiltroCinemas, req shared.PageRequest) (shared.Page[catalogo.Cinema], error) {
	if filtro.Ativo == nil {
		ativo := true
		filtro.Ativo = &ativo
	}
	return uc.Repo.Listar(ctx, filtro, req)
}
