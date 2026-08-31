package usecase

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

// Chaves de roteamento dos fatos publicados (contracts/eventos.md §2 e §3).
const (
	RoutingKeySucesso = "pagamento.sucesso"
	RoutingKeyFalhou  = "pagamento.falhou"
	VersaoFato        = 1
)

// FatoPagamentoSucesso é o payload de pagamento.sucesso. A forma é o contrato e
// só muda com versão nova.
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

// FatoPagamentoFalhou é o payload de pagamento.falhou.
type FatoPagamentoFalhou struct {
	Evento      string `json:"evento"`
	Versao      int    `json:"versao"`
	OcorridoEm  string `json:"ocorrido_em"`
	TransacaoID string `json:"transacao_id"`
	ReservaID   string `json:"reserva_id"`
	UsuarioID   string `json:"usuario_id"`
	Motivo      string `json:"motivo"`
}

// MontarFato deriva o anúncio inteiramente da transação gravada — é o que torna
// a republicação da FR-014 possível sem guardar o payload em lugar nenhum.
//
// Erra para estado não anunciável: PENDENTE_VERIFICACAO nunca vira fato.
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
