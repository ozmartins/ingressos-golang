package usecase

import (
	"context"

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

type SalaRepository interface {
	ListarPorCinema(ctx context.Context, cinemaID string, req shared.PageRequest) (shared.Page[catalogo.Sala], error)
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

type SessaoRepository interface {
	Consultar(ctx context.Context, filtro FiltroSessoes, req shared.PageRequest) (shared.Page[catalogo.SessaoDetalhada], error)
	BuscarPorID(ctx context.Context, sessaoID string) (catalogo.Sessao, error)
}

type EstoqueGateway interface {
	BloquearPoltronas(ctx context.Context, s reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error)
}
