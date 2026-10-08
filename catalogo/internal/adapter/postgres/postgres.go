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

// Banco é a conexão do serviço com o PostgreSQL. O GORM fica por cima de um
// `*sql.DB` aberto pelo pgx: a URL, o `search_path` e os limites do pool são os
// de sempre, e o que muda é só a forma de escrever as consultas.
type Banco struct {
	db  *gorm.DB
	sql *sql.DB
}

// Opcao ajusta a configuração do GORM. Existe para o teste de planos de consulta
// poder observar o SQL que o adaptador de fato emite.
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

	// O logger do GORM fica mudo: registrar SQL traria parâmetros para os logs
	// (constituição, princípio IV). Sem transação implícita por escrita: as que
	// precisam de atomicidade abrem a sua em `EmTransacao`.
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

// SQL entrega a conexão crua, para quem precisa de SQL que o GORM não escreve:
// os testes de integração preparam e inspecionam o banco por ela.
func (b *Banco) SQL() *sql.DB { return b.sql }

func (b *Banco) conn(ctx context.Context) *gorm.DB { return b.db.WithContext(ctx) }

// A transação existe para as escritas que precisam ser indivisíveis de um fato
// na caixa de saída. Retornar erro, ou entrar em pânico, desfaz tudo.
func (b *Banco) EmTransacao(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return b.conn(ctx).Transaction(fn)
}

// O carimbo de atualização é do banco, não do relógio do serviço — como sempre
// foi. Sem `UpdatedAt` automático do GORM: cada UPDATE o nomeia.
var gormAgora = gorm.Expr("CURRENT_TIMESTAMP")
