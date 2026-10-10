package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync/atomic"
	"testing"

	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// comportamento define o que o banco falso responde a qualquer comando: um
// erro, ou "nenhuma linha" nas consultas e `afetadas` linhas nas escritas.
type comportamento struct {
	err      error
	afetadas int64
}

var contadorDrivers atomic.Int64

// abrirBancoFalso devolve um *gorm.DB sobre um driver database/sql em memória,
// sem PostgreSQL, para exercitar a tradução de erros do repositório.
func abrirBancoFalso(t *testing.T, c comportamento) *gorm.DB {
	t.Helper()
	nome := "banco-falso-" + string(rune('a'+contadorDrivers.Add(1)))
	sql.Register(nome, driverFalso{c})
	conn, err := sql.Open(nome, "")
	if err != nil {
		t.Fatalf("abrir banco falso: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	db, err := gorm.Open(gormpg.New(gormpg.Config{Conn: conn}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Discard,
		DisableAutomaticPing:   true,
	})
	if err != nil {
		t.Fatalf("abrir gorm: %v", err)
	}
	return db
}

type driverFalso struct{ c comportamento }

func (d driverFalso) Open(string) (driver.Conn, error) { return conexaoFalsa(d), nil }

type conexaoFalsa struct{ c comportamento }

func (conexaoFalsa) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (conexaoFalsa) Close() error                        { return nil }
func (conexaoFalsa) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }

func (x conexaoFalsa) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if x.c.err != nil {
		return nil, x.c.err
	}
	return linhasVazias{}, nil
}

func (x conexaoFalsa) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	if x.c.err != nil {
		return nil, x.c.err
	}
	return driver.RowsAffected(x.c.afetadas), nil
}

type linhasVazias struct{}

func (linhasVazias) Columns() []string         { return []string{"id"} }
func (linhasVazias) Close() error              { return nil }
func (linhasVazias) Next([]driver.Value) error { return io.EOF }
