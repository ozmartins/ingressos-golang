// Package http expõe a única superfície síncrona do serviço: a consulta do
// andamento do pagamento, mais as duas sondas de saúde.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/health"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

// Códigos de erro do contrato (contracts/erros.md). O par status+codigo é o que
// é contrato; a mensagem é texto humano e pode mudar de redação.
const (
	CodReservaIDInvalido  = "RESERVA_ID_INVALIDO"
	CodCredencialInvalida = "CREDENCIAL_INVALIDA"
	CodNaoEncontrado      = "PAGAMENTO_NAO_ENCONTRADO"
	CodIndisponivel       = "SERVICO_INDISPONIVEL"
)

type erroResposta struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

type pagamentoResposta struct {
	TransacaoID    string      `json:"transacao_id"`
	ReservaID      string      `json:"reserva_id"`
	Status         string      `json:"status"`
	ValorTotal     json.Number `json:"valor_total"`
	FormaPagamento string      `json:"forma_pagamento"`
	CriadoEm       string      `json:"criado_em"`
}

// API reúne as dependências das rotas.
type API struct {
	Consulta  usecase.ConsultarPagamento
	Auth      *Autenticador
	Prontidao *health.Prontidao
	Log       *slog.Logger
}

// Rotas monta o roteador. São três caminhos: net/http basta, e um roteador de
// terceiro não se pagaria aqui (princípio I).
func (a *API) Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/pagamentos/reserva/{reserva_id}", a.consultar)
	mux.HandleFunc("GET /api/v1/health/live", a.vivo)
	mux.HandleFunc("GET /api/v1/health/ready", a.pronto)
	return mux
}

func (a *API) consultar(w http.ResponseWriter, r *http.Request) {
	sub, err := a.Auth.Identificar(r)
	if err != nil {
		responderErro(w, http.StatusUnauthorized, CodCredencialInvalida, "credencial ausente ou inválida")
		return
	}

	reservaID := r.PathValue("reserva_id")
	if _, err := uuid.Parse(reservaID); err != nil {
		responderErro(w, http.StatusBadRequest, CodReservaIDInvalido, "reserva_id deve ser um UUID")
		return
	}

	t, err := a.Consulta.Executar(r.Context(), reservaID, sub)
	switch {
	case usecase.NaoEncontrada(err):
		// Reserva inexistente e reserva de terceiro respondem exatamente igual.
		// A distinção fica no registro interno, nunca na resposta (FR-017).
		a.Log.Info("consulta sem resultado visível", "reserva_id", reservaID, "sub", sub)
		responderErro(w, http.StatusNotFound, CodNaoEncontrado, "não há pagamento para essa reserva")
	case err != nil:
		a.Log.Error("falha ao consultar pagamento", "reserva_id", reservaID, "erro", err)
		responderErro(w, http.StatusServiceUnavailable, CodIndisponivel, "serviço indisponível no momento")
	default:
		responderJSON(w, http.StatusOK, paraResposta(t))
	}
}

// paraResposta expõe só o que o contrato declara. motivo_falha e
// codigo_transacao_gateway ficam de fora de propósito (contracts/openapi.yaml).
func paraResposta(t transacao.Transacao) pagamentoResposta {
	return pagamentoResposta{
		TransacaoID:    t.ID,
		ReservaID:      t.ReservaID,
		Status:         string(t.Status),
		ValorTotal:     json.Number(t.ValorTotal),
		FormaPagamento: string(t.FormaPagamento),
		CriadoEm:       t.CriadoEm.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (a *API) vivo(w http.ResponseWriter, _ *http.Request) {
	responderJSON(w, http.StatusOK, map[string]string{"status": "vivo"})
}

func (a *API) pronto(w http.ResponseWriter, r *http.Request) {
	if nome, err := a.Prontidao.Verificar(r.Context()); err != nil {
		a.Log.Warn("dependência indisponível", "dependencia", nome, "erro", err)
		responderJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "indisponível", "dependencia": nome,
		})
		return
	}
	responderJSON(w, http.StatusOK, map[string]string{"status": "pronto"})
}

func responderJSON(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
}

func responderErro(w http.ResponseWriter, status int, codigo, mensagem string) {
	responderJSON(w, status, erroResposta{Codigo: codigo, Mensagem: mensagem})
}

var errSemCredencial = errors.New("http: credencial ausente")

func tokenDoCabecalho(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", errSemCredencial
	}
	partes := strings.SplitN(h, " ", 2)
	if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") {
		return "", errSemCredencial
	}
	return strings.TrimSpace(partes[1]), nil
}
