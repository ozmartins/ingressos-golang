package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type BuscarSessao struct {
	Repo SessaoRepository
}

// Diferente da grade, a busca por identificador não aplica recorte de status:
// quem tem o id de uma sessão cancelada consegue vê-la.
func (uc BuscarSessao) Executar(ctx context.Context, sessaoID string) (catalogo.Sessao, error) {
	return uc.Repo.BuscarPorID(ctx, sessaoID)
}
