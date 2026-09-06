package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

type filmesFalsos struct{ itens []catalogo.Filme }

func (f *filmesFalsos) BuscarPorID(_ context.Context, id string) (catalogo.Filme, error) {
	for _, filme := range f.itens {
		if filme.ID == id {
			return filme, nil
		}
	}
	return catalogo.Filme{}, shared.NaoEncontrado("filme", id)
}

func (f *filmesFalsos) Criar(_ context.Context, filme catalogo.Filme) error {
	f.itens = append(f.itens, filme)
	return nil
}

func (f *filmesFalsos) Atualizar(_ context.Context, filme catalogo.Filme) error {
	for i, existente := range f.itens {
		if existente.ID == filme.ID {
			f.itens[i] = filme
			return nil
		}
	}
	return shared.NaoEncontrado("filme", filme.ID)
}

func (f *filmesFalsos) MarcarForaDeCartaz(_ context.Context, id string) error {
	for i, existente := range f.itens {
		if existente.ID == id {
			f.itens[i].Status = catalogo.StatusForaDeCartaz
			return nil
		}
	}
	return shared.NaoEncontrado("filme", id)
}

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

func (c *cinemasFalsos) Listar(_ context.Context, filtro usecase.FiltroCinemas, req shared.PageRequest) (shared.Page[catalogo.Cinema], error) {
	var selecionados []catalogo.Cinema
	for _, cinema := range c.itens {
		if filtro.Ativo == nil || cinema.Ativo == *filtro.Ativo {
			selecionados = append(selecionados, cinema)
		}
	}
	return recortar(selecionados, req), nil
}

func (c *cinemasFalsos) BuscarPorID(_ context.Context, id string) (catalogo.Cinema, error) {
	for _, cinema := range c.itens {
		if cinema.ID == id {
			return cinema, nil
		}
	}
	return catalogo.Cinema{}, shared.NaoEncontrado("cinema", id)
}

func (c *cinemasFalsos) Criar(_ context.Context, cinema catalogo.Cinema) error {
	c.itens = append(c.itens, cinema)
	return nil
}

func (c *cinemasFalsos) Atualizar(_ context.Context, cinema catalogo.Cinema) error {
	for i, existente := range c.itens {
		if existente.ID == cinema.ID {
			c.itens[i] = cinema
			return nil
		}
	}
	return shared.NaoEncontrado("cinema", cinema.ID)
}

func (c *cinemasFalsos) Desativar(_ context.Context, id string) error {
	for i, existente := range c.itens {
		if existente.ID == id {
			c.itens[i].Ativo = false
			return nil
		}
	}
	return shared.NaoEncontrado("cinema", id)
}

func (c *cinemasFalsos) Existe(context.Context, string) (bool, error) { return c.existe, nil }

type salasFalsas struct {
	itens       []catalogo.Sala
	numeroEmUso bool
}

func (s *salasFalsas) Listar(_ context.Context, filtro usecase.FiltroSalas, req shared.PageRequest) (shared.Page[catalogo.Sala], error) {
	var selecionadas []catalogo.Sala
	for _, sala := range s.itens {
		if filtro.CinemaID != "" && sala.CinemaID != filtro.CinemaID {
			continue
		}
		if filtro.Ativo == nil || sala.Ativo == *filtro.Ativo {
			selecionadas = append(selecionadas, sala)
		}
	}
	return recortar(selecionadas, req), nil
}

func (s *salasFalsas) BuscarPorID(_ context.Context, id string) (catalogo.Sala, error) {
	for _, sala := range s.itens {
		if sala.ID == id {
			return sala, nil
		}
	}
	return catalogo.Sala{}, shared.NaoEncontrado("sala", id)
}

func (s *salasFalsas) Criar(_ context.Context, sala catalogo.Sala) error {
	s.itens = append(s.itens, sala)
	return nil
}

func (s *salasFalsas) Atualizar(_ context.Context, sala catalogo.Sala) error {
	for i, existente := range s.itens {
		if existente.ID == sala.ID {
			s.itens[i] = sala
			return nil
		}
	}
	return shared.NaoEncontrado("sala", sala.ID)
}

func (s *salasFalsas) Desativar(_ context.Context, id string) error {
	for i, existente := range s.itens {
		if existente.ID == id {
			s.itens[i].Ativo = false
			return nil
		}
	}
	return shared.NaoEncontrado("sala", id)
}

func (s *salasFalsas) NumeroEmUso(_ context.Context, cinemaID string, numero int, excetoID string) (bool, error) {
	if s.numeroEmUso {
		return true, nil
	}
	for _, sala := range s.itens {
		if sala.CinemaID == cinemaID && sala.Numero == numero && sala.Ativo && sala.ID != excetoID {
			return true, nil
		}
	}
	return false, nil
}

type sessoesFalsas struct {
	grade       []catalogo.SessaoDetalhada
	sessao      catalogo.Sessao
	erro        error
	itens       []catalogo.Sessao
	salaOcupada bool
}

func (s *sessoesFalsas) Consultar(_ context.Context, _ usecase.FiltroSessoes, req shared.PageRequest) (shared.Page[catalogo.SessaoDetalhada], error) {
	return recortar(s.grade, req), nil
}
func (s *sessoesFalsas) BuscarPorID(_ context.Context, id string) (catalogo.Sessao, error) {
	if s.erro != nil {
		return catalogo.Sessao{}, s.erro
	}
	for _, sessao := range s.itens {
		if sessao.ID == id {
			return sessao, nil
		}
	}
	// Sem grade montada, o dublê responde a sessão única configurada: é o que os
	// testes de reserva esperam.
	if len(s.itens) == 0 && s.sessao.ID != "" {
		return s.sessao, nil
	}
	return catalogo.Sessao{}, shared.NaoEncontrado("sessao", id)
}

func (s *sessoesFalsas) Criar(_ context.Context, sessao catalogo.Sessao) error {
	s.itens = append(s.itens, sessao)
	return nil
}

func (s *sessoesFalsas) Atualizar(_ context.Context, sessao catalogo.Sessao) error {
	for i, existente := range s.itens {
		if existente.ID == sessao.ID {
			s.itens[i] = sessao
			return nil
		}
	}
	return shared.NaoEncontrado("sessao", sessao.ID)
}

func (s *sessoesFalsas) Cancelar(_ context.Context, id string) error {
	for i, existente := range s.itens {
		if existente.ID == id {
			s.itens[i].Status = catalogo.SessaoCancelada
			return nil
		}
	}
	return shared.NaoEncontrado("sessao", id)
}

func (s *sessoesFalsas) SalaOcupada(context.Context, string, time.Time, time.Time, string) (bool, error) {
	return s.salaOcupada, nil
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

type ambiente struct {
	servidor *httptest.Server
	estoque  *estoqueContado
	sessoes  *sessoesFalsas
	cinemas  *cinemasFalsos
	salas    *salasFalsas
	filmes   *filmesFalsos
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
		salas:   &salasFalsas{},
		filmes:  &filmesFalsos{},
	}
	if ajustar != nil {
		ajustar(amb)
	}

	router := adapterhttp.NovoRouter(adapterhttp.Dependencias{
		Handlers: adapterhttp.Handlers{
			ListarFilmes:     usecase.ListarFilmes{Repo: amb.filmes},
			BuscarFilme:      usecase.BuscarFilme{Repo: amb.filmes},
			CriarFilme:       usecase.CriarFilme{Repo: amb.filmes, GerarID: gerarID()},
			AtualizarFilme:   usecase.AtualizarFilme{Repo: amb.filmes},
			RemoverFilme:     usecase.RemoverFilme{Repo: amb.filmes},
			ListarCinemas:    usecase.ListarCinemas{Repo: amb.cinemas},
			BuscarCinema:     usecase.BuscarCinema{Repo: amb.cinemas},
			CriarCinema:      usecase.CriarCinema{Repo: amb.cinemas, GerarID: gerarID()},
			AtualizarCinema:  usecase.AtualizarCinema{Repo: amb.cinemas},
			RemoverCinema:    usecase.RemoverCinema{Repo: amb.cinemas},
			ListarSalas:      usecase.ListarSalas{Cinemas: amb.cinemas, Salas: amb.salas},
			BuscarSala:       usecase.BuscarSala{Salas: amb.salas},
			CriarSala:        usecase.CriarSala{Cinemas: amb.cinemas, Salas: amb.salas, GerarID: gerarID()},
			AtualizarSala:    usecase.AtualizarSala{Cinemas: amb.cinemas, Salas: amb.salas},
			RemoverSala:      usecase.RemoverSala{Salas: amb.salas},
			ConsultarSessoes: usecase.ConsultarSessoes{Repo: amb.sessoes},
			BuscarSessao:     usecase.BuscarSessao{Repo: amb.sessoes},
			CriarSessao: usecase.CriarSessao{
				Sessoes: amb.sessoes, Filmes: amb.filmes, Salas: amb.salas, GerarID: gerarID(),
			},
			AtualizarSessao: usecase.AtualizarSessao{Sessoes: amb.sessoes, Filmes: amb.filmes, Salas: amb.salas},
			RemoverSessao:   usecase.RemoverSessao{Repo: amb.sessoes},
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
	return montar(t, func(a *ambiente) { a.filmes.itens = itens }).servidor
}

// Identificadores previsíveis: os testes de contrato conferem o `Location` e
// releem o filme criado.
func gerarID() func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", n)
	}
}

// requisitar cobre o que `obter` não alcança: verbo, corpo e credencial.
func requisitar(t *testing.T, s *httptest.Server, metodo, caminho, token, corpo string) (*http.Response, []byte) {
	t.Helper()
	var leitor io.Reader
	if corpo != "" {
		leitor = bytes.NewBufferString(corpo)
	}
	req, err := http.NewRequest(metodo, s.URL+caminho, leitor)
	if err != nil {
		t.Fatal(err)
	}
	if corpo != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	lido, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, lido
}

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
