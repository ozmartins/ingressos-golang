package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type ListarFilmes struct {
	Repo FilmeRepository
}

// Executar devolve uma página de filmes.
//
// Sem filtro de situação, aplica o recorte público: fora de cartaz não aparece
// na vitrine (FR-008). Com filtro explícito, respeita o que foi pedido —
// inclusive FORA_DE_CARTAZ, que é um valor aceito, apenas não oferecido por
// omissão.
func (uc ListarFilmes) Executar(ctx context.Context, filtro FiltroFilmes, req shared.PageRequest) (shared.Page[catalogo.Filme], error) {
	return uc.Repo.Listar(ctx, filtro, catalogo.StatusPublicos, req)
}
