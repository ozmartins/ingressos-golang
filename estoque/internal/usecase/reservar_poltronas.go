package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type ReservarPoltronas struct {
	Sessoes  CatalogoDeSessoes
	Bloqueio BloquearPoltronas
	Relogio  Relogio
	Log      Registrador
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
		uc.auditar(sessaoID, usuarioID, rotulos, "", false, err)
		return ResultadoBloqueio{}, err
	}
	if !sessao.AceitaReserva(uc.Relogio.Agora()) {
		err := fmt.Errorf("%w: sessão %s está %s", shared.ErrSessaoNaoReservavel, sessao.ID, sessao.Status)
		uc.auditar(sessaoID, usuarioID, rotulos, "", false, err)
		return ResultadoBloqueio{}, err
	}

	valor, err := sessao.ValorTotal(len(rotulos))
	if err != nil {
		uc.auditar(sessaoID, usuarioID, rotulos, "", false, err)
		return ResultadoBloqueio{}, err
	}

	resultado, err := uc.Bloqueio.Executar(ctx, sessaoID, usuarioID, rotulos, valor)
	reservaID := ""
	if err == nil && resultado.Concedido {
		reservaID = resultado.Reserva.ID
	}
	uc.auditar(sessaoID, usuarioID, rotulos, reservaID, resultado.Concedido, err)
	return resultado, err
}

func (uc ReservarPoltronas) auditar(sessaoID, usuarioID string, rotulos []string, reservaID string, concedido bool, err error) {
	desfecho := "erro"
	switch {
	case err == nil && concedido:
		desfecho = "confirmada"
	case err == nil:
		desfecho = "indisponivel"
	case errors.Is(err, shared.ErrSessaoNaoReservavel):
		desfecho = "sessao_nao_reservavel"
	case errors.Is(err, shared.ErrCatalogoIndisponivel):
		desfecho = "catalogo_indisponivel"
	}
	uc.Log.Info("auditoria de reserva",
		"evento", "solicitacao_reserva", "usuario_id", usuarioID, "sessao_id", sessaoID,
		"poltronas", rotulos, "desfecho", desfecho, "reserva_id", reservaID)
}
