package usecase

import "context"

type RemoverSessao struct {
	Repo SessaoRepository
}

// A remoção é lógica: a sessão passa a CANCELADA e sai da grade, mas a linha
// permanece — as reservas já feitas apontam para ela.
func (uc RemoverSessao) Executar(ctx context.Context, sessaoID string) error {
	return uc.Repo.Cancelar(ctx, sessaoID)
}
