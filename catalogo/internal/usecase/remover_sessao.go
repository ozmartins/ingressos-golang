package usecase

import (
	"context"
	"time"
)

type RemoverSessao struct {
	Repo  SessaoRepository
	Agora func() time.Time
	// Ver `CriarSessao`: o fato viaja com o contexto de rastreamento da
	// requisição, porque quem o publica roda fora dela.
	TraceContextDe func(context.Context) map[string]string
}

// A remoção é lógica: a sessão passa a CANCELADA e sai da grade, mas a linha
// permanece — as reservas já feitas apontam para ela.
//
// E é anunciada: quem tem estado preso à sessão precisa soltá-lo. Sem o anúncio,
// as reservas pendentes das poltronas dela seguiriam vivas no estoque, com o
// prazo correndo, para uma sessão que ninguém mais vê.
func (uc RemoverSessao) Executar(ctx context.Context, sessaoID string) error {
	fato, err := uc.anunciarCancelamento(ctx, sessaoID)
	if err != nil {
		return err
	}
	return uc.Repo.Cancelar(ctx, sessaoID, fato)
}
