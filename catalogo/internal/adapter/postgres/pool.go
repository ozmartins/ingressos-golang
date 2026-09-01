// Package postgres implementa os repositórios de leitura do catálogo.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Schema onde vivem as tabelas do catálogo. O serviço é dono dele e não usa
// `public`.
const Schema = "catalogo"

// NovoPool abre o pool e verifica a conectividade na inicialização: subir uma
// instância que não alcança o banco só adia a descoberta do problema.
func NovoPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL inválida: %w", err)
	}
	// O `search_path` é do serviço, não do operador: fixá-lo aqui garante que
	// uma DATABASE_URL sem ele nunca faça as consultas caírem em `public` e
	// encontrarem tabelas de outro serviço com o mesmo nome.
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = Schema
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("criando pool: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, fmt.Errorf("banco inacessível: %w", err)
	}
	return pool, nil
}
