//go:build integration

// Package integration exercita os adaptadores contra um PostgreSQL real.
//
// Mock de banco provaria apenas que o código chama o driver. O que precisa de
// prova aqui é o SQL: junções, ordenação, contagem na mesma consulta e o tipo
// decimal — nada disso sobrevive a um dublê.
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("catalogo"),
		postgres.WithUsername("catalogo"),
		postgres.WithPassword("catalogo"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "não foi possível subir o PostgreSQL de teste: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = testcontainers.TerminateContainer(container) }()

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "string de conexão: %v\n", err)
		os.Exit(1)
	}

	pool, err = pgadapter.NovoPool(ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := aplicarMigracoes(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "migrações: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func aplicarMigracoes(ctx context.Context) error {
	for _, arquivo := range []string{
		"../../migrations/000001_criar_esquema.up.sql",
		"../../migrations/000002_criar_indices.up.sql",
	} {
		sql, err := os.ReadFile(arquivo)
		if err != nil {
			return fmt.Errorf("lendo %s: %w", arquivo, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("aplicando %s: %w", arquivo, err)
		}
	}
	return nil
}

// carregarFixtures deixa o banco no estado conhecido antes de cada teste.
func carregarFixtures(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE sessoes, salas, cinemas, filmes CASCADE`); err != nil {
		t.Fatalf("limpando tabelas: %v", err)
	}
	sql, err := os.ReadFile("../fixtures/catalogo_exemplo.sql")
	if err != nil {
		t.Fatalf("lendo fixtures: %v", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("carregando fixtures: %v", err)
	}
}
