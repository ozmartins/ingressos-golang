package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func registrarProcessada(tx *gorm.DB, fila, messageID string) (bool, error) {
	res := tx.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&mensagemProcessadaRow{Fila: fila, MessageID: messageID})
	if res.Error != nil {
		return false, indisponivel(res.Error)
	}
	return res.RowsAffected == 1, nil
}

func (b *Banco) LimparMensagensProcessadas(ctx context.Context, retencao time.Duration) (int64, error) {
	res := b.db.WithContext(ctx).
		Where("processado_em < now() - ?::interval", retencao.String()).
		Delete(&mensagemProcessadaRow{})
	if res.Error != nil {
		return 0, indisponivel(res.Error)
	}
	return res.RowsAffected, nil
}
