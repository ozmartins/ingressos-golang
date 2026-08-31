// Package http expõe as duas operações síncronas do serviço — a listagem dos
// ingressos da pessoa e a validação na portaria — mais as sondas de saúde.
//
// Dois esquemas de credencial, sem hierarquia entre eles: cada rota aceita
// somente o seu. Apresentar o outro é recusa, não promoção (research.md D7).
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/platform/health"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

// API reúne as dependências das rotas.
type API struct {
	Listagem  usecase.ListarIngressos
	Validacao usecase.ValidarIngresso
	Auth      *Autenticador
	Chave     *ChavePortaria
	Prontidao *health.Prontidao
	Log       *slog.Logger
}

// Rotas monta o roteador. São quatro caminhos: net/http basta, e um roteador de
// terceiro não se pagaria aqui (princípio I).
func (a *API) Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ingressos/meus-ingressos", a.listar)
	mux.HandleFunc("POST /api/v1/ingressos/validar", a.validar)
	mux.HandleFunc("GET /health/live", a.vivo)
	mux.HandleFunc("GET /health/ready", a.pronto)
	return mux
}

// --- Cliente ---

type ingressoResposta struct {
	IngressoID string `json:"ingresso_id"`
	ReservaID  string `json:"reserva_id"`
	CodigoQR   string `json:"codigo_qr"`
	Status     string `json:"status"`
	CriadoEm   string `json:"criado_em"`
}

func (a *API) listar(w http.ResponseWriter, r *http.Request) {
	// A rota do cliente não olha X-API-Key: chave de portaria aqui é 401.
	sub, err := a.Auth.Identificar(r)
	if err != nil {
		problema(w, http.StatusUnauthorized, "credencial-invalida",
			"Credencial inválida", "A credencial de pessoa está ausente, expirada ou não confere.")
		return
	}

	lista, err := a.Listagem.Executar(r.Context(), sub, r.URL.Query().Get("status"))
	if errors.Is(err, usecase.ErrStatusDesconhecido) {
		problema(w, http.StatusBadRequest, "filtro-invalido",
			"Filtro de estado não reconhecido",
			"O parâmetro status aceita VALIDO, UTILIZADO ou CANCELADO.")
		return
	}
	if err != nil {
		a.log().Error("falha ao listar ingressos", "erro", err)
		problema(w, http.StatusServiceUnavailable, "indisponivel",
			"Serviço indisponível", "Não foi possível consultar os ingressos agora.")
		return
	}

	corpo := make([]ingressoResposta, 0, len(lista))
	for _, i := range lista {
		corpo = append(corpo, ingressoResposta{
			IngressoID: i.ID, ReservaID: i.ReservaID, CodigoQR: i.CodigoQR,
			Status: string(i.Status), CriadoEm: i.CriadoEm.UTC().Format(time.RFC3339),
		})
	}
	responderJSON(w, http.StatusOK, corpo)
}

// --- Portaria ---

type validarPedido struct {
	CodigoQR *string `json:"codigo_qr"`
}

// vereditoResposta é a forma da ERS — deliberadamente NÃO é RFC 9457. É o
// contrato com o dispositivo de catraca (contracts/erros.md §1).
type vereditoResposta struct {
	Valido      bool   `json:"valido"`
	Mensagem    string `json:"mensagem"`
	IngressoID  string `json:"ingresso_id,omitempty"`
	UtilizadoEm string `json:"utilizado_em,omitempty"`
}

func (a *API) validar(w http.ResponseWriter, r *http.Request) {
	// A recusa por credencial acontece ANTES de qualquer consulta ao ingresso:
	// sem chave válida, não se descobre nem que o código existe (FR-012).
	if err := a.Chave.Autorizar(r); err != nil {
		problema(w, http.StatusUnauthorized, "credencial-invalida",
			"Credencial inválida", "A chave do dispositivo de portaria está ausente ou não confere.")
		return
	}

	var pedido validarPedido
	if err := json.NewDecoder(r.Body).Decode(&pedido); err != nil || pedido.CodigoQR == nil {
		problema(w, http.StatusUnprocessableEntity, "pedido-invalido",
			"Pedido inválido", "O corpo deve conter o campo codigo_qr.")
		return
	}

	res, err := a.Validacao.Executar(r.Context(), *pedido.CodigoQR)
	if err != nil {
		a.log().Error("falha ao validar ingresso", "erro", err)
		problema(w, http.StatusServiceUnavailable, "indisponivel",
			"Serviço indisponível", "Não foi possível validar o ingresso agora.")
		return
	}

	switch res.Veredito {
	case usecase.Autorizada:
		var utilizadoEm string
		if res.Ingresso.UtilizadoEm != nil {
			utilizadoEm = res.Ingresso.UtilizadoEm.UTC().Format(time.RFC3339)
		}
		responderJSON(w, http.StatusOK, vereditoResposta{
			Valido: true, Mensagem: "Entrada autorizada.",
			IngressoID: res.Ingresso.ID, UtilizadoEm: utilizadoEm,
		})
	case usecase.Reuso:
		responderJSON(w, http.StatusConflict, vereditoResposta{
			Valido: false, Mensagem: "Ingresso já utilizado anteriormente.",
		})
	case usecase.NaoValido:
		responderJSON(w, http.StatusConflict, vereditoResposta{
			Valido: false, Mensagem: "Ingresso não está válido.",
		})
	default:
		// Malformado, assinatura inválida e inexistente: MESMA resposta, para
		// que nenhuma delas seja distinguível das outras (FR-010).
		responderJSON(w, http.StatusNotFound, vereditoResposta{
			Valido: false, Mensagem: "Ingresso não encontrado.",
		})
	}
}

// --- Saúde ---

func (a *API) vivo(w http.ResponseWriter, _ *http.Request) {
	responderJSON(w, http.StatusOK, map[string]string{"status": "vivo"})
}

func (a *API) pronto(w http.ResponseWriter, r *http.Request) {
	if nome, err := a.Prontidao.Verificar(r.Context()); err != nil {
		a.log().Warn("dependência indisponível", "dependencia", nome, "erro", err)
		problema(w, http.StatusServiceUnavailable, "indisponivel",
			"Dependência indisponível", nome+" não respondeu.")
		return
	}
	responderJSON(w, http.StatusOK, map[string]string{"status": "pronto"})
}

// --- Auxiliares ---

func (a *API) log() *slog.Logger {
	if a.Log != nil {
		return a.Log
	}
	return slog.Default()
}

func responderJSON(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
}

type problemaResposta struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// problema responde no formato RFC 9457, usado para erro de protocolo. A recusa
// da portaria não passa por aqui: ela tem forma própria (contracts/erros.md §1).
func problema(w http.ResponseWriter, status int, tipo, titulo, detalhe string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemaResposta{
		Type:   "https://ingressos.example/problemas/" + tipo,
		Title:  titulo,
		Status: status,
		Detail: detalhe,
	})
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
