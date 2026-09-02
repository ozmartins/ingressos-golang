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

// prefixoTipo é o espaço de nomes dos `type` de erro, como no Servico-Catalogo.
const prefixoTipo = "https://cinema.example/errors/"

// Sufixos de `type`. São o contrato estável que o cliente inspeciona; `title` e
// `detail` são texto humano e podem mudar de redação sem quebrar ninguém.
//
// Cada um corresponde a uma razão já publicada no contrato gRPC
// (contracts/erros.md): a superfície HTTP não inventa categorias de erro novas,
// só as reapresenta com o status equivalente em HTTP.
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

// problem é o corpo de erro da RFC 9457.
type problem struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Status   int               `json:"status"`
	Detail   string            `json:"detail,omitempty"`
	Instance string            `json:"instance,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// escreverProblem responde no formato problem+json.
//
// `instance` carrega o trace_id da requisição para que quem reporta um problema
// traga consigo a chave que localiza o rastro no log.
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

// responderErroDeDominio traduz um erro do núcleo na resposta HTTP equivalente.
//
// Os status espelham o mapa de contracts/erros.md, trocando o código gRPC pelo
// seu correspondente em HTTP: InvalidArgument→400, FailedPrecondition→422,
// NotFound→404, Unavailable→503, Internal→500. Nenhum detalhe interno atravessa
// esta função — mensagem de driver, SQL ou endereço de dependência ficam no log,
// correlacionados pelo trace_id (princípio IV).
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
		// O cliente só precisa saber que não foi possível decidir agora.
		escreverProblem(w, r, http.StatusServiceUnavailable, tipoDependencia,
			"Serviço temporariamente indisponível",
			"serviço temporariamente indisponível", nil)

	default:
		escreverProblem(w, r, http.StatusInternalServerError, tipoInterno,
			"Erro interno", "erro interno", nil)
	}
}
