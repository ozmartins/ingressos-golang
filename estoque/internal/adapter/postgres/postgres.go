package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const Schema = "estoque"

type Banco struct {
	db  *gorm.DB
	sql *sql.DB
}

func Abrir(ctx context.Context, url string) (*Banco, error) {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL malformada: %w", err)
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = map[string]string{}
	}
	cfg.RuntimeParams["search_path"] = Schema

	conexoes := stdlib.OpenDB(*cfg)
	conexoes.SetConnMaxIdleTime(5 * time.Minute)

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: conexoes}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Discard,
		DisableAutomaticPing:   true,
	})
	if err != nil {
		_ = conexoes.Close()
		return nil, fmt.Errorf("abrir pool: %w", err)
	}
	if err := conexoes.PingContext(ctx); err != nil {
		_ = conexoes.Close()
		return nil, fmt.Errorf("banco não respondeu: %w", err)
	}
	return &Banco{db: db, sql: conexoes}, nil
}

func (b *Banco) SQL() *sql.DB { return b.sql }

func (b *Banco) Fechar() { _ = b.sql.Close() }

func (b *Banco) Verificar(ctx context.Context) error { return b.sql.PingContext(ctx) }

func (b *Banco) EmTransacao(ctx context.Context, fn func(tx *gorm.DB) error) error {
	tx := b.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return indisponivel(tx.Error)
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback().Error
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return indisponivel(err)
	}
	confirmada = true
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
