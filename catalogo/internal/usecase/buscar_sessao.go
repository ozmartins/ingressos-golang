package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type BuscarSessao struct {
	Repo SessaoRepository
}

func (uc BuscarSessao) Executar(ctx context.Context, sessaoID string) (catalogo.Sessao, error) {
	return uc.Repo.BuscarPorID(ctx, sessaoID)
}
