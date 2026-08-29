package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type ConsultarSessoes struct {
	Repo SessaoRepository
}

// Executar devolve uma página da grade.
//
// O recorte por situação não é opcional nem parametrizável: a consulta pública
// só oferece o que ainda é assistível (FR-016). Sessão cancelada ou finalizada
// na vitrine seria oferta de algo que não existe.
func (uc ConsultarSessoes) Executar(ctx context.Context, filtro FiltroSessoes, req shared.PageRequest) (shared.Page[catalogo.SessaoDetalhada], error) {
	return uc.Repo.Consultar(ctx, filtro, req)
}
