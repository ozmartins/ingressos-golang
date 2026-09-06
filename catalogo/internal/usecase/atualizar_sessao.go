package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type AtualizarSessao struct {
	Sessoes SessaoRepository
	Filmes  FilmeRepository
	Salas   SalaRepository
}

// A atualização substitui a sessão inteira: o corpo descreve o estado final, e
// campo omitido volta a ser ausente.
func (uc AtualizarSessao) Executar(ctx context.Context, sessaoID string, dados catalogo.DadosSessao) (catalogo.Sessao, error) {
	if _, err := uc.Sessoes.BuscarPorID(ctx, sessaoID); err != nil {
		return catalogo.Sessao{}, err
	}
	sessao, err := catalogo.NovaSessao(sessaoID, dados)
	if err != nil {
		return catalogo.Sessao{}, err
	}
	if err := conferirGrade(ctx, uc.Filmes, uc.Salas, uc.Sessoes, sessao, sessaoID); err != nil {
		return catalogo.Sessao{}, err
	}
	if err := uc.Sessoes.Atualizar(ctx, sessao); err != nil {
		return catalogo.Sessao{}, err
	}
	return sessao, nil
}
