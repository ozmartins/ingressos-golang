package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/oseias/ingressos-golang/catalogo/internal/platform/obs"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type CaixaDeSaida struct{ banco *Banco }

func NovaCaixaDeSaida(b *Banco) *CaixaDeSaida { return &CaixaDeSaida{banco: b} }

func enfileirarFato(tx *gorm.DB, fato usecase.FatoPendente) error {
	// Guarda o trace da requisição que gerou o fato: o publicador roda depois,
	// fora dela, e continua este mesmo trace.
	linha := outboxRow{
		MessageID: fato.MessageID, RoutingKey: fato.RoutingKey, Payload: fato.Payload,
		Traceparent: obs.Traceparent(tx.Statement.Context),
		RequestID:   obs.RequestID(tx.Statement.Context),
	}

	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_id"}},
		DoNothing: true,
	}).Create(&linha).Error
	if err != nil {
		return fmt.Errorf("enfileirando fato: %w", err)
	}
	return nil
}

type FatoNaCaixa struct {
	ID         int64
	MessageID  string
	RoutingKey string
	Payload    []byte

	Traceparent string
	RequestID   string
}

func (c *CaixaDeSaida) Drenar(ctx context.Context, limite int, publicar func(FatoNaCaixa) error) (int, error) {
	publicados := 0

	err := c.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		var pendentes []outboxRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("publicado_em IS NULL").
			Order("id").
			Limit(limite).
			Find(&pendentes).Error
		if err != nil {
			return fmt.Errorf("lendo a caixa de saída: %w", err)
		}

		for _, p := range pendentes {
			f := FatoNaCaixa{ID: p.ID, MessageID: p.MessageID, RoutingKey: p.RoutingKey, Payload: p.Payload, Traceparent: p.Traceparent, RequestID: p.RequestID}

			fato := tx.Model(&outboxRow{}).Where("id = ?", f.ID)
			if err := publicar(f); err != nil {
				if errTentativa := fato.Update("tentativas", gorm.Expr("tentativas + 1")).Error; errTentativa != nil {
					return fmt.Errorf("contando a tentativa: %w", errTentativa)
				}
				continue
			}
			if err := fato.Update("publicado_em", gorm.Expr("now()")).Error; err != nil {
				return fmt.Errorf("marcando o fato como publicado: %w", err)
			}
			publicados++
		}
		return nil
	})

	return publicados, err
}
