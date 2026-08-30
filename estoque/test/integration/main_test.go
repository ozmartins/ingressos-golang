//go:build integration

// Package integration exercita o serviço contra PostgreSQL, Redis e RabbitMQ
// reais. Os critérios que esta suíte cobre — concorrência, idempotência,
// expiração e recuperação — não são verificáveis com dublês: é exatamente onde
// o comportamento real do banco e do broker decide se o desenho funciona.
package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Ambiente guarda os endereços das dependências reais, compartilhadas por toda
// a suíte (subir contêiner por teste custaria minutos).
type Ambiente struct {
	DatabaseURL string
	RedisURL    string
	RabbitURL   string
}

var ambiente Ambiente

func TestMain(m *testing.M) {
	ctx := context.Background()

	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("estoque"),
		postgres.WithUsername("estoque"),
		postgres.WithPassword("estoque"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(90*time.Second)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres: %v\n", err)
		os.Exit(1)
	}

	rd, err := redis.Run(ctx, "redis:7-alpine",
		// A notificação de chave expirada é o gatilho pronto da liberação (D4).
		testcontainers.WithCmd("redis-server", "--notify-keyspace-events", "Ex"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "redis: %v\n", err)
		os.Exit(1)
	}

	rb, err := rabbitmq.Run(ctx, "rabbitmq:3.13-alpine")
	if err != nil {
		fmt.Fprintf(os.Stderr, "rabbitmq: %v\n", err)
		os.Exit(1)
	}

	ambiente.DatabaseURL, _ = pg.ConnectionString(ctx, "sslmode=disable")
	ambiente.RedisURL, _ = rd.ConnectionString(ctx)
	ambiente.RabbitURL, _ = rb.AmqpURL(ctx)

	if err := aplicarMigracoes(ctx, ambiente.DatabaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "migrações: %v\n", err)
		os.Exit(1)
	}

	codigo := m.Run()

	ctxFim, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	_ = pg.Terminate(ctxFim)
	_ = rd.Terminate(ctxFim)
	_ = rb.Terminate(ctxFim)

	os.Exit(codigo)
}

// aplicarMigracoes roda os arquivos de migrations/ na ordem. Usa o mesmo SQL que
// vai para produção: um teste contra esquema divergente não prova nada.
func aplicarMigracoes(ctx context.Context, url string) error {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	arquivos, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil {
		return err
	}
	if len(arquivos) == 0 {
		return fmt.Errorf("nenhuma migração encontrada")
	}
	for _, arquivo := range arquivos {
		sql, err := os.ReadFile(arquivo)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(arquivo), err)
		}
	}
	return nil
}
