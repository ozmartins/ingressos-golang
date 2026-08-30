package usecase

import "context"

// ExpirarReservas invalida reservas não pagas. Tem dois gatilhos com o mesmo
// efeito: a varredura periódica (autoritativa) e a notificação de prazo do
// Redis (pronta). Ambos são idempotentes, então disparar os dois para a mesma
// reserva não causa efeito duplo (D4).
type ExpirarReservas struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
	Log      Registrador

	// LotePorVarredura limita quantas reservas cada passada invalida.
	LotePorVarredura int
}

// Varrer invalida todas as reservas vencidas e devolve quantas foram afetadas.
// É este caminho que cobre as reservas cujo prazo venceu enquanto o serviço
// estava fora do ar (FR-013, SC-008).
func (uc ExpirarReservas) Varrer(ctx context.Context) (int, error) {
	lote := uc.LotePorVarredura
	if lote <= 0 {
		lote = 500
	}

	ids, err := uc.Reservas.ExpirarVencidas(ctx, uc.Relogio.Agora(), lote)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if uc.Prazo != nil {
			if err := uc.Prazo.Liberar(ctx, id); err != nil {
				uc.Log.Warn("não foi possível limpar o índice de prazo",
					"reserva_id", id, "erro", err.Error())
			}
		}
	}
	if len(ids) > 0 {
		uc.Log.Info("reservas expiradas pela varredura", "quantidade", len(ids))
	}
	return len(ids), nil
}

// ExpirarUma invalida uma reserva específica. Chamado pela notificação de prazo;
// se a reserva já foi confirmada ou cancelada, nada acontece (FR-014).
func (uc ExpirarReservas) ExpirarUma(ctx context.Context, reservaID string) (ResultadoTransicao, error) {
	resultado, err := uc.Reservas.ExpirarUma(ctx, reservaID, uc.Relogio.Agora())
	if err != nil {
		return resultado, err
	}
	if resultado == TransicaoAplicada {
		uc.Log.Info("reserva expirada pelo índice de prazo", "reserva_id", reservaID)
	}
	return resultado, nil
}
