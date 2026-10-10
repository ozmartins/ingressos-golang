package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const Schema = "catalogo"

type Banco struct {
	db  *gorm.DB
	sql *sql.DB
}

type Opcao func(*gorm.Config)

func Abrir(ctx context.Context, databaseURL string, opcoes ...Opcao) (*Banco, error) {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL inválida: %w", err)
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = map[string]string{}
	}
	cfg.RuntimeParams["search_path"] = Schema

	sqlDB := stdlib.OpenDB(*cfg)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	gormCfg := &gorm.Config{
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
		Logger:                 logger.Discard,
	}
	for _, o := range opcoes {
		o(gormCfg)
	}

	db, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: sqlDB}), gormCfg)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("criando conexão: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctxPing); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("banco inacessível: %w", err)
	}
	return &Banco{db: db, sql: sqlDB}, nil
}

func (b *Banco) Fechar() { _ = b.sql.Close() }

func (b *Banco) Ping(ctx context.Context) error { return b.sql.PingContext(ctx) }

func (b *Banco) SQL() *sql.DB { return b.sql }

func (b *Banco) conn(ctx context.Context) *gorm.DB { return b.db.WithContext(ctx) }

func (b *Banco) EmTransacao(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return b.conn(ctx).Transaction(fn)
}

var gormAgora = gorm.Expr("CURRENT_TIMESTAMP")
