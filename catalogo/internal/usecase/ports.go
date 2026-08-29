// Package usecase contém os casos de uso e as portas que eles exigem do mundo
// externo. As portas são declaradas aqui, do lado de dentro, e implementadas
// pelos adaptadores (constituição, princípio I).
package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// FiltroFilmes recorta a listagem de filmes.
// Status nil significa "sem filtro": aplica-se o recorte público (FR-008).
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

// FiltroSessoes recorta a grade. Campos vazios significam "sem filtro".
type FiltroSessoes struct {
	FilmeID  string
	CinemaID string
	Data     *DataDoDia
}

// DataDoDia é um dia civil usado como filtro. O repositório o traduz para o
// intervalo [inicio, fim) — comparar a coluna com uma função descartaria o índice.
type DataDoDia struct {
	Ano int
	Mes int
	Dia int
}

type SessaoRepository interface {
	Consultar(ctx context.Context, filtro FiltroSessoes, req shared.PageRequest) (shared.Page[catalogo.SessaoDetalhada], error)
	BuscarPorID(ctx context.Context, sessaoID string) (catalogo.Sessao, error)
}

// EstoqueGateway é a porta para o Servico-Estoque, dono da disponibilidade de
// poltronas (constituição, princípio III).
type EstoqueGateway interface {
	BloquearPoltronas(ctx context.Context, s reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error)
}
