package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type SalaRepository struct{ pool *pgxpool.Pool }

func NovoSalaRepository(p *pgxpool.Pool) *SalaRepository { return &SalaRepository{pool: p} }

func (r *SalaRepository) ListarPorCinema(ctx context.Context, cinemaID string, req shared.PageRequest) (shared.Page[catalogo.Sala], error) {
	const sqlPagina = `SELECT id, cinema_id, numero, tipo_tela, capacidade_total FROM salas
	                   WHERE cinema_id = $1 ORDER BY numero, id LIMIT $2 OFFSET $3`
	const sqlTotal = `SELECT COUNT(*) FROM salas WHERE cinema_id = $1`

	return consultarPaginado(ctx, r.pool, sqlPagina, sqlTotal, []any{cinemaID}, req,
		func(scan func(...any) error) (catalogo.Sala, error) {
			var s catalogo.Sala
			var tipo string
			if err := scan(&s.ID, &s.CinemaID, &s.Numero, &tipo, &s.CapacidadeTotal); err != nil {
				return s, fmt.Errorf("lendo sala: %w", err)
			}
			s.TipoTela = catalogo.TipoTela(tipo)
			return s, nil
		})
}
