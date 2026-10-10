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

// Classes e códigos SQLSTATE que dizem "o servidor não está em condição de
// atender", e não "sua consulta está errada": 08 (falha de conexão), 53300
// (conexões esgotadas), 57P01..57P03 (desligamento e inicialização).
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

// marcarIndisponibilidade etiqueta com shared.ErrBancoIndisponivel o erro que é
// falha de infraestrutura, preservando a causa original na cadeia. Erro de
// consulta, de dado ou de "não encontrado" passa intacto.
func marcarIndisponibilidade(err error) error {
	if err == nil || errors.Is(err, shared.ErrBancoIndisponivel) || !ehFalhaDeInfraestrutura(err) {
		return err
	}
	return fmt.Errorf("%w: %w", shared.ErrBancoIndisponivel, err)
}

// registrarClassificacao aplica marcarIndisponibilidade a todo erro que o GORM
// produz, para que nenhum repositório precise lembrar de fazê-lo.
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
