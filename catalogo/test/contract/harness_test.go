// Package contract verifica que as respostas HTTP obedecem ao contrato
// publicado em contracts/openapi.yaml e contracts/errors.md.
package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adapterhttp "github.com/oseias/ingressos-golang/catalogo/internal/adapter/http"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

// --- dublês ------------------------------------------------------------------

type filmesFalsos struct{ itens []catalogo.Filme }

func (f *filmesFalsos) Listar(_ context.Context, filtro usecase.FiltroFilmes, publicos []catalogo.StatusFilme, req shared.PageRequest) (shared.Page[catalogo.Filme], error) {
	permitido := func(s catalogo.StatusFilme) bool {
		if filtro.Status != nil {
			return s == *filtro.Status
		}
		for _, p := range publicos {
			if s == p {
				return true
			}
		}
		return false
	}
	var selecionados []catalogo.Filme
	for _, f := range f.itens {
		if permitido(f.Status) {
			selecionados = append(selecionados, f)
		}
	}
	return recortar(selecionados, req), nil
}

type cinemasFalsos struct {
	itens  []catalogo.Cinema
	existe bool
}

func (c *cinemasFalsos) Listar(_ context.Context, req shared.PageRequest) (shared.Page[catalogo.Cinema], error) {
	return recortar(c.itens, req), nil
}
func (c *cinemasFalsos) Existe(context.Context, string) (bool, error) { return c.existe, nil }

type salasFalsas struct{ itens []catalogo.Sala }

func (s *salasFalsas) ListarPorCinema(_ context.Context, _ string, req shared.PageRequest) (shared.Page[catalogo.Sala], error) {
	return recortar(s.itens, req), nil
}

type sessoesFalsas struct {
	grade  []catalogo.SessaoDetalhada
	sessao catalogo.Sessao
	erro   error
}

func (s *sessoesFalsas) Consultar(_ context.Context, _ usecase.FiltroSessoes, req shared.PageRequest) (shared.Page[catalogo.SessaoDetalhada], error) {
	return recortar(s.grade, req), nil
}
func (s *sessoesFalsas) BuscarPorID(context.Context, string) (catalogo.Sessao, error) {
	if s.erro != nil {
		return catalogo.Sessao{}, s.erro
	}
	return s.sessao, nil
}

type estoqueContado struct {
	chamadas  int
	resultado reserva.ResultadoReserva
	erro      error
}

func (e *estoqueContado) BloquearPoltronas(context.Context, reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error) {
	e.chamadas++
	return e.resultado, e.erro
}

type verificadorFalso struct{ usuarioID string }

func (v verificadorFalso) Verificar(_ context.Context, token string) (identidade.Identidade, error) {
	if token != "token-bom" {
		return identidade.Identidade{}, identidade.ErrCredencialInvalida
	}
	return identidade.Identidade{UsuarioID: v.usuarioID}, nil
}

func recortar[T any](todos []T, req shared.PageRequest) shared.Page[T] {
	inicio := req.Offset()
	if inicio > len(todos) {
		inicio = len(todos)
	}
	fim := inicio + req.Limit()
	if fim > len(todos) {
		fim = len(todos)
	}
	return shared.NovaPage(todos[inicio:fim], len(todos), req)
}

// --- montagem ----------------------------------------------------------------

type ambiente struct {
	servidor *httptest.Server
	estoque  *estoqueContado
	sessoes  *sessoesFalsas
	cinemas  *cinemasFalsos
}

const agoraDeTeste = "2026-09-01T10:00:00Z"

func agora() time.Time {
	t, _ := time.Parse(time.RFC3339, agoraDeTeste)
	return t
}

func montar(t *testing.T, ajustar func(*ambiente)) *ambiente {
	t.Helper()
	amb := &ambiente{
		estoque: &estoqueContado{},
		sessoes: &sessoesFalsas{},
		cinemas: &cinemasFalsos{existe: true},
	}
	filmes := &filmesFalsos{}
	salas := &salasFalsas{}
	if ajustar != nil {
		ajustar(amb)
	}

	router := adapterhttp.NovoRouter(adapterhttp.Dependencias{
		Handlers: adapterhttp.Handlers{
			ListarFilmes:     usecase.ListarFilmes{Repo: filmes},
			ListarCinemas:    usecase.ListarCinemas{Repo: amb.cinemas},
			ListarSalas:      usecase.ListarSalas{Cinemas: amb.cinemas, Salas: salas},
			ConsultarSessoes: usecase.ConsultarSessoes{Repo: amb.sessoes},
			ReservarPoltronas: usecase.ReservarPoltronas{
				Sessoes: amb.sessoes, Estoque: amb.estoque, Agora: agora,
			},
			Limites: adapterhttp.LimitesPaginacao{Padrao: 20, Maximo: 100},
		},
		Saude:       func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		Verificador: verificadorFalso{usuarioID: "usuario-1"},
	})
	amb.servidor = httptest.NewServer(router)
	t.Cleanup(amb.servidor.Close)
	return amb
}

func montarComFilmes(t *testing.T, itens []catalogo.Filme) *httptest.Server {
	t.Helper()
	filmes := &filmesFalsos{itens: itens}
	cinemas := &cinemasFalsos{existe: true}
	router := adapterhttp.NovoRouter(adapterhttp.Dependencias{
		Handlers: adapterhttp.Handlers{
			ListarFilmes:     usecase.ListarFilmes{Repo: filmes},
			ListarCinemas:    usecase.ListarCinemas{Repo: cinemas},
			ListarSalas:      usecase.ListarSalas{Cinemas: cinemas, Salas: &salasFalsas{}},
			ConsultarSessoes: usecase.ConsultarSessoes{Repo: &sessoesFalsas{}},
			Limites:          adapterhttp.LimitesPaginacao{Padrao: 20, Maximo: 100},
		},
		Saude:       func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		Verificador: verificadorFalso{},
	})
	s := httptest.NewServer(router)
	t.Cleanup(s.Close)
	return s
}

// --- utilitários de asserção -------------------------------------------------

type envelopeGenerico struct {
	Itens  []map[string]any `json:"itens"`
	Pagina struct {
		Pagina     int  `json:"pagina"`
		Tamanho    int  `json:"tamanho"`
		Total      int  `json:"total"`
		TemProxima bool `json:"tem_proxima"`
	} `json:"pagina"`
}

func obter(t *testing.T, s *httptest.Server, caminho string) (*http.Response, []byte) {
	t.Helper()
	resp, err := s.Client().Get(s.URL + caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	corpo := make([]byte, 0)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		corpo = append(corpo, buf[:n]...)
		if err != nil {
			break
		}
	}
	return resp, corpo
}

func decodificarEnvelope(t *testing.T, corpo []byte) envelopeGenerico {
	t.Helper()
	var e envelopeGenerico
	if err := json.Unmarshal(corpo, &e); err != nil {
		t.Fatalf("resposta não é o envelope de paginação: %v (corpo: %s)", err, corpo)
	}
	return e
}

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func decodificarProblem(t *testing.T, resp *http.Response, corpo []byte) problem {
	t.Helper()
	if ct := resp.Header.Get("Content-Type"); ct[:len("application/problem+json")] != "application/problem+json" {
		t.Fatalf("erro deveria ser problem+json, veio %q", ct)
	}
	var p problem
	if err := json.Unmarshal(corpo, &p); err != nil {
		t.Fatalf("erro não é problem+json válido: %v (corpo: %s)", err, corpo)
	}
	if p.Status != resp.StatusCode {
		t.Errorf("campo status (%d) diverge do código HTTP (%d)", p.Status, resp.StatusCode)
	}
	return p
}
