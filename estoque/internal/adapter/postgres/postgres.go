// Package postgres implementa os repositórios sobre PostgreSQL. É aqui que mora
// a exclusividade sobre poltronas: transação com SELECT ... FOR UPDATE, na mesma
// unidade atômica que grava reserva, vínculos e caixa de saída (research D2).
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// Schema onde vivem as tabelas do estoque. O serviço é dono dele e não usa
// `public`.
const Schema = "estoque"

// Banco é o pool de conexões e o ponto de entrada dos repositórios.
type Banco struct {
	pool *pgxpool.Pool
}

// Abrir cria o pool e verifica que o banco responde. Falhar aqui é falhar na
// largada, que é o que a constituição pede.
func Abrir(ctx context.Context, url string) (*Banco, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL malformada: %w", err)
	}
	// O `search_path` é do serviço, não do operador: fixá-lo aqui garante que
	// uma DATABASE_URL sem ele nunca faça as consultas caírem em `public` e
	// encontrarem tabelas de outro serviço com o mesmo nome.
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = Schema
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("abrir pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("banco não respondeu: %w", err)
	}
	return &Banco{pool: pool}, nil
}

// Pool expõe o pool para os testes de integração e para a verificação de saúde.
func (b *Banco) Pool() *pgxpool.Pool { return b.pool }

// Fechar encerra o pool.
func (b *Banco) Fechar() { b.pool.Close() }

// Verificar responde se o banco está acessível (usado pela prontidão).
func (b *Banco) Verificar(ctx context.Context) error { return b.pool.Ping(ctx) }

// EmTransacao executa fn dentro de uma transação, confirmando no sucesso e
// desfazendo em qualquer erro ou pânico.
func (b *Banco) EmTransacao(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return indisponivel(err)
	}
	defer func() {
		// Rollback em transação já confirmada é inofensivo.
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return indisponivel(err)
	}
	return nil
}

// indisponivel traduz falha de infraestrutura para o erro de domínio, para que
// o adaptador gRPC responda UNAVAILABLE e nunca conceda sem garantia (FR-006).
func indisponivel(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", shared.ErrDependenciaIndisponivel, err.Error())
}

// ehConflitoDeTravamento reconhece o erro de "linha travada por outra transação"
// devolvido pelo FOR UPDATE NOWAIT. Preferimos recusar rápido a esperar: o
// orçamento da operação é de 100 ms (SC-001).
func ehConflitoDeTravamento(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 55P03 = lock_not_available
		return pgErr.Code == "55P03"
	}
	return false
}

// ehViolacaoDeUnicidade reconhece conflito de chave única (23505).
func ehViolacaoDeUnicidade(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
