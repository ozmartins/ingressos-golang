package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Schema onde vivem as tabelas do pagamento. O serviço é dono dele e não usa
// `public`.
const Schema = "pagamento"

// Abrir monta o pool de conexões com o `search_path` já fixado no schema do
// serviço. O `search_path` é do serviço, não do operador: fixá-lo aqui garante
// que uma DATABASE_URL sem ele nunca faça as consultas caírem em `public` e
// encontrarem tabelas de outro serviço com o mesmo nome.
func Abrir(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL malformada: %w", err)
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = Schema

	return pgxpool.NewWithConfig(ctx, cfg)
}
