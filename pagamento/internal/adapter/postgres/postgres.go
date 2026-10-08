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
)

const Schema = "pagamento"

// Banco é a conexão do serviço com o PostgreSQL: o GORM por cima de um *sql.DB
// aberto pelo pgx, com o search_path fixado no schema do serviço.
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
	// Mesmo teto do pool anterior: max(4, CPUs).
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

// SQL expõe o *sql.DB subjacente, para os testes de integração consultarem o
// banco sem passar pelo GORM.
func (b *Banco) SQL() *sql.DB { return b.sql }

func (b *Banco) Fechar() { _ = b.sql.Close() }

func (b *Banco) Verificar(ctx context.Context) error { return b.sql.PingContext(ctx) }
