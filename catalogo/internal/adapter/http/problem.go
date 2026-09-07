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
	catSessaoNaoReservavel = "sessao-nao-reservavel"
	catConflito            = "conflito"
	catPoltronasIndisp     = "poltronas-indisponiveis"
	catReservaRecusada     = "reserva-recusada"
	catPoltronaInexistente = "poltrona-inexistente"
	catSessaoSemPoltronas  = "sessao-sem-poltronas"
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
	catSalaNaoEncontrada:   {"Sala não encontrada", http.StatusNotFound},
	catFilmeNaoEncontrado:  {"Filme não encontrado", http.StatusNotFound},
	catSessaoNaoEncontrada: {"Sessão não encontrada", http.StatusNotFound},
	catSessaoNaoReservavel: {"Sessão não aceita reservas", http.StatusUnprocessableEntity},
	catConflito:            {"Conflito com o estado atual", http.StatusConflict},
	catPoltronasIndisp:     {"Poltronas indisponíveis", http.StatusConflict},
	catReservaRecusada:     {"Reserva recusada", http.StatusBadRequest},
	catPoltronaInexistente: {"Poltrona inexistente na sessão", http.StatusUnprocessableEntity},
	catSessaoSemPoltronas:  {"Sessão ainda sem poltronas", http.StatusUnprocessableEntity},
	catEstoqueIndisponivel: {"Serviço temporariamente indisponível", http.StatusServiceUnavailable},
	catRespostaInvalida:    {"Resposta inválida do serviço parceiro", http.StatusBadGateway},
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
	case errors.Is(err, shared.ErrSessaoNaoReservavel):
		EscreverProblem(w, r, catSessaoNaoReservavel, mensagemLimpa(err))
	case errors.Is(err, shared.ErrPoltronasIndisponiveis):
		EscreverProblem(w, r, catPoltronasIndisp, "Uma ou mais poltronas selecionadas não estão disponíveis.")

	// As três recusas que o estoque decide. Antes caíam todas em
	// `estoque-indisponivel`, o que dizia ao cliente para tentar de novo quando
	// o defeito estava na própria solicitação.
	case errors.Is(err, shared.ErrPoltronaInexistente):
		EscreverProblem(w, r, catPoltronaInexistente,
			"Uma ou mais poltronas informadas não existem nesta sessão.")
	case errors.Is(err, shared.ErrSessaoSemPoltronas):
		EscreverProblem(w, r, catSessaoSemPoltronas,
			"Esta sessão ainda não tem poltronas disponíveis para reserva.")
	case errors.Is(err, shared.ErrSolicitacaoRecusadaPeloEstoque):
		EscreverProblem(w, r, catReservaRecusada, mensagemLimpa(err))

	// Defeito do parceiro, e não indisponibilidade: o contrato dele diz que
	// repetir não tem por que dar certo.
	case errors.Is(err, shared.ErrEstoqueComDefeito):
		EscreverProblem(w, r, catRespostaInvalida,
			"O serviço responsável pela reserva falhou. Informe o identificador desta requisição ao suporte.")

	case errors.Is(err, shared.ErrEstoqueIndisponivel):
		EscreverProblem(w, r, catEstoqueIndisponivel, "Serviço temporariamente indisponível. Tente novamente em instantes.")
	case errors.Is(err, shared.ErrRespostaInvalidaDoParceiro):
		EscreverProblem(w, r, catRespostaInvalida, "Não foi possível confirmar a reserva junto ao serviço responsável.")
	default:
		slog.ErrorContext(r.Context(), "erro não previsto", slog.Any("erro", err), slog.String("contexto", contexto))
		EscreverProblem(w, r, catErroInterno, "Erro interno. Consulte o suporte informando o identificador desta requisição.")
	}
}

// O recurso ausente muda o `type` do problema. Quando o erro diz qual recurso
// faltou, é ele quem manda: numa escrita de sessão o ausente pode ser o filme
// ou a sala, e o caminho não denuncia isso. O contexto do handler fica como
// resposta para os erros que não se identificam.
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
		shared.ErrSessaoNaoReservavel.Error() + ": ",
	} {
		if strings.HasPrefix(msg, prefixo) {
			return strings.TrimPrefix(msg, prefixo)
		}
	}
	return msg
}
