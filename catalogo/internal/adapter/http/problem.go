// Package http é o adaptador de entrada: traduz requisições em casos de uso e
// resultados em respostas. Nenhuma regra de negócio mora aqui.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// BaseURIErros prefixa os identificadores de categoria. O type é o contrato
// estável que o cliente inspeciona; title e detail são texto humano e podem
// mudar de redação (constituição, princípio IV).
const BaseURIErros = "https://cinema.example/errors/"

// Problem é a representação RFC 9457.
type Problem struct {
	Type     string      `json:"type"`
	Title    string      `json:"title"`
	Status   int         `json:"status"`
	Detail   string      `json:"detail,omitempty"`
	Instance string      `json:"instance,omitempty"`
	Errors   []CampoErro `json:"errors,omitempty"`
}

type CampoErro struct {
	Campo    string `json:"campo"`
	Mensagem string `json:"mensagem"`
}

// Categorias de erro do contrato (contracts/errors.md).
const (
	catParametroInvalido   = "parametro-invalido"
	catCorpoInvalido       = "corpo-invalido"
	catNaoAutenticado      = "nao-autenticado"
	catCinemaNaoEncontrado = "cinema-nao-encontrado"
	catSessaoNaoEncontrada = "sessao-nao-encontrada"
	catSessaoNaoReservavel = "sessao-nao-reservavel"
	catPoltronasIndisp     = "poltronas-indisponiveis"
	catEstoqueIndisponivel = "estoque-indisponivel"
	catRespostaInvalida    = "resposta-invalida-do-parceiro"
	catErroInterno         = "erro-interno"
)

type descricaoCategoria struct {
	titulo string
	status int
}

var categorias = map[string]descricaoCategoria{
	catParametroInvalido:   {"Parâmetro inválido", http.StatusBadRequest},
	catCorpoInvalido:       {"Corpo da requisição inválido", http.StatusBadRequest},
	catNaoAutenticado:      {"Não autenticado", http.StatusUnauthorized},
	catCinemaNaoEncontrado: {"Cinema não encontrado", http.StatusNotFound},
	catSessaoNaoEncontrada: {"Sessão não encontrada", http.StatusNotFound},
	catSessaoNaoReservavel: {"Sessão não aceita reservas", http.StatusUnprocessableEntity},
	catPoltronasIndisp:     {"Poltronas indisponíveis", http.StatusConflict},
	catEstoqueIndisponivel: {"Serviço temporariamente indisponível", http.StatusServiceUnavailable},
	catRespostaInvalida:    {"Resposta inválida do serviço parceiro", http.StatusBadGateway},
	catErroInterno:         {"Erro interno", http.StatusInternalServerError},
}

// EscreverProblem serializa a resposta de erro na categoria informada.
func EscreverProblem(w http.ResponseWriter, r *http.Request, categoria, detail string, campos ...CampoErro) {
	d, ok := categorias[categoria]
	if !ok {
		d = categorias[catErroInterno]
		categoria = catErroInterno
	}
	p := Problem{
		Type:     BaseURIErros + categoria,
		Title:    d.titulo,
		Status:   d.status,
		Detail:   detail,
		Instance: instanciaDoContexto(r),
		Errors:   campos,
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(d.status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		slog.ErrorContext(r.Context(), "falha ao serializar problem+json", slog.Any("erro", err))
	}
}

// EscreverErroDeDominio traduz um erro sentinela para a categoria correspondente.
//
// Erros não previstos viram 500 com detail genérico: detalhe interno vai para o
// log, nunca para a resposta (constituição, princípio IV).
func EscreverErroDeDominio(w http.ResponseWriter, r *http.Request, err error, contexto string) {
	switch {
	case errors.Is(err, shared.ErrValidacao):
		EscreverProblem(w, r, catParametroInvalido, mensagemLimpa(err))
	case errors.Is(err, shared.ErrNaoEncontrado):
		categoria := catCinemaNaoEncontrado
		if contexto == "sessao" {
			categoria = catSessaoNaoEncontrada
		}
		EscreverProblem(w, r, categoria, mensagemLimpa(err))
	case errors.Is(err, shared.ErrSessaoNaoReservavel):
		EscreverProblem(w, r, catSessaoNaoReservavel, mensagemLimpa(err))
	case errors.Is(err, shared.ErrPoltronasIndisponiveis):
		EscreverProblem(w, r, catPoltronasIndisp, "Uma ou mais poltronas selecionadas não estão disponíveis.")
	case errors.Is(err, shared.ErrEstoqueIndisponivel):
		// Timeout e recusa rápida são indistinguíveis aqui, por exigência da
		// especificação. A distinção existe apenas nas métricas e nos logs.
		EscreverProblem(w, r, catEstoqueIndisponivel, "Serviço temporariamente indisponível. Tente novamente em instantes.")
	case errors.Is(err, shared.ErrRespostaInvalidaDoParceiro):
		EscreverProblem(w, r, catRespostaInvalida, "Não foi possível confirmar a reserva junto ao serviço responsável.")
	default:
		slog.ErrorContext(r.Context(), "erro não previsto", slog.Any("erro", err), slog.String("contexto", contexto))
		EscreverProblem(w, r, catErroInterno, "Erro interno. Consulte o suporte informando o identificador desta requisição.")
	}
}

// mensagemLimpa devolve a mensagem do erro sentinela sem o prefixo do sentinela,
// que já está codificado no type.
func mensagemLimpa(err error) string {
	msg := err.Error()
	for _, prefixo := range []string{
		shared.ErrValidacao.Error() + ": ",
		shared.ErrNaoEncontrado.Error() + ": ",
		shared.ErrSessaoNaoReservavel.Error() + ": ",
	} {
		if strings.HasPrefix(msg, prefixo) {
			return strings.TrimPrefix(msg, prefixo)
		}
	}
	return msg
}
