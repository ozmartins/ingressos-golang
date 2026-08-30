package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

// enfileirarFato grava o fato na caixa de saída, dentro da transação que o
// produziu. É isso que garante que o evento sobreviva a um broker fora do ar
// sem duplicar a reserva (FR-018, SC-005).
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

// FatoNaCaixa é uma linha pendente de publicação.
type FatoNaCaixa struct {
	ID           int64
	MessageID    string
	RoutingKey   string
	Payload      []byte
	TraceContext map[string]string
}

// PendentesParaPublicar reserva um lote de fatos ainda não publicados.
//
// SKIP LOCKED permite que mais de uma instância publique ao mesmo tempo sem
// republicar a mesma linha.
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
				// Falha de publicação: conta a tentativa e deixa a linha
				// pendente. O próximo ciclo tenta de novo (FR-018).
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
