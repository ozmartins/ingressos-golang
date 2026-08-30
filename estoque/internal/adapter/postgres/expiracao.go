package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

// chaveVarredura é o identificador do advisory lock que serializa a varredura
// entre instâncias. Valor arbitrário, mas fixo.
const chaveVarredura int64 = 8_201_477_301

// ExpirarVencidas invalida em lote as reservas cujo prazo venceu e devolve os
// identificadores afetados.
//
// É o caminho autoritativo da expiração: cobre inclusive as reservas que
// venceram enquanto o serviço estava fora do ar (FR-013, SC-008).
func (r *Reservas) ExpirarVencidas(ctx context.Context, agora time.Time, limite int) ([]string, error) {
	var ids []string

	err := r.banco.EmTransacao(ctx, func(tx pgx.Tx) error {
		// Só uma instância varre por vez. Sem espera: se outra está varrendo,
		// esta passada simplesmente não faz nada.
		var obtido bool
		if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, chaveVarredura).Scan(&obtido); err != nil {
			return indisponivel(err)
		}
		if !obtido {
			return nil
		}

		linhas, err := tx.Query(ctx, `
			UPDATE reservas
			   SET status = 'EXPIRADA', finalizado_em = $1
			 WHERE id IN (
			       SELECT id FROM reservas
			        WHERE status = 'PENDENTE' AND expira_em <= $1
			        ORDER BY expira_em
			        LIMIT $2
			        FOR UPDATE SKIP LOCKED
			 )
			RETURNING id`, agora, limite)
		if err != nil {
			return indisponivel(err)
		}
		for linhas.Next() {
			var id string
			if err := linhas.Scan(&id); err != nil {
				linhas.Close()
				return indisponivel(err)
			}
			ids = append(ids, id)
		}
		linhas.Close()
		if err := linhas.Err(); err != nil {
			return indisponivel(err)
		}
		if len(ids) == 0 {
			return nil
		}

		// As poltronas voltam ao estoque na mesma transação (FR-015).
		_, err = tx.Exec(ctx, `
			UPDATE poltronas
			   SET status = $2, atualizado_em = now()
			 WHERE id IN (SELECT poltrona_id FROM reserva_poltronas WHERE reserva_id = ANY($1))`,
			ids, string(poltrona.Livre))
		if err != nil {
			return indisponivel(err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return ids, nil
}

// ExpirarUma invalida uma reserva específica, se ainda pendente e vencida.
// É o gatilho pronto, disparado pela notificação de prazo do Redis.
func (r *Reservas) ExpirarUma(ctx context.Context, reservaID string, agora time.Time) (usecase.ResultadoTransicao, error) {
	resultado := usecase.TransicaoIgnoradaInexistente

	err := r.banco.EmTransacao(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE reservas
			   SET status = $3, finalizado_em = $2
			 WHERE id = $1 AND status = 'PENDENTE' AND expira_em <= $2`,
			reservaID, agora, string(reserva.Expirada))
		if err != nil {
			return indisponivel(err)
		}
		if tag.RowsAffected() == 0 {
			var existe bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM reservas WHERE id = $1)`, reservaID).Scan(&existe); err != nil {
				return indisponivel(err)
			}
			if existe {
				// Confirmada, cancelada, ou ainda dentro do prazo: nada a fazer.
				resultado = usecase.TransicaoIgnoradaEstadoFinal
			}
			return nil
		}

		if _, err := tx.Exec(ctx, `
			UPDATE poltronas
			   SET status = $2, atualizado_em = now()
			 WHERE id IN (SELECT poltrona_id FROM reserva_poltronas WHERE reserva_id = $1)`,
			reservaID, string(poltrona.Livre)); err != nil {
			return indisponivel(err)
		}
		resultado = usecase.TransicaoAplicada
		return nil
	})

	return resultado, err
}
