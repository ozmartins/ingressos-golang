package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
)

// Avisos implementa a porta de persistência do registro de notificação.
type Avisos struct{ Pool *pgxpool.Pool }

// Registrar grava a tentativa de aviso. Vários registros por ingresso são
// possíveis: é o que permitirá a uma feature futura de reenvio acrescentar
// tentativas sem apagar o histórico (data-model.md §3).
func (r Avisos) Registrar(ctx context.Context, reg aviso.Registro) error {
	const sql = `
		INSERT INTO registros_notificacao
		       (id, ingresso_id, usuario_id, canal, status, detalhes, enviado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	var detalhes *string
	if reg.Detalhes != "" {
		detalhes = &reg.Detalhes
	}
	_, err := r.Pool.Exec(ctx, sql,
		reg.ID, reg.IngressoID, reg.UsuarioID, string(reg.Canal), string(reg.Desfecho),
		detalhes, reg.EnviadoEm)
	if err != nil {
		return fmt.Errorf("registrar aviso: %w", err)
	}
	return nil
}
