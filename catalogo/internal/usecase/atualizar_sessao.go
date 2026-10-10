package usecase

import (
	"context"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type AtualizarSessao struct {
	Sessoes        SessaoRepository
	Filmes         FilmeRepository
	Salas          SalaRepository
	GerarID        func() string
	Agora          func() time.Time
	TraceContextDe func(context.Context) map[string]string
}

func (uc AtualizarSessao) Executar(ctx context.Context, sessaoID string, dados catalogo.DadosSessao) (catalogo.Sessao, error) {
	atual, err := uc.Sessoes.BuscarPorID(ctx, sessaoID)
	if err != nil {
		return catalogo.Sessao{}, err
	}
	if dados.SalaID == "" {
		dados.SalaID = atual.SalaID
	}
	if dados.SalaID != atual.SalaID {
		return catalogo.Sessao{}, errConflito(
			"a sessão ocorre na sala %s e não pode mudar de sala; cancele-a e crie outra", atual.SalaID)
	}

	sessao, err := catalogo.NovaSessao(sessaoID, dados)
	if err != nil {
		return catalogo.Sessao{}, err
	}
	if _, err := conferirGrade(ctx, uc.Filmes, uc.Salas, uc.Sessoes, sessao, sessaoID); err != nil {
		return catalogo.Sessao{}, err
	}

	fato, err := uc.anunciarAlteracao(ctx, sessao)
	if err != nil {
		return catalogo.Sessao{}, err
	}
	if err := uc.Sessoes.Atualizar(ctx, sessao, fato); err != nil {
		return catalogo.Sessao{}, err
	}
	return sessao, nil
}
