package usecase

import (
	"context"
	"time"
)

type RemoverSessao struct {
	Repo  SessaoRepository
	Agora func() time.Time
}

func (uc RemoverSessao) Executar(ctx context.Context, sessaoID string) error {
	fato, err := uc.anunciarCancelamento(ctx, sessaoID)
	if err != nil {
		return err
	}
	return uc.Repo.Cancelar(ctx, sessaoID, fato)
}
