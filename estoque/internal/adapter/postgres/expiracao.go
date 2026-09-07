package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

const chaveVarredura int64 = 8_201_477_301

func (r *Reservas) ExpirarVencidas(ctx context.Context, agora time.Time, limite int) ([]string, error) {
	var ids []string

	err := r.banco.EmTransacao(ctx, func(tx pgx.Tx) error {
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

// CancelarPendentesDaSessao solta todas as reservas pendentes de uma sessão de
// uma vez. A forma é a de `ExpirarVencidas` — `UPDATE ... RETURNING` sobre um
// `SELECT ... FOR UPDATE SKIP LOCKED` —, com o filtro trocado de prazo vencido
// para sessão, e sem limite: uma sessão cancelada solta tudo, não um lote.
//
// As confirmadas não são tocadas: são ingressos pagos. Elas só são contadas,
// para que o caso de uso possa registrar que existem.
func (r *Reservas) CancelarPendentesDaSessao(
	ctx context.Context,
	fila, messageID, sessaoID string,
	agora time.Time,
) (usecase.DesfechoCancelamentoDeSessao, error) {
	desfecho := usecase.DesfechoCancelamentoDeSessao{Resultado: usecase.TransicaoIgnoradaInexistente}

	err := r.banco.EmTransacao(ctx, func(tx pgx.Tx) error {
		if messageID != "" {
			novo, err := registrarProcessada(ctx, tx, fila, messageID)
			if err != nil {
				return err
			}
			if !novo {
				desfecho.Resultado = usecase.TransicaoIgnoradaDuplicata
				return nil
			}
		}

		// Contado antes do UPDATE: depois dele as pendentes viraram canceladas,
		// e a contagem de confirmadas não mudaria — mas ler antes deixa claro
		// que o número é o do instante do cancelamento.
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM reservas
			 WHERE sessao_id = $1 AND status = 'CONFIRMADA'`, sessaoID).Scan(&desfecho.Confirmadas); err != nil {
			return indisponivel(err)
		}

		linhas, err := tx.Query(ctx, `
			UPDATE reservas
			   SET status = $2, finalizado_em = $3
			 WHERE id IN (
			       SELECT id FROM reservas
			        WHERE sessao_id = $1 AND status = 'PENDENTE'
			        ORDER BY criado_em
			        FOR UPDATE SKIP LOCKED
			 )
			RETURNING id`, sessaoID, string(reserva.Cancelada), agora)
		if err != nil {
			return indisponivel(err)
		}
		for linhas.Next() {
			var id string
			if err := linhas.Scan(&id); err != nil {
				linhas.Close()
				return indisponivel(err)
			}
			desfecho.Canceladas = append(desfecho.Canceladas, id)
		}
		linhas.Close()
		if err := linhas.Err(); err != nil {
			return indisponivel(err)
		}

		if len(desfecho.Canceladas) == 0 {
			return nil
		}

		if _, err := tx.Exec(ctx, `
			UPDATE poltronas
			   SET status = $2, atualizado_em = now()
			 WHERE id IN (
			       SELECT poltrona_id FROM reserva_poltronas
			        WHERE reserva_id = ANY($1))`,
			desfecho.Canceladas, string(poltrona.Livre)); err != nil {
			return indisponivel(err)
		}

		desfecho.Resultado = usecase.TransicaoAplicada
		return nil
	})

	return desfecho, err
}
