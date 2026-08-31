// Package postgres é o adaptador de persistência. SQL à mão, para manter o
// ON CONFLICT e o UPDATE condicionado visíveis no código (research.md D2, D4).
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Conectar abre o pool e confirma que o banco responde. Falhar aqui é falhar na
// largada, que é o que se quer.
func Conectar(ctx context.Context, url string) (*pgxpool.Pool, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("abrir pool: %w", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("alcançar o banco: %w", err)
	}
	return p, nil
}
