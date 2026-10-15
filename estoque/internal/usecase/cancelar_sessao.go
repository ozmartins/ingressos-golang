package usecase

import (
	"context"
	"encoding/json"
	"fmt"
)

type EventoSessaoCancelada struct {
	Evento     string `json:"evento"`
	Versao     int    `json:"versao"`
	OcorridoEm string `json:"ocorrido_em"`
	SessaoID   string `json:"sessao_id"`
}

func LerSessaoCancelada(corpo []byte) (EventoSessaoCancelada, error) {
	var e EventoSessaoCancelada
	if err := json.Unmarshal(corpo, &e); err != nil {
		return e, fmt.Errorf("corpo não é JSON válido: %w", err)
	}
	if e.SessaoID == "" {
		return e, fmt.Errorf("campo obrigatório ausente: sessao_id")
	}
	return e, nil
}

type CancelarSessao struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
}

func (uc CancelarSessao) Executar(ctx context.Context, fila, messageID, sessaoID string) (ResultadoTransicao, error) {
	desfecho, err := uc.Reservas.CancelarPendentesDaSessao(ctx, fila, messageID, sessaoID, uc.Relogio.Agora())
	if err != nil {
		return desfecho.Resultado, err
	}

	switch desfecho.Resultado {
	case TransicaoAplicada:
		for _, id := range desfecho.Canceladas {
			if uc.Prazo == nil {
				break
			}
			_ = uc.Prazo.Liberar(ctx, id)
		}

	case TransicaoIgnoradaDuplicata:

	case TransicaoIgnoradaInexistente:
	}
	return desfecho.Resultado, nil
}
