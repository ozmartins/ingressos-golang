package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/trace"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const prefixoTipo = "https://cinema.example/errors/"

const (
	tipoNaoAutenticado        = "nao-autenticado"
	tipoCorpoInvalido         = "corpo-invalido"
	tipoSolicitacaoInvalida   = "solicitacao-invalida"
	tipoLimiteExcedido        = "limite-poltronas-excedido"
	tipoSessaoNaoProvisionada = "sessao-nao-provisionada"
	tipoPoltronaInexistente   = "poltrona-inexistente"
	tipoSessaoDesconhecida    = "sessao-desconhecida"
	tipoPoltronasIndisponivel = "poltronas-indisponiveis"
	tipoDependencia           = "dependencia-indisponivel"
	tipoInterno               = "erro-interno"
)

type problem struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Status   int               `json:"status"`
	Detail   string            `json:"detail,omitempty"`
	Instance string            `json:"instance,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func escreverProblem(w http.ResponseWriter, r *http.Request, status int, tipo, title, detail string, metadata map[string]string) {
	corpo := problem{
		Type:     prefixoTipo + tipo,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Metadata: metadata,
	}
	if sc := trace.SpanContextFromContext(r.Context()); sc.HasTraceID() {
		corpo.Instance = "urn:trace:" + sc.TraceID().String()
	}

	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(corpo); err != nil {
		slog.Debug("falha ao escrever a resposta de erro", slog.Any("erro", err))
	}
}

func responderErroDeDominio(w http.ResponseWriter, r *http.Request, err error, limitePoltronas int) {
	switch {
	case errors.Is(err, shared.ErrLimiteExcedido):
		escreverProblem(w, r, http.StatusBadRequest, tipoLimiteExcedido,
			"Limite de poltronas excedido",
			"quantidade de poltronas acima do limite por bloqueio",
			map[string]string{"limite": strconv.Itoa(limitePoltronas)})

	case errors.Is(err, shared.ErrSolicitacaoInvalida):
		escreverProblem(w, r, http.StatusBadRequest, tipoSolicitacaoInvalida,
			"Solicitação inválida", err.Error(), nil)

	case errors.Is(err, shared.ErrSessaoNaoProvisionada):
		escreverProblem(w, r, http.StatusUnprocessableEntity, tipoSessaoNaoProvisionada,
			"Sessão não provisionada",
			"sessão sem matriz de poltronas provisionada", nil)

	case errors.Is(err, shared.ErrPoltronaInexistente):
		escreverProblem(w, r, http.StatusUnprocessableEntity, tipoPoltronaInexistente,
			"Poltrona inexistente",
			"poltrona inexistente na sessão informada", nil)

	case errors.Is(err, shared.ErrSessaoDesconhecida):
		escreverProblem(w, r, http.StatusNotFound, tipoSessaoDesconhecida,
			"Sessão desconhecida", "sessão desconhecida", nil)

	case errors.Is(err, shared.ErrDependenciaIndisponivel):
		escreverProblem(w, r, http.StatusServiceUnavailable, tipoDependencia,
			"Serviço temporariamente indisponível",
			"serviço temporariamente indisponível", nil)

	default:
		escreverProblem(w, r, http.StatusInternalServerError, tipoInterno,
			"Erro interno", "erro interno", nil)
	}
}
