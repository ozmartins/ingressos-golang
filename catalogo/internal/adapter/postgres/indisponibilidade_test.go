package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func TestMarcarIndisponibilidade(t *testing.T) {
	recusada := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}

	casos := []struct {
		nome         string
		err          error
		indisponivel bool
	}{
		{"conexão recusada", recusada, true},
		{"conexão recusada embrulhada", fmt.Errorf("lendo filme: %w", recusada), true},
		{"conexão inválida do pool", driver.ErrBadConn, true},
		{"conexão já fechada", sql.ErrConnDone, true},
		{"conexão cortada no meio", io.ErrUnexpectedEOF, true},
		{"tempo excedido", context.DeadlineExceeded, true},
		{"falha de conexão do servidor (08006)", &pgconn.PgError{Code: "08006"}, true},
		{"conexões esgotadas (53300)", &pgconn.PgError{Code: "53300"}, true},
		{"desligamento administrativo (57P01)", &pgconn.PgError{Code: "57P01"}, true},
		{"servidor iniciando (57P03)", &pgconn.PgError{Code: "57P03"}, true},

		{"nil", nil, false},
		{"cliente desistiu", context.Canceled, false},
		{"tabela inexistente é defeito nosso (42P01)", &pgconn.PgError{Code: "42P01"}, false},
		{"violação de unicidade (23505)", &pgconn.PgError{Code: "23505"}, false},
		{"não encontrado", shared.NaoEncontrado("filme", "x"), false},
		{"erro qualquer", errors.New("falha"), false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			err := marcarIndisponibilidade(c.err)
			if got := errors.Is(err, shared.ErrBancoIndisponivel); got != c.indisponivel {
				t.Fatalf("errors.Is(ErrBancoIndisponivel) = %v, quer %v (err=%v)", got, c.indisponivel, err)
			}
			if c.err != nil && !errors.Is(err, c.err) {
				t.Fatalf("a causa original deveria seguir na cadeia: %v", err)
			}
		})
	}
}

func TestMarcarIndisponibilidadeEhIdempotente(t *testing.T) {
	uma := marcarIndisponibilidade(driver.ErrBadConn)
	duas := marcarIndisponibilidade(uma)
	if uma.Error() != duas.Error() {
		t.Fatalf("marcar duas vezes alterou o erro:\n %v\n %v", uma, duas)
	}
}
