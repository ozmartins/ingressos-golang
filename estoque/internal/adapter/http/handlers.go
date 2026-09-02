package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/adapter/http/openapi"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

type CasoDeUsoBloqueio interface {
	Executar(ctx context.Context, sessaoID, usuarioID string, rotulos []string) (usecase.ResultadoBloqueio, error)
}

type CasoDeUsoMapa interface {
	Executar(ctx context.Context, sessaoID string) ([]poltrona.Poltrona, error)
}

type API struct {
	Bloqueio CasoDeUsoBloqueio
	Mapa     CasoDeUsoMapa
	Auth     *Autenticador
	Limite   int
}

type solicitacaoBloqueio struct {
	PoltronasIDs []string `json:"poltronas_ids"`
}

type respostaBloqueio struct {
	ReservaID string `json:"reserva_id"`
	ExpiraEm  string `json:"expira_em"`
	Mensagem  string `json:"mensagem"`
}

type poltronaResposta struct {
	Rotulo  string `json:"rotulo"`
	Fileira string `json:"fileira"`
	Numero  int    `json:"numero"`
	Tipo    string `json:"tipo"`
	Status  string `json:"status"`
}

type respostaMapa struct {
	SessaoID  string             `json:"sessao_id"`
	Poltronas []poltronaResposta `json:"poltronas"`
}

func (a *API) Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/sessoes/{sessao_id}/bloqueios", a.bloquear)
	mux.HandleFunc("GET /api/v1/sessoes/{sessao_id}/poltronas", a.consultarMapa)

	mux.Handle("GET /openapi.yaml", openapi.HandlerEspecificacao())
	mux.Handle("GET /docs", openapi.HandlerUI("/openapi.yaml"))
	mux.Handle("GET /docs/", openapi.HandlerUI("/openapi.yaml"))
	return mux
}

func (a *API) bloquear(w http.ResponseWriter, r *http.Request) {
	usuarioID, err := a.Auth.Identificar(r)
	if err != nil {
		escreverProblem(w, r, http.StatusUnauthorized, tipoNaoAutenticado,
			"Não autenticado", "credencial ausente ou inválida", nil)
		return
	}

	var corpo solicitacaoBloqueio
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		escreverProblem(w, r, http.StatusBadRequest, tipoCorpoInvalido,
			"Corpo inválido", "o corpo da requisição não é um JSON válido", nil)
		return
	}

	resultado, err := a.Bloqueio.Executar(r.Context(), r.PathValue("sessao_id"), usuarioID, corpo.PoltronasIDs)
	if err != nil {
		responderErroDeDominio(w, r, err, a.Limite)
		return
	}

	if !resultado.Concedido {
		escreverProblem(w, r, http.StatusConflict, tipoPoltronasIndisponivel,
			"Poltronas indisponíveis", resultado.Mensagem, nil)
		return
	}

	responderJSON(w, http.StatusCreated, respostaBloqueio{
		ReservaID: resultado.Reserva.ID,
		ExpiraEm:  resultado.Reserva.ExpiraEm.UTC().Format(time.RFC3339),
		Mensagem:  resultado.Mensagem,
	})
}

func (a *API) consultarMapa(w http.ResponseWriter, r *http.Request) {
	if _, err := a.Auth.Identificar(r); err != nil {
		escreverProblem(w, r, http.StatusUnauthorized, tipoNaoAutenticado,
			"Não autenticado", "credencial ausente ou inválida", nil)
		return
	}

	sessaoID := r.PathValue("sessao_id")
	mapa, err := a.Mapa.Executar(r.Context(), sessaoID)
	if err != nil {
		responderErroDeDominio(w, r, err, a.Limite)
		return
	}

	resposta := respostaMapa{SessaoID: sessaoID, Poltronas: make([]poltronaResposta, 0, len(mapa))}
	for _, p := range mapa {
		resposta.Poltronas = append(resposta.Poltronas, poltronaResposta{
			Rotulo:  p.Rotulo,
			Fileira: p.Fileira,
			Numero:  p.Numero,
			Tipo:    string(p.Tipo),
			Status:  string(p.Status),
		})
	}
	responderJSON(w, http.StatusOK, resposta)
}

func responderJSON(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(corpo); err != nil {
		slog.Debug("falha ao escrever a resposta", slog.Any("erro", err))
	}
}
