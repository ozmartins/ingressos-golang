package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type CriarFilme struct {
	Repo    FilmeRepository
	GerarID func() string
}

func (uc CriarFilme) Executar(ctx context.Context, dados catalogo.DadosFilme) (catalogo.Filme, error) {
	f, err := catalogo.NovoFilme(uc.GerarID(), dados)
	if err != nil {
		return catalogo.Filme{}, err
	}
	if err := uc.Repo.Criar(ctx, f); err != nil {
		return catalogo.Filme{}, err
	}
	return f, nil
}
