package usecase

import "context"

type CancelarReserva struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
}

func (uc CancelarReserva) Executar(ctx context.Context, fila, messageID, reservaID string) (ResultadoTransicao, error) {
	resultado, err := uc.Reservas.Cancelar(ctx, fila, messageID, reservaID, uc.Relogio.Agora())
	if err != nil {
		return resultado, err
	}

	switch resultado {
	case TransicaoAplicada:
		if uc.Prazo != nil {
			_ = uc.Prazo.Liberar(ctx, reservaID)
		}
	case TransicaoIgnoradaEstadoFinal:

	case TransicaoIgnoradaInexistente:

	case TransicaoIgnoradaDuplicata:
	}
	return resultado, nil
}
