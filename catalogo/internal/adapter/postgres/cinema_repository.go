package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type CinemaRepository struct{ pool *pgxpool.Pool }

func NovoCinemaRepository(p *pgxpool.Pool) *CinemaRepository { return &CinemaRepository{pool: p} }

const colunasCinema = `id, nome, cidade, estado, endereco, ativo`

func lerCinema(scan func(...any) error) (catalogo.Cinema, error) {
	var c catalogo.Cinema
	if err := scan(&c.ID, &c.Nome, &c.Cidade, &c.Estado, &c.Endereco, &c.Ativo); err != nil {
		return c, fmt.Errorf("lendo cinema: %w", err)
	}
	return c, nil
}

func (r *CinemaRepository) Listar(
	ctx context.Context,
	filtro usecase.FiltroCinemas,
	req shared.PageRequest,
) (shared.Page[catalogo.Cinema], error) {
	// Filtro nulo significa "qualquer situação": um só SQL atende os dois casos.
	filtros := []any{filtro.Ativo}

	const sqlPagina = `SELECT ` + colunasCinema + ` FROM cinemas
	                   WHERE ($1::boolean IS NULL OR ativo = $1)
	                   ORDER BY nome, id LIMIT $2 OFFSET $3`
	const sqlTotal = `SELECT COUNT(*) FROM cinemas WHERE ($1::boolean IS NULL OR ativo = $1)`

	return consultarPaginado(ctx, r.pool, sqlPagina, sqlTotal, filtros, req, lerCinema)
}

func (r *CinemaRepository) BuscarPorID(ctx context.Context, cinemaID string) (catalogo.Cinema, error) {
	linha := r.pool.QueryRow(ctx, `SELECT `+colunasCinema+` FROM cinemas WHERE id = $1`, cinemaID)
	c, err := lerCinema(linha.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalogo.Cinema{}, shared.NaoEncontrado("cinema", cinemaID)
	}
	if err != nil {
		return catalogo.Cinema{}, err
	}
	return c, nil
}

func (r *CinemaRepository) Criar(ctx context.Context, c catalogo.Cinema) error {
	const sqlInserir = `INSERT INTO cinemas (id, nome, cidade, estado, endereco, ativo)
	                    VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, sqlInserir, c.ID, c.Nome, c.Cidade, c.Estado, c.Endereco, c.Ativo)
	if err != nil {
		return fmt.Errorf("inserindo cinema: %w", err)
	}
	return nil
}

func (r *CinemaRepository) Atualizar(ctx context.Context, c catalogo.Cinema) error {
	const sqlAtualizar = `UPDATE cinemas SET nome = $2, cidade = $3, estado = $4,
	                          endereco = $5, ativo = $6, atualizado_em = CURRENT_TIMESTAMP
	                      WHERE id = $1`
	etiqueta, err := r.pool.Exec(ctx, sqlAtualizar, c.ID, c.Nome, c.Cidade, c.Estado, c.Endereco, c.Ativo)
	if err != nil {
		return fmt.Errorf("atualizando cinema: %w", err)
	}
	if etiqueta.RowsAffected() == 0 {
		return shared.NaoEncontrado("cinema", c.ID)
	}
	return nil
}

func (r *CinemaRepository) Desativar(ctx context.Context, cinemaID string) error {
	const sqlDesativar = `UPDATE cinemas SET ativo = FALSE, atualizado_em = CURRENT_TIMESTAMP
	                      WHERE id = $1`
	etiqueta, err := r.pool.Exec(ctx, sqlDesativar, cinemaID)
	if err != nil {
		return fmt.Errorf("desativando cinema: %w", err)
	}
	if etiqueta.RowsAffected() == 0 {
		return shared.NaoEncontrado("cinema", cinemaID)
	}
	return nil
}

func (r *CinemaRepository) Existe(ctx context.Context, cinemaID string) (bool, error) {
	var existe bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cinemas WHERE id = $1)`, cinemaID).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("verificando cinema: %w", err)
	}
	return existe, nil
}
