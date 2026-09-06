package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type BuscarFilme struct {
	Repo FilmeRepository
}

// Diferente da listagem, a busca por identificador não aplica recorte público:
// quem tem o id de um filme fora de cartaz consegue vê-lo.
func (uc BuscarFilme) Executar(ctx context.Context, filmeID string) (catalogo.Filme, error) {
	return uc.Repo.BuscarPorID(ctx, filmeID)
}
