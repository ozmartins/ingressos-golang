// Package http expõe em REST as mesmas operações que o serviço oferece em
// gRPC: bloquear poltronas de uma sessão e consultar o mapa de poltronas.
//
// As duas superfícies chamam os mesmos casos de uso e compartilham, portanto,
// as mesmas regras — este pacote traduz transporte, não decide nada. O que
// muda entre elas é só a identidade do chamador: o gRPC é máquina-a-máquina,
// autenticado por mTLS, e recebe o `usuario_id` no corpo porque quem chama já
// validou o token; aqui o chamador é o cliente final e a identidade vem da
// claim `sub` de um JWT do Keycloak.
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

// CasoDeUsoBloqueio é a porta de entrada do bloqueio, do ponto de vista deste
// adaptador. Declarada aqui, e não importada do adaptador gRPC, para que uma
// superfície não passe a depender da outra.
type CasoDeUsoBloqueio interface {
	Executar(ctx context.Context, sessaoID, usuarioID string, rotulos []string) (usecase.ResultadoBloqueio, error)
}

// CasoDeUsoMapa é a porta de leitura do mapa de poltronas.
type CasoDeUsoMapa interface {
	Executar(ctx context.Context, sessaoID string) ([]poltrona.Poltrona, error)
}

// API reúne as dependências das rotas.
type API struct {
	Bloqueio CasoDeUsoBloqueio
	Mapa     CasoDeUsoMapa
	Auth     *Autenticador
	// Limite é o máximo de poltronas por bloqueio. Vive aqui só para compor o
	// `metadata.limite` da resposta 400, que diz ao cliente qual é o teto em
	// vez de deixá-lo descobrir por tentativa.
	Limite int
}

type solicitacaoBloqueio struct {
	PoltronasIDs []string `json:"poltronas_ids"`
}

type respostaBloqueio struct {
	ReservaID string `json:"reserva_id"`
	// RFC 3339, e não o epoch do gRPC: em REST a data legível é a convenção, e
	// o cliente HTTP não tem o schema do proto para saber a unidade.
	ExpiraEm string `json:"expira_em"`
	Mensagem string `json:"mensagem"`
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

// Rotas monta o roteador. São poucos caminhos: net/http basta, e um roteador de
// terceiro não se pagaria aqui (princípio I).
//
// /docs e /openapi.yaml ficam na raiz, e não sob /api/v1: servem o contrato,
// não fazem parte dele, e por isso não acompanham a versão da API.
func (a *API) Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/sessoes/{sessao_id}/bloqueios", a.bloquear)
	mux.HandleFunc("GET /api/v1/sessoes/{sessao_id}/poltronas", a.consultarMapa)

	mux.Handle("GET /openapi.yaml", openapi.HandlerEspecificacao())
	mux.Handle("GET /docs", openapi.HandlerUI("/openapi.yaml"))
	// A variante com barra evita um 404 confuso para quem digita a URL à mão.
	mux.Handle("GET /docs/", openapi.HandlerUI("/openapi.yaml"))
	return mux
}

// bloquear atende POST /api/v1/sessoes/{sessao_id}/bloqueios.
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
		// 409, e não 200 com sucesso=false: em HTTP o status É a categoria do
		// desfecho, e o Servico-Catalogo já traduz este caso para 409 quando
		// fala gRPC — as duas superfícies dizem a mesma coisa ao cliente final.
		escreverProblem(w, r, http.StatusConflict, tipoPoltronasIndisponivel,
			"Poltronas indisponíveis", resultado.Mensagem, nil)
		return
	}

	// 201: o bloqueio criou uma reserva, que é um recurso com identidade.
	responderJSON(w, http.StatusCreated, respostaBloqueio{
		ReservaID: resultado.Reserva.ID,
		ExpiraEm:  resultado.Reserva.ExpiraEm.UTC().Format(time.RFC3339),
		Mensagem:  resultado.Mensagem,
	})
}

// consultarMapa atende GET /api/v1/sessoes/{sessao_id}/poltronas.
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
		// O status já foi enviado: não há resposta de erro a dar ao cliente,
		// só registro para quem investiga depois.
		slog.Debug("falha ao escrever a resposta", slog.Any("erro", err))
	}
}
