package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/adapter/http/openapi"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/health"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

const (
	CodReservaIDInvalido  = "RESERVA_ID_INVALIDO"
	CodCredencialInvalida = "CREDENCIAL_INVALIDA"
	CodNaoEncontrado      = "PAGAMENTO_NAO_ENCONTRADO"
	CodIndisponivel       = "SERVICO_INDISPONIVEL"
	CodCorpoInvalido      = "CORPO_INVALIDO"
	CodFormaDesconhecida  = "FORMA_PAGAMENTO_DESCONHECIDA"
	CodFormaJaEscolhida   = "FORMA_PAGAMENTO_JA_ESCOLHIDA"
	CodReservaExpirada    = "RESERVA_EXPIRADA"
)

type erroResposta struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

type pagamentoResposta struct {
	TransacaoID string      `json:"transacao_id"`
	ReservaID   string      `json:"reserva_id"`
	Status      string      `json:"status"`
	ValorTotal  json.Number `json:"valor_total"`
	// Vazio enquanto a escolha não acontece: o valor é conhecido desde a
	// reserva, a forma não.
	FormaPagamento string `json:"forma_pagamento,omitempty"`
	ExpiraEm       string `json:"expira_em"`
	CriadoEm       string `json:"criado_em"`
}

type escolhaEntrada struct {
	FormaPagamento string `json:"forma_pagamento"`
}

type API struct {
	Consulta  usecase.ConsultarPagamento
	Escolha   usecase.EscolherForma
	Auth      *Autenticador
	Prontidao *health.Prontidao
	Log       *slog.Logger
}

func (a *API) Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/pagamentos/reserva/{reserva_id}", a.consultar)
	mux.HandleFunc("POST /api/v1/pagamentos/reserva/{reserva_id}", a.escolherForma)
	mux.HandleFunc("GET /api/v1/health/live", a.vivo)
	mux.HandleFunc("GET /api/v1/health/ready", a.pronto)

	mux.Handle("GET /openapi.yaml", openapi.HandlerEspecificacao())
	mux.Handle("GET /docs", openapi.HandlerUI("/openapi.yaml"))
	mux.Handle("GET /docs/", openapi.HandlerUI("/openapi.yaml"))
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
		a.Log.Info("consulta sem resultado visível", "reserva_id", reservaID, "sub", sub)
		responderErro(w, http.StatusNotFound, CodNaoEncontrado, "não há pagamento para essa reserva")
	case err != nil:
		a.Log.Error("falha ao consultar pagamento", "reserva_id", reservaID, "erro", err)
		responderErro(w, http.StatusServiceUnavailable, CodIndisponivel, "serviço indisponível no momento")
	default:
		responderJSON(w, http.StatusOK, paraResposta(t))
	}
}

// A escolha da forma de pagamento. Responde 202: o pedido foi aceito e a
// cobrança acontece fora desta requisição — quem paga não espera o adquirente.
func (a *API) escolherForma(w http.ResponseWriter, r *http.Request) {
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

	var corpo escolhaEntrada
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, CodCorpoInvalido, "o corpo da requisição não é um JSON válido")
		return
	}

	t, err := a.Escolha.Executar(r.Context(), reservaID, sub, transacao.FormaPagamento(corpo.FormaPagamento))
	switch {
	case usecase.NaoEncontrada(err):
		a.Log.Info("escolha sem pagamento visível", "reserva_id", reservaID, "sub", sub)
		responderErro(w, http.StatusNotFound, CodNaoEncontrado, "não há pagamento para essa reserva")

	case errors.Is(err, transacao.ErrFormaDesconhecida):
		responderErro(w, http.StatusBadRequest, CodFormaDesconhecida,
			"forma_pagamento deve ser PIX ou CARTAO_CREDITO")

	// A corrida entre duas escolhas simultâneas cai aqui pelo mesmo caminho da
	// segunda escolha deliberada: em ambos os casos a forma já está definida.
	case errors.Is(err, transacao.ErrFormaJaEscolhida), errors.Is(err, usecase.ErrJaFinalizada):
		responderErro(w, http.StatusConflict, CodFormaJaEscolhida,
			"a forma de pagamento desta reserva já foi escolhida")

	case errors.Is(err, transacao.ErrTransicaoInvalida):
		responderErro(w, http.StatusConflict, CodFormaJaEscolhida,
			"esta reserva não aceita mais escolha de forma de pagamento")

	// O prazo venceu antes da escolha. O cancelamento é da varredura; aqui só
	// se informa que não há mais o que pagar.
	case errors.Is(err, transacao.ErrReservaExpirada):
		responderErro(w, http.StatusConflict, CodReservaExpirada,
			"o prazo da reserva venceu e ela não pode mais ser paga")

	case err != nil:
		a.Log.Error("falha ao registrar a forma de pagamento", "reserva_id", reservaID, "erro", err)
		responderErro(w, http.StatusServiceUnavailable, CodIndisponivel, "serviço indisponível no momento")

	default:
		responderJSON(w, http.StatusAccepted, paraResposta(t))
	}
}

func paraResposta(t transacao.Transacao) pagamentoResposta {
	return pagamentoResposta{
		TransacaoID:    t.ID,
		ReservaID:      t.ReservaID,
		Status:         string(t.Status),
		ValorTotal:     json.Number(t.ValorTotal),
		FormaPagamento: string(t.FormaPagamento),
		ExpiraEm:       t.ExpiraEm.UTC().Format("2006-01-02T15:04:05Z07:00"),
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
