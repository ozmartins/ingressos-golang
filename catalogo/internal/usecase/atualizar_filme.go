package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type AtualizarFilme struct {
	Repo FilmeRepository
}

func (uc AtualizarFilme) Executar(ctx context.Context, filmeID string, dados catalogo.DadosFilme) (catalogo.Filme, error) {
	f, err := catalogo.NovoFilme(filmeID, dados)
	if err != nil {
		return catalogo.Filme{}, err
	}
	if err := uc.Repo.Atualizar(ctx, f); err != nil {
		return catalogo.Filme{}, err
	}
	return f, nil
}
