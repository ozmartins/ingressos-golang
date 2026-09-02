package usecase

import "context"

type ExpirarReservas struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
	Log      Registrador

	LotePorVarredura int
}

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
