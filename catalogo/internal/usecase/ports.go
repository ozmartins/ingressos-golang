package usecase

import (
	"context"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type FiltroFilmes struct {
	Status *catalogo.StatusFilme
}

type FilmeRepository interface {
	Listar(ctx context.Context, filtro FiltroFilmes, publicos []catalogo.StatusFilme, req shared.PageRequest) (shared.Page[catalogo.Filme], error)
	BuscarPorID(ctx context.Context, filmeID string) (catalogo.Filme, error)
	Criar(ctx context.Context, f catalogo.Filme) error
	Atualizar(ctx context.Context, f catalogo.Filme) error
	MarcarForaDeCartaz(ctx context.Context, filmeID string) error
}

type FiltroCinemas struct {
	Ativo *bool
}

type CinemaRepository interface {
	Listar(ctx context.Context, filtro FiltroCinemas, req shared.PageRequest) (shared.Page[catalogo.Cinema], error)
	BuscarPorID(ctx context.Context, cinemaID string) (catalogo.Cinema, error)
	Criar(ctx context.Context, c catalogo.Cinema) error
	Atualizar(ctx context.Context, c catalogo.Cinema) error
	Desativar(ctx context.Context, cinemaID string) error
	// Existe responde pela linha, não pela situação: as salas de um cinema
	// desativado seguem consultáveis, como as sessões de um filme fora de cartaz.
	Existe(ctx context.Context, cinemaID string) (bool, error)
}

type FiltroSalas struct {
	// Vazio significa "a rede toda": a sala é endereçada por si, e o cinema é
	// um recorte opcional da listagem, como na grade de sessões.
	CinemaID string
	Ativo    *bool
}

type SalaRepository interface {
	Listar(ctx context.Context, filtro FiltroSalas, req shared.PageRequest) (shared.Page[catalogo.Sala], error)
	BuscarPorID(ctx context.Context, salaID string) (catalogo.Sala, error)
	Criar(ctx context.Context, s catalogo.Sala) error
	Atualizar(ctx context.Context, s catalogo.Sala) error
	Desativar(ctx context.Context, salaID string) error
	// O número identifica a sala na grade: dois "3" ativos no mesmo cinema a
	// tornariam ambígua. `excetoID` deixa a sala se manter no próprio número.
	NumeroEmUso(ctx context.Context, cinemaID string, numero int, excetoID string) (bool, error)
}

type FiltroSessoes struct {
	FilmeID  string
	CinemaID string
	Data     *DataDoDia
}

type DataDoDia struct {
	Ano int
	Mes int
	Dia int
}

// Um fato pronto para sair, do jeito que ele será publicado. O núcleo o produz;
// o adaptador o grava na mesma transação do efeito que o produziu e, depois,
// entrega ao intermediário — a resposta ao cliente não espera por isso.
type FatoPendente struct {
	MessageID    string
	RoutingKey   string
	Payload      []byte
	TraceContext map[string]string
}

type SessaoRepository interface {
	Consultar(ctx context.Context, filtro FiltroSessoes, req shared.PageRequest) (shared.Page[catalogo.SessaoDetalhada], error)
	BuscarPorID(ctx context.Context, sessaoID string) (catalogo.Sessao, error)
	// A sessão e o anúncio dela são gravados juntos ou não são gravados.
	Criar(ctx context.Context, s catalogo.Sessao, fato FatoPendente) error
	Atualizar(ctx context.Context, s catalogo.Sessao) error
	Cancelar(ctx context.Context, sessaoID string) error
	// Uma sala projeta um filme de cada vez: a janela é `[inicio, fim)`, e o fim
	// de cada sessão concorrente sai da duração do filme dela.
	SalaOcupada(ctx context.Context, salaID string, inicio, fim time.Time, excetoID string) (bool, error)
}

type EstoqueGateway interface {
	BloquearPoltronas(ctx context.Context, s reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error)
}
