package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const Schema = "estoque"

type Banco struct {
	pool *pgxpool.Pool
}

func Abrir(ctx context.Context, url string) (*Banco, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL malformada: %w", err)
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = Schema
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("abrir pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("banco não respondeu: %w", err)
	}
	return &Banco{pool: pool}, nil
}

func (b *Banco) Pool() *pgxpool.Pool { return b.pool }

func (b *Banco) Fechar() { b.pool.Close() }

func (b *Banco) Verificar(ctx context.Context) error { return b.pool.Ping(ctx) }

func (b *Banco) EmTransacao(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return indisponivel(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return indisponivel(err)
	}
	return nil
}

func indisponivel(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", shared.ErrDependenciaIndisponivel, err.Error())
}

func ehConflitoDeTravamento(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "55P03"
	}
	return false
}

func ehViolacaoDeUnicidade(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
