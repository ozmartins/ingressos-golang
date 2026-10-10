package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func TestErroDeEscritaDeSala(t *testing.T) {
	casos := []struct {
		nome         string
		err          error
		querConflito bool
	}{
		{"unicidade do número da sala ativa",
			&pgconn.PgError{Code: "23505", ConstraintName: indiceNumeroDaSalaAtiva}, true},
		{"unicidade embrulhada por camadas acima",
			fmt.Errorf("gorm: %w", &pgconn.PgError{Code: "23505", ConstraintName: indiceNumeroDaSalaAtiva}), true},
		{"unicidade de outra restrição (chave primária)",
			&pgconn.PgError{Code: "23505", ConstraintName: "salas_pkey"}, false},
		{"outro código do Postgres",
			&pgconn.PgError{Code: "08006", ConstraintName: indiceNumeroDaSalaAtiva}, false},
		{"erro de infraestrutura qualquer", errors.New("conexão recusada"), false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			err := erroDeEscritaDeSala(c.err, "inserindo", 3)
			if got := errors.Is(err, shared.ErrConflito); got != c.querConflito {
				t.Fatalf("errors.Is(ErrConflito) = %v, quer %v (err=%v)", got, c.querConflito, err)
			}
			if !c.querConflito && !errors.Is(err, c.err) {
				t.Fatalf("o erro original deveria seguir embrulhado: %v", err)
			}
		})
	}
}
