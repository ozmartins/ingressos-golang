package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func enfileirarFato(ctx context.Context, tx pgx.Tx, fato usecase.FatoPendente) error {
	var traceJSON []byte
	if len(fato.TraceContext) > 0 {
		var err error
		traceJSON, err = json.Marshal(fato.TraceContext)
		if err != nil {
			return err
		}
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_eventos (message_id, routing_key, payload, trace_context)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id) DO NOTHING`,
		fato.MessageID, fato.RoutingKey, fato.Payload, traceJSON)
	if err != nil {
		return indisponivel(err)
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

func (b *Banco) PendentesParaPublicar(ctx context.Context, limite int, fn func(FatoNaCaixa) error) (int, error) {
	publicados := 0

	err := b.EmTransacao(ctx, func(tx pgx.Tx) error {
		linhas, err := tx.Query(ctx, `
			SELECT id, message_id, routing_key, payload, trace_context
			  FROM outbox_eventos
			 WHERE publicado_em IS NULL
			 ORDER BY id
			 LIMIT $1
			   FOR UPDATE SKIP LOCKED`, limite)
		if err != nil {
			return indisponivel(err)
		}

		var lote []FatoNaCaixa
		for linhas.Next() {
			var f FatoNaCaixa
			var traceJSON []byte
			if err := linhas.Scan(&f.ID, &f.MessageID, &f.RoutingKey, &f.Payload, &traceJSON); err != nil {
				linhas.Close()
				return indisponivel(err)
			}
			if len(traceJSON) > 0 {
				_ = json.Unmarshal(traceJSON, &f.TraceContext)
			}
			lote = append(lote, f)
		}
		linhas.Close()
		if err := linhas.Err(); err != nil {
			return indisponivel(err)
		}

		for _, f := range lote {
			if err := fn(f); err != nil {
				if _, errTent := tx.Exec(ctx,
					`UPDATE outbox_eventos SET tentativas = tentativas + 1 WHERE id = $1`, f.ID); errTent != nil {
					return indisponivel(errTent)
				}
				continue
			}
			if _, err := tx.Exec(ctx,
				`UPDATE outbox_eventos SET publicado_em = now() WHERE id = $1`, f.ID); err != nil {
				return indisponivel(err)
			}
			publicados++
		}
		return nil
	})

	return publicados, err
}
