package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

// A caixa de saída. Ela é escrita por quem grava o fato — dentro da transação
// daquele fato — e lida pelo publicador, que roda fora da requisição.
type CaixaDeSaida struct{ pool *pgxpool.Pool }

func NovaCaixaDeSaida(p *pgxpool.Pool) *CaixaDeSaida { return &CaixaDeSaida{pool: p} }

// Enfileirar recebe a transação de quem está gravando: é isso que torna o fato e
// o efeito que o produziu indivisíveis. O `DO NOTHING` cobre a retentativa de uma
// escrita que já tinha enfileirado o mesmo fato.
func enfileirarFato(ctx context.Context, tx pgx.Tx, fato usecase.FatoPendente) error {
	var traceJSON []byte
	if len(fato.TraceContext) > 0 {
		var err error
		if traceJSON, err = json.Marshal(fato.TraceContext); err != nil {
			return fmt.Errorf("serializando contexto de rastreamento: %w", err)
		}
	}

	const sqlInserir = `INSERT INTO outbox_eventos (message_id, routing_key, payload, trace_context)
	                    VALUES ($1, $2, $3, $4)
	                    ON CONFLICT (message_id) DO NOTHING`
	if _, err := tx.Exec(ctx, sqlInserir,
		fato.MessageID, fato.RoutingKey, fato.Payload, traceJSON); err != nil {
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

// Drenar lê um lote de pendentes e entrega cada um a `publicar`. O que sai é
// marcado; o que falha só tem a tentativa contada e volta no próximo tique — é
// daí que vem o "ao menos uma vez": a mesma mensagem pode ser reenviada se o
// broker a aceitou mas a marcação não chegou a ser gravada.
//
// `FOR UPDATE SKIP LOCKED` deixa duas réplicas drenarem a mesma caixa sem
// disputar as mesmas linhas.
func (c *CaixaDeSaida) Drenar(ctx context.Context, limite int, publicar func(FatoNaCaixa) error) (int, error) {
	publicados := 0

	err := emTransacao(ctx, c.pool, func(tx pgx.Tx) error {
		const sqlPendentes = `SELECT id, message_id, routing_key, payload, trace_context
		                        FROM outbox_eventos
		                       WHERE publicado_em IS NULL
		                       ORDER BY id
		                       LIMIT $1
		                         FOR UPDATE SKIP LOCKED`
		linhas, err := tx.Query(ctx, sqlPendentes, limite)
		if err != nil {
			return fmt.Errorf("lendo a caixa de saída: %w", err)
		}

		var lote []FatoNaCaixa
		for linhas.Next() {
			var (
				f         FatoNaCaixa
				traceJSON []byte
			)
			if err := linhas.Scan(&f.ID, &f.MessageID, &f.RoutingKey, &f.Payload, &traceJSON); err != nil {
				linhas.Close()
				return fmt.Errorf("lendo fato da caixa: %w", err)
			}
			if len(traceJSON) > 0 {
				_ = json.Unmarshal(traceJSON, &f.TraceContext)
			}
			lote = append(lote, f)
		}
		linhas.Close()
		if err := linhas.Err(); err != nil {
			return fmt.Errorf("lendo a caixa de saída: %w", err)
		}

		for _, f := range lote {
			if err := publicar(f); err != nil {
				const sqlTentativa = `UPDATE outbox_eventos SET tentativas = tentativas + 1 WHERE id = $1`
				if _, errTentativa := tx.Exec(ctx, sqlTentativa, f.ID); errTentativa != nil {
					return fmt.Errorf("contando a tentativa: %w", errTentativa)
				}
				continue
			}
			const sqlPublicado = `UPDATE outbox_eventos SET publicado_em = now() WHERE id = $1`
			if _, err := tx.Exec(ctx, sqlPublicado, f.ID); err != nil {
				return fmt.Errorf("marcando o fato como publicado: %w", err)
			}
			publicados++
		}
		return nil
	})

	return publicados, err
}
