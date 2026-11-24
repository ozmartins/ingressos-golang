package http

import (
	"encoding/json"
	"errors"

	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/adapter/http/openapi"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/health"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/obs"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

const prefixoTipo = "https://cinema.example/errors/"

const (
	CodReservaIDInvalido  = "reserva-id-invalido"
	CodCredencialInvalida = "credencial-invalida"
	CodAcessoNegado       = "acesso-negado"
	CodNaoEncontrado      = "pagamento-nao-encontrado"
	CodIndisponivel       = "servico-indisponivel"
	CodCorpoInvalido      = "corpo-invalido"
	CodFormaDesconhecida  = "forma-pagamento-desconhecida"
	CodFormaJaEscolhida   = "forma-pagamento-ja-escolhida"
	CodReservaExpirada    = "reserva-expirada"
)

var titulos = map[string]string{
	CodReservaIDInvalido:  "Identificador de reserva inválido",
	CodCredencialInvalida: "Credencial ausente ou inválida",
	CodAcessoNegado:       "Acesso negado",
	CodNaoEncontrado:      "Pagamento não encontrado",
	CodIndisponivel:       "Serviço temporariamente indisponível",
	CodCorpoInvalido:      "Corpo da requisição inválido",
	CodFormaDesconhecida:  "Forma de pagamento desconhecida",
	CodFormaJaEscolhida:   "Forma de pagamento já escolhida",
	CodReservaExpirada:    "Reserva expirada",
}

type problema struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type pagamentoResposta struct {
	TransacaoID    string      `json:"transacao_id"`
	ReservaID      string      `json:"reserva_id"`
	Status         string      `json:"status"`
	ValorTotal     json.Number `json:"valor_total"`
	FormaPagamento string      `json:"forma_pagamento,omitempty"`
	ExpiraEm       string      `json:"expira_em"`
	CriadoEm       string      `json:"criado_em"`
}

type escolhaEntrada struct {
	FormaPagamento string `json:"forma_pagamento"`
}

type API struct {
	Consulta  usecase.ConsultarPagamento
	Escolha   usecase.EscolherForma
	Auth      *Autenticador
	Prontidao *health.Prontidao
}

func (a *API) Rotas() http.Handler {
	r := chi.NewRouter()
	r.Use(obs.MiddlewareHTTP)
	get := func(caminho string, h http.HandlerFunc) {
		r.Get(caminho, h)
		r.Head(caminho, h)
	}
	get("/api/v1/pagamentos/reserva/{reserva_id}", a.consultar)
	r.Post("/api/v1/pagamentos/reserva/{reserva_id}", a.escolherForma)
	get("/api/v1/health/live", a.vivo)
	get("/api/v1/health/ready", a.pronto)

	ui := openapi.HandlerUI("/openapi.yaml")
	get("/openapi.yaml", openapi.HandlerEspecificacao())
	get("/docs", ui)
	get("/docs/*", ui)

	r.NotFound(http.NotFound)
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Allow", strings.Join(metodosPermitidos(r, req.URL.Path), ", "))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})
	return r
}

func metodosPermitidos(r chi.Routes, caminho string) []string {
	var ms []string
	for _, m := range []string{
		http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodOptions,
		http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace,
	} {
		if r.Match(chi.NewRouteContext(), m, caminho) {
			ms = append(ms, m)
		}
	}
	return ms
}

func (a *API) consultar(w http.ResponseWriter, r *http.Request) {
	sub, err := a.Auth.Exigir(r, PermissaoCliente)
	if err != nil {
		recusarCredencial(w, err)
		return
	}

	reservaID := chi.URLParam(r, "reserva_id")
	if _, err := uuid.Parse(reservaID); err != nil {
		responderErro(w, http.StatusBadRequest, CodReservaIDInvalido, "reserva_id deve ser um UUID")
		return
	}

	t, err := a.Consulta.Executar(r.Context(), reservaID, sub)
	switch {
	case usecase.NaoEncontrada(err):

		responderErro(w, http.StatusNotFound, CodNaoEncontrado, "não há pagamento para essa reserva")
	case errors.Is(err, usecase.ErrDependenciaIndisponivel):

		responderErro(w, http.StatusServiceUnavailable, CodIndisponivel, "serviço indisponível no momento")
	case err != nil:

		responderErro(w, http.StatusServiceUnavailable, CodIndisponivel, "serviço indisponível no momento")
	default:
		responderJSON(w, http.StatusOK, paraResposta(t))
	}
}

func (a *API) escolherForma(w http.ResponseWriter, r *http.Request) {
	sub, err := a.Auth.Exigir(r, PermissaoCliente)
	if err != nil {
		recusarCredencial(w, err)
		return
	}

	reservaID := chi.URLParam(r, "reserva_id")
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

		responderErro(w, http.StatusNotFound, CodNaoEncontrado, "não há pagamento para essa reserva")

	case errors.Is(err, transacao.ErrFormaDesconhecida):
		responderErro(w, http.StatusBadRequest, CodFormaDesconhecida,
			"forma_pagamento deve ser PIX ou CARTAO_CREDITO")

	case errors.Is(err, transacao.ErrFormaJaEscolhida), errors.Is(err, usecase.ErrJaFinalizada):
		responderErro(w, http.StatusConflict, CodFormaJaEscolhida,
			"a forma de pagamento desta reserva já foi escolhida")

	case errors.Is(err, transacao.ErrTransicaoInvalida):
		responderErro(w, http.StatusConflict, CodFormaJaEscolhida,
			"esta reserva não aceita mais escolha de forma de pagamento")

	case errors.Is(err, transacao.ErrReservaExpirada):
		responderErro(w, http.StatusConflict, CodReservaExpirada,
			"o prazo da reserva venceu e ela não pode mais ser paga")

	case errors.Is(err, usecase.ErrDependenciaIndisponivel):

		responderErro(w, http.StatusServiceUnavailable, CodIndisponivel, "serviço indisponível no momento")

	case err != nil:

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
		responderJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "indisponível", "dependencia": nome,
		})
		return
	}
	responderJSON(w, http.StatusOK, map[string]string{"status": "pronto"})
}

func responderJSON(w http.ResponseWriter, status int, corpo any) {
	escrever(w, "application/json", status, corpo)
}

// recusarCredencial responde 403 se o token é válido mas sem permissão, e 401
// em qualquer outro caso (ausente, inválido, expirado).
func recusarCredencial(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrSemPermissao) {
		responderErro(w, http.StatusForbidden, CodAcessoNegado, "a credencial é válida, mas não concede permissão para esta operação")
		return
	}
	responderErro(w, http.StatusUnauthorized, CodCredencialInvalida, "credencial ausente ou inválida")
}

func responderErro(w http.ResponseWriter, status int, codigo, detalhe string) {
	escrever(w, "application/problem+json", status, problema{
		Type:   prefixoTipo + codigo,
		Title:  titulos[codigo],
		Status: status,
		Detail: detalhe,
	})
}

func escrever(w http.ResponseWriter, tipo string, status int, corpo any) {
	w.Header().Set("Content-Type", tipo)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
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
