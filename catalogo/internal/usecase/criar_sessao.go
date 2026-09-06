package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type CriarSessao struct {
	Sessoes SessaoRepository
	Filmes  FilmeRepository
	Salas   SalaRepository
	GerarID func() string
}

func (uc CriarSessao) Executar(ctx context.Context, dados catalogo.DadosSessao) (catalogo.Sessao, error) {
	sessao, err := catalogo.NovaSessao(uc.GerarID(), dados)
	if err != nil {
		return catalogo.Sessao{}, err
	}
	if err := conferirGrade(ctx, uc.Filmes, uc.Salas, uc.Sessoes, sessao, ""); err != nil {
		return catalogo.Sessao{}, err
	}
	if err := uc.Sessoes.Criar(ctx, sessao); err != nil {
		return catalogo.Sessao{}, err
	}
	return sessao, nil
}

// O banco garante apenas que o filme e a sala existem. O que ele não sabe é que
// uma sala projeta um filme de cada vez: a janela da sessão vai do início até o
// fim do filme, e não pode alcançar outra sessão viva na mesma sala.
func conferirGrade(
	ctx context.Context,
	filmes FilmeRepository,
	salas SalaRepository,
	sessoes SessaoRepository,
	sessao catalogo.Sessao,
	excetoID string,
) error {
	filme, err := filmes.BuscarPorID(ctx, sessao.FilmeID)
	if err != nil {
		return err
	}
	if _, err := salas.BuscarPorID(ctx, sessao.SalaID); err != nil {
		return err
	}

	// Uma sessão cancelada ou já finalizada não ocupa a sala.
	if sessao.Status != catalogo.SessaoAgendada && sessao.Status != catalogo.SessaoEmAndamento {
		return nil
	}

	fim := sessao.FimPrevisto(filme.DuracaoMinutos)
	ocupada, err := sessoes.SalaOcupada(ctx, sessao.SalaID, sessao.DataHoraInicio, fim, excetoID)
	if err != nil {
		return err
	}
	if ocupada {
		return errConflito("a sala já tem uma sessão entre %s e %s",
			sessao.DataHoraInicio.Format("2006-01-02 15:04"), fim.Format("15:04"))
	}
	return nil
}
