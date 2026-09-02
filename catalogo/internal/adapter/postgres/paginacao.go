package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func consultarPaginado[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	sqlPagina string,
	sqlTotal string,
	filtros []any,
	req shared.PageRequest,
	ler func(scan func(...any) error) (T, error),
) (shared.Page[T], error) {
	var vazia shared.Page[T]

	args := append(append([]any{}, filtros...), req.Limit(), req.Offset())
	rows, err := pool.Query(ctx, sqlPagina, args...)
	if err != nil {
		return vazia, fmt.Errorf("consultando página: %w", err)
	}
	defer rows.Close()

	var itens []T
	for rows.Next() {
		item, err := ler(rows.Scan)
		if err != nil {
			return vazia, err
		}
		itens = append(itens, item)
	}
	if err := rows.Err(); err != nil {
		return vazia, fmt.Errorf("iterando página: %w", err)
	}
	rows.Close()

	var total int
	if err := pool.QueryRow(ctx, sqlTotal, filtros...).Scan(&total); err != nil {
		return vazia, fmt.Errorf("contando registros: %w", err)
	}
	return shared.NovaPage(itens, total, req), nil
}
