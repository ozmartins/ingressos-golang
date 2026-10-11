package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

const BaseURIErros = "https://cinema.example/errors/"

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

const (
	catParametroInvalido   = "parametro-invalido"
	catCorpoInvalido       = "corpo-invalido"
	catNaoAutenticado      = "nao-autenticado"
	catCinemaNaoEncontrado = "cinema-nao-encontrado"
	catSalaNaoEncontrada   = "sala-nao-encontrada"
	catFilmeNaoEncontrado  = "filme-nao-encontrado"
	catSessaoNaoEncontrada = "sessao-nao-encontrada"
	catConflito            = "conflito"
	catBancoIndisponivel   = "banco-indisponivel"
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
	catSalaNaoEncontrada:   {"Sala não encontrada", http.StatusNotFound},
	catFilmeNaoEncontrado:  {"Filme não encontrado", http.StatusNotFound},
	catSessaoNaoEncontrada: {"Sessão não encontrada", http.StatusNotFound},
	catConflito:            {"Conflito com o estado atual", http.StatusConflict},
	catBancoIndisponivel:   {"Serviço temporariamente indisponível", http.StatusServiceUnavailable},
	catErroInterno:         {"Erro interno", http.StatusInternalServerError},
}

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

func EscreverErroDeDominio(w http.ResponseWriter, r *http.Request, err error, contexto string) {
	switch {
	case errors.Is(err, shared.ErrValidacao):
		EscreverProblem(w, r, catParametroInvalido, mensagemLimpa(err))
	case errors.Is(err, shared.ErrConflito):
		EscreverProblem(w, r, catConflito, mensagemLimpa(err))
	case errors.Is(err, shared.ErrNaoEncontrado):
		EscreverProblem(w, r, categoriaNaoEncontrado(err, contexto), mensagemLimpa(err))
	case errors.Is(err, shared.ErrBancoIndisponivel):
		slog.ErrorContext(r.Context(), "banco de dados indisponível", slog.Any("erro", err), slog.String("contexto", contexto))
		EscreverProblem(w, r, catBancoIndisponivel, "Serviço temporariamente indisponível. Tente novamente em instantes.")
	default:
		slog.ErrorContext(r.Context(), "erro não previsto", slog.Any("erro", err), slog.String("contexto", contexto))
		EscreverProblem(w, r, catErroInterno, "Erro interno. Consulte o suporte informando o identificador desta requisição.")
	}
}

func categoriaNaoEncontrado(err error, contexto string) string {
	var ausente shared.RecursoAusente
	if errors.As(err, &ausente) {
		contexto = ausente.Recurso
	}
	switch contexto {
	case "sessao":
		return catSessaoNaoEncontrada
	case "filme":
		return catFilmeNaoEncontrado
	case "sala":
		return catSalaNaoEncontrada
	default:
		return catCinemaNaoEncontrado
	}
}

func mensagemLimpa(err error) string {
	msg := err.Error()
	for _, prefixo := range []string{
		shared.ErrValidacao.Error() + ": ",
		shared.ErrNaoEncontrado.Error() + ": ",
		shared.ErrConflito.Error() + ": ",
	} {
		if strings.HasPrefix(msg, prefixo) {
			return strings.TrimPrefix(msg, prefixo)
		}
	}
	return msg
}
