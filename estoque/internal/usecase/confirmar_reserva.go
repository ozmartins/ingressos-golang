package usecase

import (
	"context"
	"encoding/json"
	"fmt"
)

// DesfechoPagamento é o corpo dos fatos pagamento.sucesso e pagamento.falhou.
type DesfechoPagamento struct {
	Evento     string `json:"evento"`
	Versao     int    `json:"versao"`
	OcorridoEm string `json:"ocorrido_em"`
	ReservaID  string `json:"reserva_id"`
	Motivo     string `json:"motivo,omitempty"`
}

// LerDesfechoPagamento valida o corpo recebido. Corpo inválido é erro
// definitivo: a mensagem vai para a DLQ, não volta para a fila (FR-023).
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

// ConfirmarReserva reage ao pagamento aprovado tornando a posse definitiva.
type ConfirmarReserva struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
	Log      Registrador
}

// Executar aplica a confirmação de forma idempotente. Reentrega e chegada fora
// de ordem são inofensivas: a transição só parte de PENDENTE (FR-021, FR-022).
func (uc ConfirmarReserva) Executar(ctx context.Context, fila, messageID, reservaID string) (ResultadoTransicao, error) {
	resultado, err := uc.Reservas.Confirmar(ctx, fila, messageID, reservaID, uc.Relogio.Agora())
	if err != nil {
		return resultado, err
	}

	switch resultado {
	case TransicaoAplicada:
		// A poltrona agora é definitiva; o prazo deixa de importar.
		if uc.Prazo != nil {
			if err := uc.Prazo.Liberar(ctx, reservaID); err != nil {
				uc.Log.Warn("não foi possível limpar o índice de prazo",
					"reserva_id", reservaID, "erro", err.Error())
			}
		}
	case TransicaoIgnoradaEstadoFinal:
		// Divergência auditável: alguém pagou uma reserva já expirada ou
		// cancelada. Não sobrescrevemos o estado atual das poltronas (FR-022).
		uc.Log.Warn("pagamento aprovado para reserva já finalizada",
			"reserva_id", reservaID, "fila", fila, "acao", "ignorado")
	case TransicaoIgnoradaInexistente:
		uc.Log.Warn("pagamento aprovado para reserva desconhecida",
			"reserva_id", reservaID, "fila", fila, "acao", "ignorado")
	case TransicaoIgnoradaDuplicata:
		uc.Log.Info("reentrega de pagamento aprovado já processada",
			"reserva_id", reservaID, "fila", fila)
	}
	return resultado, nil
}
