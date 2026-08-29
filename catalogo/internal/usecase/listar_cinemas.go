package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type ListarCinemas struct {
	Repo CinemaRepository
}

func (uc ListarCinemas) Executar(ctx context.Context, req shared.PageRequest) (shared.Page[catalogo.Cinema], error) {
	return uc.Repo.Listar(ctx, req)
}
