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

// A sessão saiu da grade do catálogo. Sem este consumo, as reservas pendentes
// das poltronas dela seguiriam vivas — com o prazo correndo no índice — para uma
// sessão que ninguém mais pode comprar.
type CancelarSessao struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
	Log      Registrador
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
			if err := uc.Prazo.Liberar(ctx, id); err != nil {
				uc.Log.Warn("não foi possível limpar o índice de prazo",
					"reserva_id", id, "sessao_id", sessaoID, "erro", err.Error())
			}
		}
		uc.Log.Info("sessão cancelada; reservas pendentes liberadas",
			"sessao_id", sessaoID, "canceladas", len(desfecho.Canceladas))

		// Uma reserva confirmada é um ingresso pago, e não se apaga porque a
		// sessão caiu: apagá-la destruiria o registro da venda sem devolver o
		// dinheiro, e reembolso não é coisa que este sistema faça. Fica o
		// registro para quem precisa resolver isso fora do sistema.
		if desfecho.Confirmadas > 0 {
			uc.Log.Warn("sessão cancelada tinha ingressos confirmados; exigem tratamento fora do sistema",
				"sessao_id", sessaoID, "confirmadas", desfecho.Confirmadas)
		}

	case TransicaoIgnoradaDuplicata:
		uc.Log.Info("reentrega de cancelamento de sessão já processada",
			"sessao_id", sessaoID, "fila", fila)

	case TransicaoIgnoradaInexistente:
		// Sessão sem matriz aqui, ou sem reserva alguma: não há o que soltar, e
		// o cancelamento não tem por que falhar por isso.
		uc.Log.Info("cancelamento de sessão sem reservas pendentes",
			"sessao_id", sessaoID)
	}
	return desfecho.Resultado, nil
}
