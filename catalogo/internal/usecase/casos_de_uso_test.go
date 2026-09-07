package usecase

import (
	"context"
	"encoding/json"
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
	duracao          int
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
	return catalogo.Filme{ID: id, DuracaoMinutos: f.duracao}, nil
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
	sessao           catalogo.Sessao
	erroBusca        error
	buscasFeitas     int
	criada           catalogo.Sessao
	fatoEnfileirado  FatoPendente
	atualizada       catalogo.Sessao
	cancelada        string
	salaOcupada      bool
	janelaConsultada [2]time.Time
	excetoRecebido   string
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

func (s *sessaoRepoFalso) Criar(_ context.Context, sessao catalogo.Sessao, fato FatoPendente) error {
	s.criada = sessao
	s.fatoEnfileirado = fato
	return nil
}

func (s *sessaoRepoFalso) Atualizar(_ context.Context, sessao catalogo.Sessao, fato FatoPendente) error {
	s.atualizada = sessao
	s.fatoEnfileirado = fato
	return nil
}

func (s *sessaoRepoFalso) Cancelar(_ context.Context, id string, fato FatoPendente) error {
	s.cancelada = id
	s.fatoEnfileirado = fato
	return nil
}

func (s *sessaoRepoFalso) SalaOcupada(_ context.Context, salaID string, inicio, fim time.Time, excetoID string) (bool, error) {
	s.janelaConsultada = [2]time.Time{inicio, fim}
	s.excetoRecebido = excetoID
	return s.salaOcupada, nil
}

type estoqueFalso struct {
	chamadas    int
	solicitacao reserva.SolicitacaoReserva
	resultado   reserva.ResultadoReserva
	erro        error
}

func (e *estoqueFalso) BloquearPoltronas(_ context.Context, s reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error) {
	e.chamadas++
	e.solicitacao = s
	return e.resultado, e.erro
}

type cinemaRepoFalso struct {
	existe         bool
	filtroRecebido FiltroCinemas
	criado         catalogo.Cinema
	atualizado     catalogo.Cinema
	desativado     string
	erro           error
}

func (c *cinemaRepoFalso) Listar(_ context.Context, filtro FiltroCinemas, req shared.PageRequest) (shared.Page[catalogo.Cinema], error) {
	c.filtroRecebido = filtro
	return shared.NovaPage([]catalogo.Cinema{}, 0, req), nil
}

func (c *cinemaRepoFalso) BuscarPorID(_ context.Context, id string) (catalogo.Cinema, error) {
	if c.erro != nil {
		return catalogo.Cinema{}, c.erro
	}
	return catalogo.Cinema{ID: id}, nil
}

func (c *cinemaRepoFalso) Criar(_ context.Context, cinema catalogo.Cinema) error {
	c.criado = cinema
	return c.erro
}

func (c *cinemaRepoFalso) Atualizar(_ context.Context, cinema catalogo.Cinema) error {
	c.atualizado = cinema
	return c.erro
}

func (c *cinemaRepoFalso) Desativar(_ context.Context, id string) error {
	c.desativado = id
	return c.erro
}

func (c *cinemaRepoFalso) Existe(context.Context, string) (bool, error) { return c.existe, nil }

type salaRepoFalso struct {
	chamado     bool
	sala        catalogo.Sala
	erroBusca   error
	numeroEmUso bool
	criada      catalogo.Sala
	atualizada  catalogo.Sala
	desativada  string
}

func (s *salaRepoFalso) Listar(_ context.Context, _ FiltroSalas, req shared.PageRequest) (shared.Page[catalogo.Sala], error) {
	s.chamado = true
	return shared.NovaPage([]catalogo.Sala{}, 0, req), nil
}

func (s *salaRepoFalso) BuscarPorID(_ context.Context, id string) (catalogo.Sala, error) {
	if s.erroBusca != nil {
		return catalogo.Sala{}, s.erroBusca
	}
	if s.sala.ID == "" {
		return catalogo.Sala{ID: id}, nil
	}
	return s.sala, nil
}

func (s *salaRepoFalso) Criar(_ context.Context, sala catalogo.Sala) error {
	s.criada = sala
	return nil
}

func (s *salaRepoFalso) Atualizar(_ context.Context, sala catalogo.Sala) error {
	s.atualizada = sala
	return nil
}

func (s *salaRepoFalso) Desativar(_ context.Context, id string) error {
	s.desativada = id
	return nil
}

func (s *salaRepoFalso) NumeroEmUso(context.Context, string, int, string) (bool, error) {
	return s.numeroEmUso, nil
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
		Executar(context.Background(), FiltroSalas{CinemaID: "id-qualquer"}, pagina())
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
	if salas.chamado {
		t.Fatal("não deveria consultar salas de um cinema inexistente")
	}
}

// Sem recorte por cinema não há cinema para conferir: a listagem vai direto ao
// repositório, mesmo que nenhum cinema exista.
func TestListarSalasSemCinemaNaoConfereCinema(t *testing.T) {
	salas := &salaRepoFalso{}
	cinemas := &cinemaRepoFalso{existe: false}
	if _, err := (ListarSalas{Cinemas: cinemas, Salas: salas}).
		Executar(context.Background(), FiltroSalas{}, pagina()); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !salas.chamado {
		t.Fatal("a listagem da rede deveria chegar ao repositório de salas")
	}
}

func TestAtualizarSalaRecusaTrocaDeCinema(t *testing.T) {
	salas := &salaRepoFalso{sala: catalogo.Sala{ID: "sala-1", CinemaID: "cinema-a", Numero: 3}}
	uc := AtualizarSala{Cinemas: &cinemaRepoFalso{existe: true}, Salas: salas}

	_, err := uc.Executar(context.Background(), "sala-1", catalogo.DadosSala{
		CinemaID: "cinema-b", Numero: 3, TipoTela: "2D", Fileiras: fileirasSala(),
	})
	if !errors.Is(err, shared.ErrConflito) {
		t.Fatalf("esperava ErrConflito, obteve %v", err)
	}
	if salas.atualizada.ID != "" {
		t.Fatal("a sala não deveria ter sido gravada")
	}
}

func TestAtualizarSalaSemCinemaIDMantemOCinemaAtual(t *testing.T) {
	salas := &salaRepoFalso{sala: catalogo.Sala{ID: "sala-1", CinemaID: "cinema-a", Numero: 3}}
	uc := AtualizarSala{Cinemas: &cinemaRepoFalso{existe: true}, Salas: salas}

	sala, err := uc.Executar(context.Background(), "sala-1", catalogo.DadosSala{
		Numero: 4, TipoTela: "3D", Fileiras: fileirasSala(),
	})
	if err != nil {
		t.Fatalf("não esperava erro, obteve %v", err)
	}
	if sala.CinemaID != "cinema-a" {
		t.Fatalf("esperava cinema-a, obteve %q", sala.CinemaID)
	}
	if salas.atualizada.CinemaID != "cinema-a" {
		t.Fatalf("a sala gravada deveria manter cinema-a, obteve %q", salas.atualizada.CinemaID)
	}
}

func sessaoReservavel() catalogo.Sessao {
	return catalogo.Sessao{
		ID:             "f781a9b2-11e2-4f81-a901-8890bc123456",
		Status:         catalogo.SessaoAgendada,
		DataHoraInicio: time.Date(2026, 9, 1, 20, 30, 0, 0, time.UTC),
		PrecoBase:      catalogo.DinheiroDeCentavos(4250),
	}
}

// O cliente pede poltronas; quanto custam é decisão do catálogo, dono do
// cadastro da sessão. Quem cobra recebe este valor pelo fato do estoque.
func TestReservarPoltronasDerivaOValorDoPrecoDaSessao(t *testing.T) {
	est := &estoqueFalso{resultado: reserva.ResultadoReserva{
		ReservaID: "9982a1b3-44c1-4221-a123-902183120192",
		ExpiraEm:  agoraFixo().Add(10 * time.Minute),
	}}
	uc := ReservarPoltronas{Sessoes: &sessaoRepoFalso{sessao: sessaoReservavel()}, Estoque: est, Agora: agoraFixo}

	s := solicitacao()
	s.PoltronasIDs = []string{"A1", "A2", "F1"}
	if _, err := uc.Executar(context.Background(), s); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// 42,50 × 3 poltronas, sem centavo perdido no caminho.
	if got := est.solicitacao.ValorTotal; got != "127.50" {
		t.Fatalf("valor_total = %q, esperado \"127.50\"", got)
	}
}

// O valor é derivado, não recebido: o que o cliente mandar é ignorado.
func TestReservarPoltronasIgnoraValorInformadoPeloCliente(t *testing.T) {
	est := &estoqueFalso{resultado: reserva.ResultadoReserva{
		ReservaID: "9982a1b3-44c1-4221-a123-902183120192",
		ExpiraEm:  agoraFixo().Add(10 * time.Minute),
	}}
	uc := ReservarPoltronas{Sessoes: &sessaoRepoFalso{sessao: sessaoReservavel()}, Estoque: est, Agora: agoraFixo}

	s := solicitacao()
	s.ValorTotal = "0.01"
	if _, err := uc.Executar(context.Background(), s); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if got := est.solicitacao.ValorTotal; got == "0.01" {
		t.Fatal("o valor informado pelo cliente não deveria chegar ao estoque")
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

func dadosCinemaValidos() catalogo.DadosCinema {
	return catalogo.DadosCinema{
		Nome:     "CineMark - Shopping Centro",
		Cidade:   "Florianópolis",
		Estado:   "sc",
		Endereco: "Rua X, 100",
	}
}

func TestListarCinemasSemFiltroAplicaRecortePublico(t *testing.T) {
	repo := &cinemaRepoFalso{}
	if _, err := (ListarCinemas{Repo: repo}).Executar(context.Background(), FiltroCinemas{}, pagina()); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.filtroRecebido.Ativo == nil || !*repo.filtroRecebido.Ativo {
		t.Fatalf("sem filtro explícito, o repositório deveria receber ativo=true: %+v", repo.filtroRecebido.Ativo)
	}
}

func TestListarCinemasRespeitaFiltroExplicito(t *testing.T) {
	repo := &cinemaRepoFalso{}
	inativo := false
	if _, err := (ListarCinemas{Repo: repo}).Executar(context.Background(), FiltroCinemas{Ativo: &inativo}, pagina()); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.filtroRecebido.Ativo == nil || *repo.filtroRecebido.Ativo {
		t.Fatal("o filtro explícito ativo=false deveria chegar ao repositório")
	}
}

func TestCriarCinemaUsaOIdentificadorGerado(t *testing.T) {
	repo := &cinemaRepoFalso{}
	uc := CriarCinema{Repo: repo, GerarID: func() string { return "id-fixo" }}

	cinema, err := uc.Executar(context.Background(), dadosCinemaValidos())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cinema.ID != "id-fixo" || repo.criado.ID != "id-fixo" {
		t.Fatalf("o id gerado deveria chegar ao repositório e à resposta: %+v", repo.criado)
	}
	if !cinema.Ativo {
		t.Fatal("sem `ativo` no corpo, o cinema deveria nascer ativo")
	}
	if cinema.Estado != "SC" {
		t.Fatalf("a sigla deveria ser normalizada para maiúsculas, veio %q", cinema.Estado)
	}
}

func TestCriarCinemaInvalidoNaoChegaAoRepositorio(t *testing.T) {
	repo := &cinemaRepoFalso{}
	uc := CriarCinema{Repo: repo, GerarID: func() string { return "id-fixo" }}

	dados := dadosCinemaValidos()
	dados.Estado = "SCA"
	_, err := uc.Executar(context.Background(), dados)

	if !errors.Is(err, shared.ErrValidacao) {
		t.Fatalf("esperava erro de validação, obteve %v", err)
	}
	if repo.criado.ID != "" {
		t.Fatal("cinema inválido não pode ser gravado")
	}
}

func TestAtualizarCinemaPreservaOIdentificadorDaRota(t *testing.T) {
	repo := &cinemaRepoFalso{}
	cinema, err := AtualizarCinema{Repo: repo}.Executar(context.Background(), "id-da-rota", dadosCinemaValidos())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cinema.ID != "id-da-rota" || repo.atualizado.ID != "id-da-rota" {
		t.Fatalf("o id da rota deveria identificar o cinema atualizado: %+v", repo.atualizado)
	}
}

func TestAtualizarCinemaPropagaNaoEncontrado(t *testing.T) {
	repo := &cinemaRepoFalso{erro: shared.ErrNaoEncontrado}
	_, err := AtualizarCinema{Repo: repo}.Executar(context.Background(), "id-da-rota", dadosCinemaValidos())
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
}

func TestRemoverCinemaDesativa(t *testing.T) {
	repo := &cinemaRepoFalso{}
	if err := (RemoverCinema{Repo: repo}).Executar(context.Background(), "id-x"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.desativado != "id-x" {
		t.Fatalf("esperava remoção lógica de id-x, obteve %q", repo.desativado)
	}
}

func fileirasSala() []catalogo.DadosFileira {
	return []catalogo.DadosFileira{{Fileira: "A", Assentos: 10}, {Fileira: "B", Assentos: 8}}
}

func dadosSala() catalogo.DadosSala {
	return catalogo.DadosSala{CinemaID: "cinema-1", Numero: 3, TipoTela: "IMAX", Fileiras: fileirasSala()}
}

func TestCriarSalaRecusaCinemaInexistente(t *testing.T) {
	salas := &salaRepoFalso{}
	uc := CriarSala{Cinemas: &cinemaRepoFalso{existe: false}, Salas: salas, GerarID: idFixo}

	_, err := uc.Executar(context.Background(), dadosSala())
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
	if salas.criada.ID != "" {
		t.Fatal("não deveria gravar sala em cinema inexistente")
	}
}

func TestCriarSalaRecusaNumeroJaUsado(t *testing.T) {
	salas := &salaRepoFalso{numeroEmUso: true}
	uc := CriarSala{Cinemas: &cinemaRepoFalso{existe: true}, Salas: salas, GerarID: idFixo}

	_, err := uc.Executar(context.Background(), dadosSala())
	if !errors.Is(err, shared.ErrConflito) {
		t.Fatalf("esperava ErrConflito, obteve %v", err)
	}
	if salas.criada.ID != "" {
		t.Fatal("não deveria gravar sala com número repetido")
	}
}

func TestCriarSalaInativaNaoDisputaONumero(t *testing.T) {
	inativa := false
	dados := dadosSala()
	dados.Ativo = &inativa

	salas := &salaRepoFalso{numeroEmUso: true}
	uc := CriarSala{Cinemas: &cinemaRepoFalso{existe: true}, Salas: salas, GerarID: idFixo}

	if _, err := uc.Executar(context.Background(), dados); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if salas.criada.ID == "" {
		t.Fatal("a sala inativa deveria ser gravada: o índice único vale entre as ativas")
	}
}

func TestCriarSalaTomaOCinemaDoCorpo(t *testing.T) {
	salas := &salaRepoFalso{}
	uc := CriarSala{Cinemas: &cinemaRepoFalso{existe: true}, Salas: salas, GerarID: idFixo}

	sala, err := uc.Executar(context.Background(), dadosSala())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if sala.CinemaID != "cinema-1" || salas.criada.CinemaID != "cinema-1" {
		t.Fatalf("cinema_id = %q, esperava o do corpo", salas.criada.CinemaID)
	}
}

func TestBuscarSalaInexistenteNaoEncontra(t *testing.T) {
	salas := &salaRepoFalso{erroBusca: shared.NaoEncontrado("sala", "sala-1")}

	_, err := BuscarSala{Salas: salas}.Executar(context.Background(), "sala-1")
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
	var ausente shared.RecursoAusente
	if !errors.As(err, &ausente) || ausente.Recurso != "sala" {
		t.Fatalf("o ausente deveria ser a sala, obteve %v", err)
	}
}

func TestRemoverSalaDesativa(t *testing.T) {
	salas := &salaRepoFalso{}

	if err := (RemoverSala{Salas: salas}).Executar(context.Background(), "sala-1"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if salas.desativada != "sala-1" {
		t.Fatalf("esperava remoção lógica de sala-1, obteve %q", salas.desativada)
	}
}

func dadosSessao() catalogo.DadosSessao {
	return catalogo.DadosSessao{
		FilmeID:        "filme-1",
		SalaID:         "sala-1",
		DataHoraInicio: time.Date(2026, 9, 20, 19, 30, 0, 0, time.UTC),
		Idioma:         "LEGENDADO",
		PrecoBase:      "42.50",
	}
}

func idFixo() string { return "id-gerado" }

// O anúncio da sessão sai junto com ela, e leva a planta da sala expandida
// assento a assento — é dela que quem consome monta a matriz de poltronas.
func TestCriarSessaoEnfileiraOAnuncioComAPlantaDaSala(t *testing.T) {
	layout, err := catalogo.NovoLayoutSala([]catalogo.DadosFileira{
		{Fileira: "A", Assentos: 2},
		{Fileira: "B", Assentos: 1, Tipo: "PCD"},
	})
	if err != nil {
		t.Fatal(err)
	}

	sessoes := &sessaoRepoFalso{}
	salas := &salaRepoFalso{sala: catalogo.Sala{ID: "sala-1", Layout: layout}}
	uc := CriarSessao{
		Sessoes: sessoes, Filmes: &filmeRepoFalso{}, Salas: salas, GerarID: idFixo,
		Agora:          func() time.Time { return time.Date(2026, 9, 6, 18, 0, 0, 0, time.UTC) },
		TraceContextDe: func(context.Context) map[string]string { return map[string]string{"traceparent": "00-abc-def-01"} },
	}

	sessao, err := uc.Executar(context.Background(), dadosSessao())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	fato := sessoes.fatoEnfileirado
	if fato.RoutingKey != RoutingKeySessaoCriada {
		t.Errorf("routing key = %q, esperava %q", fato.RoutingKey, RoutingKeySessaoCriada)
	}
	// É pelo `message_id` que quem consome descarta a repetição.
	if fato.MessageID != sessao.ID {
		t.Errorf("message_id = %q, esperava o id da sessão %q", fato.MessageID, sessao.ID)
	}
	if fato.TraceContext["traceparent"] == "" {
		t.Error("o contexto de rastreamento da requisição deveria viajar com o fato")
	}

	var evento EventoSessaoCriada
	if err := json.Unmarshal(fato.Payload, &evento); err != nil {
		t.Fatalf("o corpo do fato não é JSON válido: %v", err)
	}
	if evento.Evento != "SESSAO_CRIADA" || evento.Versao != 1 {
		t.Errorf("envelope inesperado: %+v", evento)
	}
	if evento.SessaoID != sessao.ID || evento.SalaID != sessao.SalaID {
		t.Errorf("o fato deveria identificar a sessão e a sala: %+v", evento)
	}
	if evento.OcorridoEm != "2026-09-06T18:00:00Z" {
		t.Errorf("ocorrido_em = %q", evento.OcorridoEm)
	}

	esperadas := []PoltronaNoFato{
		{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
		{Fileira: "A", Numero: 2, Tipo: "NORMAL"},
		{Fileira: "B", Numero: 1, Tipo: "PCD"},
	}
	if len(evento.Poltronas) != len(esperadas) {
		t.Fatalf("anunciou %d poltronas, esperava %d", len(evento.Poltronas), len(esperadas))
	}
	for i, esperada := range esperadas {
		if evento.Poltronas[i] != esperada {
			t.Errorf("poltrona %d = %+v, esperava %+v", i, evento.Poltronas[i], esperada)
		}
	}
}

// A sessão que não chega a ser criada não anuncia nada.
func TestCriarSessaoRecusadaNaoEnfileiraAnuncio(t *testing.T) {
	sessoes := &sessaoRepoFalso{salaOcupada: true}
	uc := CriarSessao{Sessoes: sessoes, Filmes: &filmeRepoFalso{}, Salas: &salaRepoFalso{}, GerarID: idFixo}

	if _, err := uc.Executar(context.Background(), dadosSessao()); err == nil {
		t.Fatal("esperava recusa por sala ocupada")
	}
	if sessoes.fatoEnfileirado.MessageID != "" {
		t.Fatalf("não deveria anunciar sessão que não existe: %+v", sessoes.fatoEnfileirado)
	}
}

func TestCriarSessaoRecusaSalaOcupada(t *testing.T) {
	sessoes := &sessaoRepoFalso{salaOcupada: true}
	uc := CriarSessao{Sessoes: sessoes, Filmes: &filmeRepoFalso{}, Salas: &salaRepoFalso{}, GerarID: idFixo}

	_, err := uc.Executar(context.Background(), dadosSessao())
	if !errors.Is(err, shared.ErrConflito) {
		t.Fatalf("esperava ErrConflito, obteve %v", err)
	}
	if sessoes.criada.ID != "" {
		t.Fatal("não deveria gravar sessão sobreposta")
	}
}

func TestCriarSessaoCalculaAJanelaComADuracaoDoFilme(t *testing.T) {
	sessoes := &sessaoRepoFalso{}
	filmes := &filmeRepoFalso{duracao: 148}
	uc := CriarSessao{Sessoes: sessoes, Filmes: filmes, Salas: &salaRepoFalso{}, GerarID: idFixo}

	if _, err := uc.Executar(context.Background(), dadosSessao()); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	inicio, fim := sessoes.janelaConsultada[0], sessoes.janelaConsultada[1]
	if esperado := time.Date(2026, 9, 20, 21, 58, 0, 0, time.UTC); !fim.Equal(esperado) {
		t.Fatalf("fim da janela = %s, esperava %s", fim, esperado)
	}
	if !inicio.Equal(dadosSessao().DataHoraInicio) {
		t.Fatalf("início da janela = %s", inicio)
	}
	if sessoes.excetoRecebido != "" {
		t.Fatal("na criação não há sessão a excluir da checagem")
	}
}

func TestCriarSessaoCanceladaNaoOcupaASala(t *testing.T) {
	dados := dadosSessao()
	dados.Status = string(catalogo.SessaoCancelada)

	sessoes := &sessaoRepoFalso{salaOcupada: true}
	uc := CriarSessao{Sessoes: sessoes, Filmes: &filmeRepoFalso{}, Salas: &salaRepoFalso{}, GerarID: idFixo}

	if _, err := uc.Executar(context.Background(), dados); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if sessoes.criada.ID == "" {
		t.Fatal("uma sessão cancelada não disputa a sala e deveria ser gravada")
	}
}

func TestCriarSessaoRecusaFilmeInexistente(t *testing.T) {
	sessoes := &sessaoRepoFalso{}
	filmes := &filmeRepoFalso{erro: shared.NaoEncontrado("filme", "filme-1")}
	uc := CriarSessao{Sessoes: sessoes, Filmes: filmes, Salas: &salaRepoFalso{}, GerarID: idFixo}

	_, err := uc.Executar(context.Background(), dadosSessao())
	var ausente shared.RecursoAusente
	if !errors.As(err, &ausente) || ausente.Recurso != "filme" {
		t.Fatalf("o ausente deveria ser o filme, obteve %v", err)
	}
	if sessoes.criada.ID != "" {
		t.Fatal("não deveria gravar sessão de filme inexistente")
	}
}

func TestCriarSessaoRecusaSalaInexistente(t *testing.T) {
	sessoes := &sessaoRepoFalso{}
	salas := &salaRepoFalso{erroBusca: shared.NaoEncontrado("sala", "sala-1")}
	uc := CriarSessao{Sessoes: sessoes, Filmes: &filmeRepoFalso{}, Salas: salas, GerarID: idFixo}

	_, err := uc.Executar(context.Background(), dadosSessao())
	var ausente shared.RecursoAusente
	if !errors.As(err, &ausente) || ausente.Recurso != "sala" {
		t.Fatalf("o ausente deveria ser a sala, obteve %v", err)
	}
	if sessoes.criada.ID != "" {
		t.Fatal("não deveria gravar sessão em sala inexistente")
	}
}

func TestAtualizarSessaoIgnoraAPropriaNaChecagemDeOcupacao(t *testing.T) {
	atual := sessaoReservavel()
	atual.SalaID = dadosSessao().SalaID
	sessoes := &sessaoRepoFalso{sessao: atual}
	uc := AtualizarSessao{
		Sessoes: sessoes, Filmes: &filmeRepoFalso{duracao: 100}, Salas: &salaRepoFalso{},
		GerarID: idFixo, Agora: agoraFixo,
	}

	if _, err := uc.Executar(context.Background(), "sessao-1", dadosSessao()); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if sessoes.excetoRecebido != "sessao-1" {
		t.Fatalf("excetoID = %q, esperava a própria sessão", sessoes.excetoRecebido)
	}
}

// A sala é do cadastro da sessão, não do estado que o PUT redesenha: trocá-la
// apagaria o chão sob quem já reservou. Mesma regra do `cinema_id` da sala.
func TestAtualizarSessaoRecusaTrocaDeSala(t *testing.T) {
	atual := sessaoReservavel()
	atual.SalaID = "sala-original"
	sessoes := &sessaoRepoFalso{sessao: atual}
	uc := AtualizarSessao{
		Sessoes: sessoes, Filmes: &filmeRepoFalso{duracao: 100}, Salas: &salaRepoFalso{},
		GerarID: idFixo, Agora: agoraFixo,
	}

	dados := dadosSessao()
	dados.SalaID = "outra-sala"
	if _, err := uc.Executar(context.Background(), "sessao-1", dados); !errors.Is(err, shared.ErrConflito) {
		t.Fatalf("esperava ErrConflito, obteve %v", err)
	}
	if sessoes.atualizada.ID != "" {
		t.Fatal("a sessão não deveria ter sido gravada")
	}
	if sessoes.fatoEnfileirado.MessageID != "" {
		t.Fatal("nada foi alterado; nada deveria ser anunciado")
	}
}

func TestAtualizarSessaoSemSalaIDMantemASalaAtual(t *testing.T) {
	atual := sessaoReservavel()
	atual.SalaID = "sala-original"
	sessoes := &sessaoRepoFalso{sessao: atual}
	uc := AtualizarSessao{
		Sessoes: sessoes, Filmes: &filmeRepoFalso{duracao: 100}, Salas: &salaRepoFalso{},
		GerarID: idFixo, Agora: agoraFixo,
	}

	dados := dadosSessao()
	dados.SalaID = ""
	sessao, err := uc.Executar(context.Background(), "sessao-1", dados)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if sessao.SalaID != "sala-original" {
		t.Fatalf("sala = %q, esperava a atual", sessao.SalaID)
	}
}

// Alterar anuncia. Sem isso, quem tem estado preso à sessão não fica sabendo.
func TestAtualizarSessaoEnfileiraOAnuncio(t *testing.T) {
	atual := sessaoReservavel()
	atual.SalaID = dadosSessao().SalaID
	sessoes := &sessaoRepoFalso{sessao: atual}
	uc := AtualizarSessao{
		Sessoes: sessoes, Filmes: &filmeRepoFalso{duracao: 100}, Salas: &salaRepoFalso{},
		GerarID: idFixo, Agora: agoraFixo,
		TraceContextDe: func(context.Context) map[string]string { return map[string]string{"traceparent": "00-abc-def-01"} },
	}

	dados := dadosSessao()
	dados.PrecoBase = "55.00"
	if _, err := uc.Executar(context.Background(), "sessao-1", dados); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	fato := sessoes.fatoEnfileirado
	if fato.RoutingKey != RoutingKeySessaoAlterada {
		t.Errorf("routing key = %q", fato.RoutingKey)
	}
	// Identificador próprio, e não o da sessão: duas alterações não podem
	// colidir na caixa de saída, cujo `message_id` é único.
	if fato.MessageID != idFixo() {
		t.Errorf("message_id = %q, esperava um identificador próprio do fato", fato.MessageID)
	}
	if fato.TraceContext["traceparent"] == "" {
		t.Error("o contexto de rastreamento deveria viajar com o fato")
	}

	var evento EventoSessaoAlterada
	if err := json.Unmarshal(fato.Payload, &evento); err != nil {
		t.Fatalf("o corpo não é JSON válido: %v", err)
	}
	if evento.Evento != "SESSAO_ALTERADA" || evento.Versao != 1 {
		t.Errorf("envelope inesperado: %+v", evento)
	}
	if evento.PrecoBase != "55.00" || evento.SalaID != dadosSessao().SalaID {
		t.Errorf("o fato deveria levar o estado final: %+v", evento)
	}
}

// Cancelar anuncia: é o fato que solta as reservas pendentes no estoque.
func TestRemoverSessaoEnfileiraOAnuncio(t *testing.T) {
	sessoes := &sessaoRepoFalso{sessao: sessaoReservavel()}
	uc := RemoverSessao{Repo: sessoes, Agora: agoraFixo}

	if err := uc.Executar(context.Background(), "sessao-1"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	fato := sessoes.fatoEnfileirado
	if fato.RoutingKey != RoutingKeySessaoCancelada {
		t.Errorf("routing key = %q", fato.RoutingKey)
	}
	// O cancelamento é terminal e acontece uma vez: a chave é estável, e é ela
	// que dá idempotência a quem consome.
	if fato.MessageID != "sessao-1:cancelada" {
		t.Errorf("message_id = %q", fato.MessageID)
	}

	var evento EventoSessaoCancelada
	if err := json.Unmarshal(fato.Payload, &evento); err != nil {
		t.Fatalf("o corpo não é JSON válido: %v", err)
	}
	if evento.Evento != "SESSAO_CANCELADA" || evento.SessaoID != "sessao-1" {
		t.Errorf("fato inesperado: %+v", evento)
	}
}

func TestAtualizarSessaoInexistenteNaoGrava(t *testing.T) {
	sessoes := &sessaoRepoFalso{erroBusca: shared.NaoEncontrado("sessao", "sessao-1")}
	uc := AtualizarSessao{Sessoes: sessoes, Filmes: &filmeRepoFalso{}, Salas: &salaRepoFalso{}}

	_, err := uc.Executar(context.Background(), "sessao-1", dadosSessao())
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
	if sessoes.atualizada.ID != "" {
		t.Fatal("não deveria gravar sessão inexistente")
	}
}

func TestRemoverSessaoCancela(t *testing.T) {
	sessoes := &sessaoRepoFalso{}
	if err := (RemoverSessao{Repo: sessoes}).Executar(context.Background(), "sessao-1"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if sessoes.cancelada != "sessao-1" {
		t.Fatalf("cancelada = %q", sessoes.cancelada)
	}
}
