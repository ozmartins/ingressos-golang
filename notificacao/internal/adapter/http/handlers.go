package http

import (
	"encoding/json"
	"errors"

	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/http/openapi"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/health"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/obs"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/ratelimit"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

type API struct {
	Listagem  usecase.ListarIngressos
	Validacao usecase.ValidarIngresso
	Auth      *Autenticador
	Prontidao *health.Prontidao
	// TaxaValidacao limita /validar por IP de origem, contra abuso do
	// endpoint de portaria (nil = sem limite).
	TaxaValidacao *ratelimit.Limitador
}

func (a *API) Rotas() http.Handler {
	r := chi.NewRouter()
	r.Use(obs.MiddlewareHTTP, middleware.GetHead)
	r.MethodNotAllowed(metodoNaoPermitido(r))

	r.Get("/api/v1/ingressos/meus-ingressos", a.listar)
	r.Post("/api/v1/ingressos/validar", a.validar)
	r.Get("/health/live", a.vivo)
	r.Get("/health/ready", a.pronto)

	r.Get("/openapi.yaml", openapi.HandlerEspecificacao())
	ui := openapi.HandlerUI("/openapi.yaml")
	r.Get("/docs", ui)
	r.Get("/docs/*", ui)
	return r
}

func metodoNaoPermitido(r *chi.Mux) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var permitidos []string
		for _, m := range []string{http.MethodGet, http.MethodPost} {
			if !r.Match(chi.NewRouteContext(), m, req.URL.Path) {
				continue
			}
			permitidos = append(permitidos, m)
			if m == http.MethodGet {
				permitidos = append(permitidos, http.MethodHead)
			}
		}
		if len(permitidos) > 0 {
			w.Header().Set("Allow", strings.Join(permitidos, ", "))
		}
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

type ingressoResposta struct {
	IngressoID string `json:"ingresso_id"`
	ReservaID  string `json:"reserva_id"`
	CodigoQR   string `json:"codigo_qr"`
	Status     string `json:"status"`
	CriadoEm   string `json:"criado_em"`
}

func (a *API) listar(w http.ResponseWriter, r *http.Request) {
	sub, err := a.Auth.Exigir(r, PermissaoCliente)
	if errors.Is(err, ErrSemPermissao) {
		problema(w, http.StatusForbidden, "acesso-negado",
			"Acesso negado", "A credencial é válida, mas não concede permissão para ver ingressos.")
		return
	}
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
	if errors.Is(err, usecase.ErrDependenciaIndisponivel) {
		problema(w, http.StatusServiceUnavailable, "indisponivel",
			"Serviço indisponível", "Não foi possível consultar os ingressos agora.")
		return
	}
	if err != nil {
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

type validarPedido struct {
	CodigoQR *string `json:"codigo_qr"`
}

type vereditoResposta struct {
	Valido      bool   `json:"valido"`
	Mensagem    string `json:"mensagem"`
	IngressoID  string `json:"ingresso_id,omitempty"`
	UtilizadoEm string `json:"utilizado_em,omitempty"`
}

func (a *API) validar(w http.ResponseWriter, r *http.Request) {
	if !a.TaxaValidacao.Permitir(ratelimit.IPDe(r)) {
		w.Header().Set("Retry-After", "1")
		problema(w, http.StatusTooManyRequests, "muitas-requisicoes",
			"Muitas requisições", "Limite de validações por segundo excedido; tente novamente em instantes.")
		return
	}
	if _, err := a.Auth.Exigir(r, PermissaoPortaria); err != nil {
		if errors.Is(err, ErrSemPermissao) {
			problema(w, http.StatusForbidden, "acesso-negado",
				"Acesso negado", "A credencial é válida, mas não é de um dispositivo de portaria.")
			return
		}
		problema(w, http.StatusUnauthorized, "credencial-invalida",
			"Credencial inválida", "O token do dispositivo de portaria está ausente, expirado ou não confere.")
		return
	}

	var pedido validarPedido
	if err := json.NewDecoder(r.Body).Decode(&pedido); err != nil || pedido.CodigoQR == nil {
		problema(w, http.StatusUnprocessableEntity, "pedido-invalido",
			"Pedido inválido", "O corpo deve conter o campo codigo_qr.")
		return
	}

	res, err := a.Validacao.Executar(r.Context(), *pedido.CodigoQR)
	if errors.Is(err, usecase.ErrDependenciaIndisponivel) {
		problema(w, http.StatusServiceUnavailable, "indisponivel",
			"Serviço indisponível", "Não foi possível validar o ingresso agora.")
		return
	}
	if err != nil {
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
		responderJSON(w, http.StatusNotFound, vereditoResposta{
			Valido: false, Mensagem: "Ingresso não encontrado.",
		})
	}
}

func (a *API) vivo(w http.ResponseWriter, _ *http.Request) {
	responderJSON(w, http.StatusOK, map[string]string{"status": "vivo"})
}

func (a *API) pronto(w http.ResponseWriter, r *http.Request) {
	if nome, err := a.Prontidao.Verificar(r.Context()); err != nil {
		problema(w, http.StatusServiceUnavailable, "indisponivel",
			"Dependência indisponível", nome+" não respondeu.")
		return
	}
	responderJSON(w, http.StatusOK, map[string]string{"status": "pronto"})
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

func problema(w http.ResponseWriter, status int, tipo, titulo, detalhe string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemaResposta{
		Type:   "https://cinema.example/errors/" + tipo,
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
