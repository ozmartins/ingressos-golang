package usecase

import (
	"context"
	"fmt"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type ReservarPoltronas struct {
	Sessoes  CatalogoDeSessoes
	Bloqueio BloquearPoltronas
	Relogio  Relogio
}

func (uc ReservarPoltronas) Executar(ctx context.Context, sessaoID, usuarioID string, rotulos []string) (ResultadoBloqueio, error) {
	if sessaoID == "" {
		return ResultadoBloqueio{}, fmt.Errorf("%w: sessão não informada", shared.ErrSolicitacaoInvalida)
	}
	if len(rotulos) == 0 {
		return ResultadoBloqueio{}, fmt.Errorf("%w: nenhuma poltrona solicitada", shared.ErrSolicitacaoInvalida)
	}

	sessao, err := uc.Sessoes.BuscarSessao(ctx, sessaoID)
	if err != nil {
		return ResultadoBloqueio{}, err
	}
	if !sessao.AceitaReserva(uc.Relogio.Agora()) {
		err := fmt.Errorf("%w: sessão %s está %s", shared.ErrSessaoNaoReservavel, sessao.ID, sessao.Status)
		return ResultadoBloqueio{}, err
	}

	valor, err := sessao.ValorTotal(len(rotulos))
	if err != nil {
		return ResultadoBloqueio{}, err
	}

	return uc.Bloqueio.Executar(ctx, sessaoID, usuarioID, rotulos, valor)
}
