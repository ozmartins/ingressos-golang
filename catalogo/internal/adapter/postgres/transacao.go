package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A transação existe para as escritas que precisam ser indivisíveis de um fato
// na caixa de saída. Um `Rollback` depois do `Commit` é inofensivo, e é o que
// garante a desistência em qualquer retorno de erro ou pânico.
func emTransacao(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
