package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func codigoDeIndisponibilidade(codigo string) bool {
	return strings.HasPrefix(codigo, "08") ||
		codigo == "53300" ||
		codigo == "57P01" || codigo == "57P02" || codigo == "57P03"
}

func ehFalhaDeInfraestrutura(err error) bool {
	var (
		pgErr   *pgconn.PgError
		conexao *pgconn.ConnectError
		rede    net.Error
	)
	switch {
	case err == nil:
		return false
	case errors.As(err, &pgErr):
		return codigoDeIndisponibilidade(pgErr.Code)
	case errors.As(err, &conexao), errors.As(err, &rede):
		return true
	}
	return errors.Is(err, driver.ErrBadConn) ||
		errors.Is(err, sql.ErrConnDone) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, context.DeadlineExceeded)
}

func marcarIndisponibilidade(err error) error {
	if err == nil || errors.Is(err, shared.ErrBancoIndisponivel) || !ehFalhaDeInfraestrutura(err) {
		return err
	}
	return fmt.Errorf("%w: %w", shared.ErrBancoIndisponivel, err)
}

func registrarClassificacao(db *gorm.DB) error {
	marcar := func(tx *gorm.DB) { tx.Error = marcarIndisponibilidade(tx.Error) }
	cb := db.Callback()
	for nome, registrar := range map[string]func() error{
		"classificar:create": func() error { return cb.Create().After("gorm:create").Register("classificar:create", marcar) },
		"classificar:query":  func() error { return cb.Query().After("gorm:query").Register("classificar:query", marcar) },
		"classificar:update": func() error { return cb.Update().After("gorm:update").Register("classificar:update", marcar) },
		"classificar:delete": func() error { return cb.Delete().After("gorm:delete").Register("classificar:delete", marcar) },
		"classificar:row":    func() error { return cb.Row().After("gorm:row").Register("classificar:row", marcar) },
		"classificar:raw":    func() error { return cb.Raw().After("gorm:raw").Register("classificar:raw", marcar) },
	} {
		if err := registrar(); err != nil {
			return fmt.Errorf("registrando %s: %w", nome, err)
		}
	}
	return nil
}
