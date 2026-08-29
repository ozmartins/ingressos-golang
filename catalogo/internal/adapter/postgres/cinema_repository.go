package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type CinemaRepository struct{ pool *pgxpool.Pool }

func NovoCinemaRepository(p *pgxpool.Pool) *CinemaRepository { return &CinemaRepository{pool: p} }

func (r *CinemaRepository) Listar(ctx context.Context, req shared.PageRequest) (shared.Page[catalogo.Cinema], error) {
	const sqlPagina = `SELECT id, nome, cidade, estado, endereco FROM cinemas
	                   ORDER BY nome, id LIMIT $1 OFFSET $2`
	const sqlTotal = `SELECT COUNT(*) FROM cinemas`

	return consultarPaginado(ctx, r.pool, sqlPagina, sqlTotal, nil, req,
		func(scan func(...any) error) (catalogo.Cinema, error) {
			var c catalogo.Cinema
			if err := scan(&c.ID, &c.Nome, &c.Cidade, &c.Estado, &c.Endereco); err != nil {
				return c, fmt.Errorf("lendo cinema: %w", err)
			}
			return c, nil
		})
}

func (r *CinemaRepository) Existe(ctx context.Context, cinemaID string) (bool, error) {
	var existe bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cinemas WHERE id = $1)`, cinemaID).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("verificando cinema: %w", err)
	}
	return existe, nil
}
