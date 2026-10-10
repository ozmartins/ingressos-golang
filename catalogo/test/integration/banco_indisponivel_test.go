//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

// Sobe um PostgreSQL só para este teste e o derruba: o banco compartilhado do
// pacote não pode ser interrompido sem quebrar os demais testes.
func TestBancoForaDoArViraErroDeIndisponibilidade(t *testing.T) {
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
		t.Fatalf("subindo PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	efemero, err := pgadapter.Abrir(ctx, url)
	if err != nil {
		t.Fatalf("Abrir: %v", err)
	}
	t.Cleanup(efemero.Fechar)

	parada := 10 * time.Second
	if err := container.Stop(ctx, &parada); err != nil {
		t.Fatalf("parando PostgreSQL: %v", err)
	}

	t.Run("leitura fora de transação", func(t *testing.T) {
		_, err := pgadapter.NovoFilmeRepository(efemero).BuscarPorID(ctx, "c394c8b3-76a1-4328-b803-02f5923b7a15")
		if !errors.Is(err, shared.ErrBancoIndisponivel) {
			t.Fatalf("esperava ErrBancoIndisponivel, obteve: %v", err)
		}
		if errors.Is(err, shared.ErrNaoEncontrado) {
			t.Fatalf("banco fora do ar não pode parecer \"não encontrado\": %v", err)
		}
	})

	t.Run("escrita em transação", func(t *testing.T) {
		err := pgadapter.NovoSessaoRepository(efemero).Cancelar(ctx,
			"00000000-0000-4000-8000-000000000001", usecase.FatoPendente{MessageID: "m", RoutingKey: "sessao.cancelada"})
		if !errors.Is(err, shared.ErrBancoIndisponivel) {
			t.Fatalf("esperava ErrBancoIndisponivel, obteve: %v", err)
		}
	})

	t.Run("listagem paginada", func(t *testing.T) {
		_, err := pgadapter.NovoFilmeRepository(efemero).Listar(ctx, usecase.FiltroFilmes{},
			[]catalogo.StatusFilme{catalogo.StatusForaDeCartaz}, pagina(t, 1, 20))
		if !errors.Is(err, shared.ErrBancoIndisponivel) {
			t.Fatalf("esperava ErrBancoIndisponivel, obteve: %v", err)
		}
	})
}
