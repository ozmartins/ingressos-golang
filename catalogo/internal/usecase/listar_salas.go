package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type ListarSalas struct {
	Cinemas CinemaRepository
	Salas   SalaRepository
}

// Sem `CinemaID` a listagem é da rede inteira. Com ele, o cinema precisa
// existir: um recorte por um cinema que não existe é 404, não uma página vazia.
func (uc ListarSalas) Executar(ctx context.Context, filtro FiltroSalas, req shared.PageRequest) (shared.Page[catalogo.Sala], error) {
	if filtro.CinemaID != "" {
		existe, err := uc.Cinemas.Existe(ctx, filtro.CinemaID)
		if err != nil {
			return shared.Page[catalogo.Sala]{}, err
		}
		if !existe {
			return shared.Page[catalogo.Sala]{}, shared.NaoEncontrado("cinema", filtro.CinemaID)
		}
	}
	return uc.Salas.Listar(ctx, filtro, req)
}
