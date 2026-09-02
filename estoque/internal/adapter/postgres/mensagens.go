package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func registrarProcessada(ctx context.Context, tx pgx.Tx, fila, messageID string) (bool, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO mensagens_processadas (fila, message_id)
		VALUES ($1, $2)
		ON CONFLICT (fila, message_id) DO NOTHING`, fila, messageID)
	if err != nil {
		return false, indisponivel(err)
	}
	return tag.RowsAffected() == 1, nil
}

func (b *Banco) LimparMensagensProcessadas(ctx context.Context, retencao time.Duration) (int64, error) {
	tag, err := b.pool.Exec(ctx, `
		DELETE FROM mensagens_processadas
		WHERE processado_em < now() - $1::interval`, retencao.String())
	if err != nil {
		return 0, indisponivel(err)
	}
	return tag.RowsAffected(), nil
}
