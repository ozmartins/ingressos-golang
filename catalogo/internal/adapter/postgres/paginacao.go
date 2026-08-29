package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// consultarPaginado executa a consulta da página e a contagem do total.
//
// São duas idas ao banco, e isso é deliberado. A alternativa aparentemente mais
// barata — `COUNT(*) OVER ()` na mesma consulta — obriga o planejador a montar
// uma janela sobre todo o conjunto filtrado antes de ordenar, o que descarta o
// índice de ordenação e transforma toda listagem em varredura completa. Medido
// com EXPLAIN ANALYZE: com a janela, `Seq Scan + WindowAgg + Sort`; sem ela,
// `Index Scan` com LIMIT. A segunda consulta é uma contagem indexada barata; a
// primeira passa a ser proporcional ao tamanho da página, não ao do acervo.
//
// As duas consultas não compartilham transação: uma escrita entre elas pode
// fazer o total divergir da página em um registro. Para navegação de catálogo
// isso é irrelevante, e abrir transação em toda listagem custaria mais do que
// resolve.
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
