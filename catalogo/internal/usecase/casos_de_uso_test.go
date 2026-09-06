package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type filmeRepoFalso struct {
	filtroRecebido   FiltroFilmes
	publicosRecebido []catalogo.StatusFilme
	criado           catalogo.Filme
	atualizado       catalogo.Filme
	removido         string
	erro             error
}

func (f *filmeRepoFalso) Listar(_ context.Context, filtro FiltroFilmes, publicos []catalogo.StatusFilme, req shared.PageRequest) (shared.Page[catalogo.Filme], error) {
	f.filtroRecebido, f.publicosRecebido = filtro, publicos
	return shared.NovaPage([]catalogo.Filme{{ID: "x"}}, 1, req), nil
}

func (f *filmeRepoFalso) BuscarPorID(_ context.Context, id string) (catalogo.Filme, error) {
	if f.erro != nil {
		return catalogo.Filme{}, f.erro
	}
	return catalogo.Filme{ID: id}, nil
}

func (f *filmeRepoFalso) Criar(_ context.Context, filme catalogo.Filme) error {
	f.criado = filme
	return f.erro
}

func (f *filmeRepoFalso) Atualizar(_ context.Context, filme catalogo.Filme) error {
	f.atualizado = filme
	return f.erro
}

func (f *filmeRepoFalso) MarcarForaDeCartaz(_ context.Context, id string) error {
	f.removido = id
	return f.erro
}

type sessaoRepoFalso struct {
	sessao       catalogo.Sessao
	erroBusca    error
	buscasFeitas int
}

func (s *sessaoRepoFalso) Consultar(context.Context, FiltroSessoes, shared.PageRequest) (shared.Page[catalogo.SessaoDetalhada], error) {
	return shared.Page[catalogo.SessaoDetalhada]{}, nil
}

func (s *sessaoRepoFalso) BuscarPorID(_ context.Context, id string) (catalogo.Sessao, error) {
	s.buscasFeitas++
	if s.erroBusca != nil {
		return catalogo.Sessao{}, s.erroBusca
	}
	return s.sessao, nil
}

type estoqueFalso struct {
	chamadas  int
	resultado reserva.ResultadoReserva
	erro      error
}

func (e *estoqueFalso) BloquearPoltronas(context.Context, reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error) {
	e.chamadas++
	return e.resultado, e.erro
}

type cinemaRepoFalso struct{ existe bool }

func (c *cinemaRepoFalso) Listar(_ context.Context, req shared.PageRequest) (shared.Page[catalogo.Cinema], error) {
	return shared.NovaPage([]catalogo.Cinema{}, 0, req), nil
}
func (c *cinemaRepoFalso) Existe(context.Context, string) (bool, error) { return c.existe, nil }

type salaRepoFalso struct{ chamado bool }

func (s *salaRepoFalso) ListarPorCinema(_ context.Context, _ string, req shared.PageRequest) (shared.Page[catalogo.Sala], error) {
	s.chamado = true
	return shared.NovaPage([]catalogo.Sala{}, 0, req), nil
}

func pagina() shared.PageRequest {
	p, _ := shared.NovoPageRequest(1, 20, 20, 100)
	return p
}

func TestListarFilmesSemFiltroAplicaRecortePublico(t *testing.T) {
	repo := &filmeRepoFalso{}
	_, err := ListarFilmes{Repo: repo}.Executar(context.Background(), FiltroFilmes{}, pagina())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.filtroRecebido.Status != nil {
		t.Fatal("sem filtro explícito, Status deveria chegar nil ao repositório")
	}
	if len(repo.publicosRecebido) != 2 {
		t.Fatalf("esperava 2 situações públicas, obteve %v", repo.publicosRecebido)
	}
	for _, s := range repo.publicosRecebido {
		if s == catalogo.StatusForaDeCartaz {
			t.Fatal("FORA_DE_CARTAZ não pode entrar no recorte público (FR-008)")
		}
	}
}

func TestListarFilmesComFiltroExplicitoRespeitaOPedido(t *testing.T) {
	repo := &filmeRepoFalso{}
	fora := catalogo.StatusForaDeCartaz
	_, err := ListarFilmes{Repo: repo}.Executar(context.Background(), FiltroFilmes{Status: &fora}, pagina())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.filtroRecebido.Status == nil || *repo.filtroRecebido.Status != catalogo.StatusForaDeCartaz {
		t.Fatal("filtro explícito deveria chegar intacto ao repositório")
	}
}

func TestListarSalasRecusaCinemaInexistente(t *testing.T) {
	salas := &salaRepoFalso{}
	_, err := ListarSalas{Cinemas: &cinemaRepoFalso{existe: false}, Salas: salas}.
		Executar(context.Background(), "id-qualquer", pagina())
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
	if salas.chamado {
		t.Fatal("não deveria consultar salas de um cinema inexistente")
	}
}

func sessaoReservavel() catalogo.Sessao {
	return catalogo.Sessao{
		ID:             "f781a9b2-11e2-4f81-a901-8890bc123456",
		Status:         catalogo.SessaoAgendada,
		DataHoraInicio: time.Date(2026, 9, 1, 20, 30, 0, 0, time.UTC),
	}
}

func agoraFixo() time.Time { return time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC) }

func solicitacao() reserva.SolicitacaoReserva {
	return reserva.SolicitacaoReserva{
		SessaoID:     "f781a9b2-11e2-4f81-a901-8890bc123456",
		PoltronasIDs: []string{"A1"},
		UsuarioID:    "usuario-1",
	}
}

func TestReservarCaminhoFeliz(t *testing.T) {
	est := &estoqueFalso{resultado: reserva.ResultadoReserva{ReservaID: "r1", ExpiraEm: agoraFixo().Add(10 * time.Minute)}}
	uc := ReservarPoltronas{Sessoes: &sessaoRepoFalso{sessao: sessaoReservavel()}, Estoque: est, Agora: agoraFixo}

	r, err := uc.Executar(context.Background(), solicitacao())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r.ReservaID != "r1" || est.chamadas != 1 {
		t.Fatalf("resultado=%v chamadas=%d", r, est.chamadas)
	}
}

func TestReservarNaoChamaEstoqueQuandoRecusaLocalmente(t *testing.T) {
	casos := []struct {
		nome        string
		solicitacao reserva.SolicitacaoReserva
		repo        *sessaoRepoFalso
		sentinela   error
	}{
		{
			nome:        "sem identidade",
			solicitacao: reserva.SolicitacaoReserva{SessaoID: "s", PoltronasIDs: []string{"A1"}},
			repo:        &sessaoRepoFalso{sessao: sessaoReservavel()},
			sentinela:   shared.ErrValidacao,
		},
		{
			nome:        "lista vazia",
			solicitacao: reserva.SolicitacaoReserva{SessaoID: "s", UsuarioID: "u"},
			repo:        &sessaoRepoFalso{sessao: sessaoReservavel()},
			sentinela:   shared.ErrValidacao,
		},
		{
			nome:        "poltronas duplicadas",
			solicitacao: reserva.SolicitacaoReserva{SessaoID: "s", UsuarioID: "u", PoltronasIDs: []string{"A1", "A1"}},
			repo:        &sessaoRepoFalso{sessao: sessaoReservavel()},
			sentinela:   shared.ErrValidacao,
		},
		{
			nome:        "sessão inexistente",
			solicitacao: solicitacao(),
			repo:        &sessaoRepoFalso{erroBusca: shared.ErrNaoEncontrado},
			sentinela:   shared.ErrNaoEncontrado,
		},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			est := &estoqueFalso{}
			uc := ReservarPoltronas{Sessoes: c.repo, Estoque: est, Agora: agoraFixo}
			_, err := uc.Executar(context.Background(), c.solicitacao)
			if !errors.Is(err, c.sentinela) {
				t.Fatalf("esperava %v, obteve %v", c.sentinela, err)
			}
			if est.chamadas != 0 {
				t.Fatalf("o estoque foi contatado %d vez(es) numa recusa local", est.chamadas)
			}
		})
	}
}

func TestReservarRecusaSessaoQueNaoAceitaMais(t *testing.T) {
	for _, s := range []catalogo.Sessao{
		{ID: "s", Status: catalogo.SessaoCancelada, DataHoraInicio: agoraFixo().Add(time.Hour)},
		{ID: "s", Status: catalogo.SessaoFinalizada, DataHoraInicio: agoraFixo().Add(-time.Hour)},
		{ID: "s", Status: catalogo.SessaoEmAndamento, DataHoraInicio: agoraFixo().Add(-time.Hour)},
		{ID: "s", Status: catalogo.SessaoAgendada, DataHoraInicio: agoraFixo().Add(-time.Minute)},
	} {
		est := &estoqueFalso{}
		uc := ReservarPoltronas{Sessoes: &sessaoRepoFalso{sessao: s}, Estoque: est, Agora: agoraFixo}
		_, err := uc.Executar(context.Background(), solicitacao())
		if !errors.Is(err, shared.ErrSessaoNaoReservavel) {
			t.Errorf("status %s: esperava ErrSessaoNaoReservavel, obteve %v", s.Status, err)
		}
		if est.chamadas != 0 {
			t.Errorf("status %s: estoque contatado indevidamente", s.Status)
		}
	}
}

func TestReservarRecusaSucessoSemDadosObrigatorios(t *testing.T) {
	est := &estoqueFalso{resultado: reserva.ResultadoReserva{ExpiraEm: agoraFixo()}}
	uc := ReservarPoltronas{Sessoes: &sessaoRepoFalso{sessao: sessaoReservavel()}, Estoque: est, Agora: agoraFixo}
	_, err := uc.Executar(context.Background(), solicitacao())
	if !errors.Is(err, shared.ErrRespostaInvalidaDoParceiro) {
		t.Fatalf("esperava ErrRespostaInvalidaDoParceiro, obteve %v", err)
	}
}

func TestReservarPropagaIndisponibilidade(t *testing.T) {
	for _, sentinela := range []error{shared.ErrPoltronasIndisponiveis, shared.ErrEstoqueIndisponivel} {
		est := &estoqueFalso{erro: sentinela}
		uc := ReservarPoltronas{Sessoes: &sessaoRepoFalso{sessao: sessaoReservavel()}, Estoque: est, Agora: agoraFixo}
		if _, err := uc.Executar(context.Background(), solicitacao()); !errors.Is(err, sentinela) {
			t.Errorf("esperava %v, obteve %v", sentinela, err)
		}
	}
}

func dadosValidos() catalogo.DadosFilme {
	return catalogo.DadosFilme{
		Titulo: "Duna: Parte 2", DuracaoMinutos: 166,
		ClassificacaoEtaria: "14 anos", Genero: "Ficção Científica",
	}
}

func TestCriarFilmeUsaOIdentificadorGerado(t *testing.T) {
	repo := &filmeRepoFalso{}
	uc := CriarFilme{Repo: repo, GerarID: func() string { return "id-fixo" }}

	filme, err := uc.Executar(context.Background(), dadosValidos())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if filme.ID != "id-fixo" || repo.criado.ID != "id-fixo" {
		t.Fatalf("o id gerado deveria chegar ao repositório e à resposta: %+v", repo.criado)
	}
	if filme.Status != catalogo.StatusEmCartaz {
		t.Fatalf("sem status no corpo, o filme deveria nascer EM_CARTAZ, veio %q", filme.Status)
	}
}

func TestCriarFilmeInvalidoNaoChegaAoRepositorio(t *testing.T) {
	repo := &filmeRepoFalso{}
	uc := CriarFilme{Repo: repo, GerarID: func() string { return "id-fixo" }}

	dados := dadosValidos()
	dados.Titulo = "   "
	_, err := uc.Executar(context.Background(), dados)

	if !errors.Is(err, shared.ErrValidacao) {
		t.Fatalf("esperava erro de validação, obteve %v", err)
	}
	if repo.criado.ID != "" {
		t.Fatal("filme inválido não pode ser gravado")
	}
}

func TestAtualizarFilmePreservaOIdentificadorDaRota(t *testing.T) {
	repo := &filmeRepoFalso{}
	filme, err := AtualizarFilme{Repo: repo}.Executar(context.Background(), "id-da-rota", dadosValidos())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if filme.ID != "id-da-rota" || repo.atualizado.ID != "id-da-rota" {
		t.Fatalf("o id da rota deveria identificar o filme atualizado: %+v", repo.atualizado)
	}
}

func TestAtualizarFilmePropagaNaoEncontrado(t *testing.T) {
	repo := &filmeRepoFalso{erro: shared.ErrNaoEncontrado}
	_, err := AtualizarFilme{Repo: repo}.Executar(context.Background(), "id-da-rota", dadosValidos())
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
}

func TestRemoverFilmeMarcaForaDeCartaz(t *testing.T) {
	repo := &filmeRepoFalso{}
	if err := (RemoverFilme{Repo: repo}).Executar(context.Background(), "id-x"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.removido != "id-x" {
		t.Fatalf("esperava remoção lógica de id-x, obteve %q", repo.removido)
	}
}

func TestRemoverFilmePropagaNaoEncontrado(t *testing.T) {
	repo := &filmeRepoFalso{erro: shared.ErrNaoEncontrado}
	err := RemoverFilme{Repo: repo}.Executar(context.Background(), "inexistente")
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
}
