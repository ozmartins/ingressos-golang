// Package postgres é o adaptador de persistência. SQL à mão, para manter o
// ON CONFLICT e o UPDATE condicionado visíveis no código (research.md D2, D4).
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Schema onde vivem as tabelas da notificação. O serviço é dono dele e não usa
// `public`.
const Schema = "notificacao"

// Conectar abre o pool e confirma que o banco responde. Falhar aqui é falhar na
// largada, que é o que se quer.
func Conectar(ctx context.Context, url string) (*pgxpool.Pool, error) {
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

	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("abrir pool: %w", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("alcançar o banco: %w", err)
	}
	return p, nil
}
