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

type SalaRepository struct{ pool *pgxpool.Pool }

func NovoSalaRepository(p *pgxpool.Pool) *SalaRepository { return &SalaRepository{pool: p} }

const colunasSala = `id, cinema_id, numero, tipo_tela, capacidade_total, ativo`

func lerSala(scan func(...any) error) (catalogo.Sala, error) {
	var (
		s    catalogo.Sala
		tipo string
	)
	if err := scan(&s.ID, &s.CinemaID, &s.Numero, &tipo, &s.CapacidadeTotal, &s.Ativo); err != nil {
		return s, fmt.Errorf("lendo sala: %w", err)
	}
	s.TipoTela = catalogo.TipoTela(tipo)
	return s, nil
}

func (r *SalaRepository) ListarPorCinema(
	ctx context.Context,
	cinemaID string,
	filtro usecase.FiltroSalas,
	req shared.PageRequest,
) (shared.Page[catalogo.Sala], error) {
	// Filtro nulo significa "qualquer situação": um só SQL atende os dois casos.
	filtros := []any{cinemaID, filtro.Ativo}

	const sqlPagina = `SELECT ` + colunasSala + ` FROM salas
	                   WHERE cinema_id = $1 AND ($2::boolean IS NULL OR ativo = $2)
	                   ORDER BY numero, id LIMIT $3 OFFSET $4`
	const sqlTotal = `SELECT COUNT(*) FROM salas
	                  WHERE cinema_id = $1 AND ($2::boolean IS NULL OR ativo = $2)`

	return consultarPaginado(ctx, r.pool, sqlPagina, sqlTotal, filtros, req, lerSala)
}

func (r *SalaRepository) BuscarPorID(ctx context.Context, salaID string) (catalogo.Sala, error) {
	linha := r.pool.QueryRow(ctx, `SELECT `+colunasSala+` FROM salas WHERE id = $1`, salaID)
	s, err := lerSala(linha.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalogo.Sala{}, shared.NaoEncontrado("sala", salaID)
	}
	if err != nil {
		return catalogo.Sala{}, err
	}
	return s, nil
}

func (r *SalaRepository) Criar(ctx context.Context, s catalogo.Sala) error {
	const sqlInserir = `INSERT INTO salas (id, cinema_id, numero, tipo_tela, capacidade_total, ativo)
	                    VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, sqlInserir, s.ID, s.CinemaID, s.Numero, string(s.TipoTela), s.CapacidadeTotal, s.Ativo)
	if err != nil {
		return fmt.Errorf("inserindo sala: %w", err)
	}
	return nil
}

func (r *SalaRepository) Atualizar(ctx context.Context, s catalogo.Sala) error {
	const sqlAtualizar = `UPDATE salas SET numero = $2, tipo_tela = $3, capacidade_total = $4,
	                          ativo = $5, atualizado_em = CURRENT_TIMESTAMP
	                      WHERE id = $1`
	etiqueta, err := r.pool.Exec(ctx, sqlAtualizar, s.ID, s.Numero, string(s.TipoTela), s.CapacidadeTotal, s.Ativo)
	if err != nil {
		return fmt.Errorf("atualizando sala: %w", err)
	}
	if etiqueta.RowsAffected() == 0 {
		return shared.NaoEncontrado("sala", s.ID)
	}
	return nil
}

func (r *SalaRepository) Desativar(ctx context.Context, salaID string) error {
	const sqlDesativar = `UPDATE salas SET ativo = FALSE, atualizado_em = CURRENT_TIMESTAMP
	                      WHERE id = $1`
	etiqueta, err := r.pool.Exec(ctx, sqlDesativar, salaID)
	if err != nil {
		return fmt.Errorf("desativando sala: %w", err)
	}
	if etiqueta.RowsAffected() == 0 {
		return shared.NaoEncontrado("sala", salaID)
	}
	return nil
}

func (r *SalaRepository) NumeroEmUso(ctx context.Context, cinemaID string, numero int, excetoID string) (bool, error) {
	const sql = `SELECT EXISTS(SELECT 1 FROM salas
	             WHERE cinema_id = $1 AND numero = $2 AND ativo AND id <> $3)`
	var emUso bool
	if err := r.pool.QueryRow(ctx, sql, cinemaID, numero, excetoID).Scan(&emUso); err != nil {
		return false, fmt.Errorf("verificando número da sala: %w", err)
	}
	return emUso, nil
}
