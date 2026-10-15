package usecase

import "context"

type ExpirarReservas struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio

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
			_ = uc.Prazo.Liberar(ctx, id)
		}
	}

	return len(ids), nil
}

func (uc ExpirarReservas) ExpirarUma(ctx context.Context, reservaID string) (ResultadoTransicao, error) {
	resultado, err := uc.Reservas.ExpirarUma(ctx, reservaID, uc.Relogio.Agora())
	if err != nil {
		return resultado, err
	}

	return resultado, nil
}
