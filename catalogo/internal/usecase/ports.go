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
}

type CinemaRepository interface {
	Listar(ctx context.Context, req shared.PageRequest) (shared.Page[catalogo.Cinema], error)
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
