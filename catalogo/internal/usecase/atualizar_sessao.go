package usecase

import (
	"context"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type AtualizarSessao struct {
	Sessoes SessaoRepository
	Filmes  FilmeRepository
	Salas   SalaRepository
	GerarID func() string
	Agora   func() time.Time
	// Ver `CriarSessao`: o fato viaja com o contexto de rastreamento da
	// requisição, porque quem o publica roda fora dela.
	TraceContextDe func(context.Context) map[string]string
}

// A atualização substitui a sessão inteira: o corpo descreve o estado final, e
// campo omitido volta a ser ausente. A sala é a exceção — ela faz parte do
// cadastro da sessão, não do estado que o PUT redesenha: omitida, permanece a
// atual; informada, precisa repetir a atual.
//
// Trocar a sala apagaria o chão sob quem já reservou: os assentos vendidos
// deixariam de existir, e reembolso não é coisa que este sistema faça. Quem
// precisa de outra sala cancela a sessão e cria outra. É a mesma regra que o
// `cinema_id` da sala já segue, pelo mesmo motivo.
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
