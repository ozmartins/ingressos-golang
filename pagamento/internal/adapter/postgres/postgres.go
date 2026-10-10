package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

const Schema = "pagamento"

type Banco struct {
	db  *gorm.DB
	sql *sql.DB
}

func Conectar(ctx context.Context, url string) (*Banco, error) {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL malformada: %w", err)
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = map[string]string{}
	}
	cfg.RuntimeParams["search_path"] = Schema

	conexoes := stdlib.OpenDB(*cfg)
	conexoes.SetMaxOpenConns(max(4, runtime.NumCPU()))

	db, err := gorm.Open(gormpg.New(gormpg.Config{Conn: conexoes}), &gorm.Config{
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
		return nil, fmt.Errorf("alcançar o banco: %w", err)
	}
	return &Banco{db: db, sql: conexoes}, nil
}

func (b *Banco) DB() *gorm.DB { return b.db }

func (b *Banco) SQL() *sql.DB { return b.sql }

func (b *Banco) Fechar() { _ = b.sql.Close() }

func (b *Banco) Verificar(ctx context.Context) error { return b.sql.PingContext(ctx) }

// falhaInfra marca como falha de infraestrutura qualquer erro do banco que não
// seja "não encontrado"; os casos de negócio são traduzidos antes de chegar aqui.
func falhaInfra(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", usecase.ErrDependenciaIndisponivel, err)
}
