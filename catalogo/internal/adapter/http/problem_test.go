package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func executar(t *testing.T, fn func(w http.ResponseWriter, r *http.Request)) (*httptest.ResponseRecorder, Problem) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/filmes", nil)
	fn(w, r)
	var p Problem
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("resposta não é JSON válido: %v (corpo: %s)", err, w.Body.String())
	}
	return w, p
}

func TestTodasAsCategoriasDoContrato(t *testing.T) {
	casos := []struct {
		categoria string
		status    int
	}{
		{catParametroInvalido, http.StatusBadRequest},
		{catCorpoInvalido, http.StatusBadRequest},
		{catNaoAutenticado, http.StatusUnauthorized},
		{catCinemaNaoEncontrado, http.StatusNotFound},
		{catSessaoNaoEncontrada, http.StatusNotFound},
		{catSessaoNaoReservavel, http.StatusUnprocessableEntity},
		{catPoltronasIndisp, http.StatusConflict},
		{catEstoqueIndisponivel, http.StatusServiceUnavailable},
		{catRespostaInvalida, http.StatusBadGateway},
		{catErroInterno, http.StatusInternalServerError},
	}
	if len(casos) != len(categorias) {
		t.Fatalf("o teste cobre %d categorias, o código define %d", len(casos), len(categorias))
	}
	for _, c := range casos {
		t.Run(c.categoria, func(t *testing.T) {
			w, p := executar(t, func(w http.ResponseWriter, r *http.Request) {
				EscreverProblem(w, r, c.categoria, "detalhe")
			})
			if w.Code != c.status {
				t.Errorf("status: esperava %d, obteve %d", c.status, w.Code)
			}
			if p.Type != BaseURIErros+c.categoria {
				t.Errorf("type: esperava %s, obteve %s", BaseURIErros+c.categoria, p.Type)
			}
			if p.Status != c.status {
				t.Errorf("campo status divergente do código HTTP: %d vs %d", p.Status, w.Code)
			}
			if p.Title == "" {
				t.Error("title vazio")
			}
			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
				t.Errorf("content-type: obteve %q", ct)
			}
		})
	}
}

func TestCategoriaDesconhecidaViraErroInterno(t *testing.T) {
	w, p := executar(t, func(w http.ResponseWriter, r *http.Request) {
		EscreverProblem(w, r, "categoria-que-nao-existe", "x")
	})
	if w.Code != http.StatusInternalServerError || p.Type != BaseURIErros+catErroInterno {
		t.Fatalf("esperava fallback para erro-interno, obteve %d / %s", w.Code, p.Type)
	}
}

func TestErroDeDominioMapeiaParaCategoria(t *testing.T) {
	casos := []struct {
		err       error
		contexto  string
		categoria string
		status    int
	}{
		{fmt.Errorf("%w: page inválida", shared.ErrValidacao), "", catParametroInvalido, 400},
		{shared.ErrNaoEncontrado, "cinema", catCinemaNaoEncontrado, 404},
		{shared.ErrNaoEncontrado, "sessao", catSessaoNaoEncontrada, 404},
		{shared.ErrSessaoNaoReservavel, "", catSessaoNaoReservavel, 422},
		{shared.ErrPoltronasIndisponiveis, "", catPoltronasIndisp, 409},
		{shared.ErrEstoqueIndisponivel, "", catEstoqueIndisponivel, 503},
		{shared.ErrRespostaInvalidaDoParceiro, "", catRespostaInvalida, 502},
		{errors.New("qualquer coisa inesperada"), "", catErroInterno, 500},
	}
	for _, c := range casos {
		w, p := executar(t, func(w http.ResponseWriter, r *http.Request) {
			EscreverErroDeDominio(w, r, c.err, c.contexto)
		})
		if w.Code != c.status || p.Type != BaseURIErros+c.categoria {
			t.Errorf("%v (contexto %q): esperava %d/%s, obteve %d/%s", c.err, c.contexto, c.status, c.categoria, w.Code, p.Type)
		}
	}
}

func TestNaoVazaDetalheInterno(t *testing.T) {
	interno := errors.New(`pq: relation "sessoes" does not exist (host=10.0.0.5 port=5432)`)
	_, p := executar(t, func(w http.ResponseWriter, r *http.Request) {
		EscreverErroDeDominio(w, r, interno, "sessao")
	})
	corpo := p.Detail + p.Title
	for _, vazamento := range []string{"pq:", "relation", "10.0.0.5", "5432"} {
		if strings.Contains(corpo, vazamento) {
			t.Errorf("resposta vazou %q: %s", vazamento, corpo)
		}
	}
}

func TestDetailNaoRepeteONomeDoSentinela(t *testing.T) {
	_, p := executar(t, func(w http.ResponseWriter, r *http.Request) {
		EscreverErroDeDominio(w, r, fmt.Errorf("%w: page deve ser maior ou igual a 1", shared.ErrValidacao), "")
	})
	if strings.HasPrefix(p.Detail, shared.ErrValidacao.Error()) {
		t.Fatalf("detail repete o sentinela: %q", p.Detail)
	}
	if p.Detail != "page deve ser maior ou igual a 1" {
		t.Fatalf("detail inesperado: %q", p.Detail)
	}
}

func TestCamposDeValidacaoMultiplos(t *testing.T) {
	_, p := executar(t, func(w http.ResponseWriter, r *http.Request) {
		EscreverProblem(w, r, catCorpoInvalido, "corpo inválido",
			CampoErro{Campo: "poltronas_ids", Mensagem: "não pode ser vazia"},
			CampoErro{Campo: "page", Mensagem: "deve ser >= 1"})
	})
	if len(p.Errors) != 2 {
		t.Fatalf("esperava 2 campos, obteve %d", len(p.Errors))
	}
}
