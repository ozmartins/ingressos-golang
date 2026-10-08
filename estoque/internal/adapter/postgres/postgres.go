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

// Abrir monta a conexão a partir do pgx — que aceita URL e DSN chave=valor e
// deixa fixar o `search_path` — e a entrega ao GORM. O driver por baixo continua
// sendo o pgx, então os erros seguem sendo *pgconn.PgError.
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
		// Toda escrita de mais de um passo já é explícita em EmTransacao; a
		// transação implícita do GORM só custaria uma ida e volta a mais.
		SkipDefaultTransaction: true,
		// O logger do GORM imprimiria SQL e parâmetros.
		Logger:               logger.Discard,
		DisableAutomaticPing: true,
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

// SQL expõe o pool de conexões, para quem precisa conferir o estado do banco
// por fora do GORM (os testes de integração).
func (b *Banco) SQL() *sql.DB { return b.sql }

func (b *Banco) Fechar() { _ = b.sql.Close() }

func (b *Banco) Verificar(ctx context.Context) error { return b.sql.PingContext(ctx) }

// EmTransacao roda fn numa transação. Erro de fn volta como veio (são erros de
// domínio); falha ao abrir ou confirmar vira indisponibilidade.
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
