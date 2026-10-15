package usecase

import (
	"context"
	"encoding/json"
	"fmt"
)

type DesfechoPagamento struct {
	Evento     string `json:"evento"`
	Versao     int    `json:"versao"`
	OcorridoEm string `json:"ocorrido_em"`
	ReservaID  string `json:"reserva_id"`
	Motivo     string `json:"motivo,omitempty"`
}

func LerDesfechoPagamento(corpo []byte) (DesfechoPagamento, error) {
	var d DesfechoPagamento
	if err := json.Unmarshal(corpo, &d); err != nil {
		return d, fmt.Errorf("corpo não é JSON válido: %w", err)
	}
	if d.ReservaID == "" {
		return d, fmt.Errorf("campo obrigatório ausente: reserva_id")
	}
	return d, nil
}

type ConfirmarReserva struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
}

func (uc ConfirmarReserva) Executar(ctx context.Context, fila, messageID, reservaID string) (ResultadoTransicao, error) {
	resultado, err := uc.Reservas.Confirmar(ctx, fila, messageID, reservaID, uc.Relogio.Agora())
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
