package postgres

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func enfileirarFato(tx *gorm.DB, fato usecase.FatoPendente) error {
	res := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "message_id"}}, DoNothing: true}).
		Create(&outboxRow{
			MessageID:  fato.MessageID,
			RoutingKey: fato.RoutingKey,
			Payload:    fato.Payload,
		})
	if res.Error != nil {
		return indisponivel(res.Error)
	}
	return nil
}

type FatoNaCaixa struct {
	ID         int64
	MessageID  string
	RoutingKey string
	Payload    []byte
}

func (b *Banco) PendentesParaPublicar(ctx context.Context, limite int, fn func(FatoNaCaixa) error) (int, error) {
	publicados := 0

	err := b.EmTransacao(ctx, func(tx *gorm.DB) error {
		var linhas []outboxRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Select("id", "message_id", "routing_key", "payload").
			Where("publicado_em IS NULL").
			Order("id").
			Limit(limite).
			Find(&linhas).Error; err != nil {
			return indisponivel(err)
		}

		for _, l := range linhas {
			f := FatoNaCaixa{ID: l.ID, MessageID: l.MessageID, RoutingKey: l.RoutingKey, Payload: l.Payload}
			caixa := tx.Model(&outboxRow{}).Where("id = ?", f.ID)
			if err := fn(f); err != nil {
				if errTent := caixa.Update("tentativas", gorm.Expr("tentativas + 1")).Error; errTent != nil {
					return indisponivel(errTent)
				}
				continue
			}
			if err := caixa.Update("publicado_em", gorm.Expr("now()")).Error; err != nil {
				return indisponivel(err)
			}
			publicados++
		}
		return nil
	})

	return publicados, err
}
