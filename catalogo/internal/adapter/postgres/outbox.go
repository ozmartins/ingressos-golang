package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type CaixaDeSaida struct{ banco *Banco }

func NovaCaixaDeSaida(b *Banco) *CaixaDeSaida { return &CaixaDeSaida{banco: b} }

func enfileirarFato(tx *gorm.DB, fato usecase.FatoPendente) error {
	linha := outboxRow{MessageID: fato.MessageID, RoutingKey: fato.RoutingKey, Payload: fato.Payload}
	if len(fato.TraceContext) > 0 {
		trace, err := json.Marshal(fato.TraceContext)
		if err != nil {
			return fmt.Errorf("serializando contexto de rastreamento: %w", err)
		}
		linha.TraceContext = trace
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
	ID           int64
	MessageID    string
	RoutingKey   string
	Payload      []byte
	TraceContext map[string]string
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
			f := FatoNaCaixa{ID: p.ID, MessageID: p.MessageID, RoutingKey: p.RoutingKey, Payload: p.Payload}
			if len(p.TraceContext) > 0 {
				if err := json.Unmarshal(p.TraceContext, &f.TraceContext); err != nil {
					slog.WarnContext(ctx, "contexto de rastreamento ilegível na caixa de saída; publicando sem ele",
						slog.String("message_id", p.MessageID), slog.Any("erro", err))
					f.TraceContext = nil
				}
			}

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
