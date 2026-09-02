//go:build integration

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

func TestSchemaProprio(t *testing.T) {
	ctx := context.Background()

	var searchPath string
	if err := pool.QueryRow(ctx, `SELECT current_setting('search_path')`).Scan(&searchPath); err != nil {
		t.Fatalf("consultando search_path: %v", err)
	}
	if searchPath != "catalogo" {
		t.Errorf("search_path = %q, esperado \"catalogo\"", searchPath)
	}

	for _, tabela := range []string{"filmes", "cinemas", "salas", "sessoes"} {
		var emPublic *string
		if err := pool.QueryRow(ctx, `SELECT to_regclass('public.'|| $1)::text`, tabela).Scan(&emPublic); err != nil {
			t.Fatalf("consultando public.%s: %v", tabela, err)
		}
		if emPublic != nil {
			t.Errorf("tabela %q existe em public (%s); deveria existir só em catalogo", tabela, *emPublic)
		}
	}
}
