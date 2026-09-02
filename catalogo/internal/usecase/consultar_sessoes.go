package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type ConsultarSessoes struct {
	Repo SessaoRepository
}

func (uc ConsultarSessoes) Executar(ctx context.Context, filtro FiltroSessoes, req shared.PageRequest) (shared.Page[catalogo.SessaoDetalhada], error) {
	return uc.Repo.Consultar(ctx, filtro, req)
}
