package usecase

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

const (
	RoutingKeySucesso = "pagamento.sucesso"
	RoutingKeyFalhou  = "pagamento.falhou"
	VersaoFato        = 1
)

type FatoPagamentoSucesso struct {
	Evento      string      `json:"evento"`
	Versao      int         `json:"versao"`
	OcorridoEm  string      `json:"ocorrido_em"`
	TransacaoID string      `json:"transacao_id"`
	ReservaID   string      `json:"reserva_id"`
	UsuarioID   string      `json:"usuario_id"`
	ValorTotal  json.Number `json:"valor_total"`
	PagoEm      string      `json:"pago_em"`
}

type FatoPagamentoFalhou struct {
	Evento      string `json:"evento"`
	Versao      int    `json:"versao"`
	OcorridoEm  string `json:"ocorrido_em"`
	TransacaoID string `json:"transacao_id"`
	ReservaID   string `json:"reserva_id"`
	UsuarioID   string `json:"usuario_id"`
	Motivo      string `json:"motivo"`
}

func MontarFato(t transacao.Transacao) (Fato, error) {
	if !t.Status.Anunciavel() {
		return Fato{}, fmt.Errorf("montar fato: estado %s não é anunciável", t.Status)
	}

	if t.Status == transacao.Pago {
		if t.PagoEm == nil {
			return Fato{}, fmt.Errorf("montar fato: transação paga sem instante de pagamento")
		}
		payload, err := json.Marshal(FatoPagamentoSucesso{
			Evento:      "PAGAMENTO_SUCESSO",
			Versao:      VersaoFato,
			OcorridoEm:  t.PagoEm.UTC().Format(time.RFC3339),
			TransacaoID: t.ID,
			ReservaID:   t.ReservaID,
			UsuarioID:   t.UsuarioID,
			ValorTotal:  json.Number(t.ValorTotal),
			PagoEm:      t.PagoEm.UTC().Format(time.RFC3339),
		})
		if err != nil {
			return Fato{}, err
		}
		return Fato{RoutingKey: RoutingKeySucesso, MessageID: t.ID, Payload: payload}, nil
	}

	payload, err := json.Marshal(FatoPagamentoFalhou{
		Evento:      "PAGAMENTO_FALHOU",
		Versao:      VersaoFato,
		OcorridoEm:  t.AtualizadoEm.UTC().Format(time.RFC3339),
		TransacaoID: t.ID,
		ReservaID:   t.ReservaID,
		UsuarioID:   t.UsuarioID,
		Motivo:      string(t.MotivoFalha),
	})
	if err != nil {
		return Fato{}, err
	}
	return Fato{RoutingKey: RoutingKeyFalhou, MessageID: t.ID, Payload: payload}, nil
}
