package usecase

import "context"

// CancelarReserva reage ao pagamento recusado devolvendo as poltronas ao estoque.
type CancelarReserva struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
	Log      Registrador
}

// Executar cancela a reserva e libera as poltronas, de forma idempotente.
func (uc CancelarReserva) Executar(ctx context.Context, fila, messageID, reservaID string) (ResultadoTransicao, error) {
	resultado, err := uc.Reservas.Cancelar(ctx, fila, messageID, reservaID, uc.Relogio.Agora())
	if err != nil {
		return resultado, err
	}

	switch resultado {
	case TransicaoAplicada:
		if uc.Prazo != nil {
			if err := uc.Prazo.Liberar(ctx, reservaID); err != nil {
				uc.Log.Warn("não foi possível limpar o índice de prazo",
					"reserva_id", reservaID, "erro", err.Error())
			}
		}
	case TransicaoIgnoradaEstadoFinal:
		// Chegou recusa para reserva já confirmada: as poltronas permanecem
		// ocupadas e a divergência fica registrada (FR-022).
		uc.Log.Warn("pagamento recusado para reserva já finalizada",
			"reserva_id", reservaID, "fila", fila, "acao", "ignorado")
	case TransicaoIgnoradaInexistente:
		uc.Log.Warn("pagamento recusado para reserva desconhecida",
			"reserva_id", reservaID, "fila", fila, "acao", "ignorado")
	case TransicaoIgnoradaDuplicata:
		uc.Log.Info("reentrega de pagamento recusado já processada",
			"reserva_id", reservaID, "fila", fila)
	}
	return resultado, nil
}
