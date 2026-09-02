package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

type Reservas struct{ banco *Banco }

func NovoRepositorioReservas(b *Banco) *Reservas { return &Reservas{banco: b} }

func (r *Reservas) Conceder(ctx context.Context, sol reserva.Solicitacao, res reserva.Reserva, fato usecase.FatoPendente) error {
	return r.banco.EmTransacao(ctx, func(tx pgx.Tx) error {
		linhas, err := tx.Query(ctx, `
			SELECT id, rotulo, status
			  FROM poltronas
			 WHERE sessao_id = $1 AND rotulo = ANY($2)
			 ORDER BY rotulo
			   FOR UPDATE NOWAIT`, sol.SessaoID, sol.Rotulos)
		if err != nil {
			if ehConflitoDeTravamento(err) {
				return fmt.Errorf("%w: disputa simultânea", shared.ErrPoltronasIndisponiveis)
			}
			return indisponivel(err)
		}

		type travada struct{ id, rotulo, status string }
		var encontradas []travada
		for linhas.Next() {
			var t travada
			if err := linhas.Scan(&t.id, &t.rotulo, &t.status); err != nil {
				linhas.Close()
				return indisponivel(err)
			}
			encontradas = append(encontradas, t)
		}
		linhas.Close()
		if err := linhas.Err(); err != nil {
			if ehConflitoDeTravamento(err) {
				return fmt.Errorf("%w: disputa simultânea", shared.ErrPoltronasIndisponiveis)
			}
			return indisponivel(err)
		}

		if len(encontradas) != len(sol.Rotulos) {
			if len(encontradas) == 0 {
				provisionada, err := sessaoProvisionada(ctx, tx, sol.SessaoID)
				if err != nil {
					return err
				}
				if !provisionada {
					return fmt.Errorf("%w: sessão %s", shared.ErrSessaoNaoProvisionada, sol.SessaoID)
				}
			}
			return fmt.Errorf("%w: um ou mais rótulos não existem na sessão %s", shared.ErrPoltronaInexistente, sol.SessaoID)
		}

		ids := make([]string, 0, len(encontradas))
		for _, t := range encontradas {
			if poltrona.Status(t.status) != poltrona.Livre {
				return fmt.Errorf("%w: poltrona %s está %s", shared.ErrPoltronasIndisponiveis, t.rotulo, t.status)
			}
			ids = append(ids, t.id)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO reservas (id, sessao_id, usuario_id, expira_em, status, criado_em)
			VALUES ($1, $2, $3, $4, 'PENDENTE', $5)`,
			res.ID, res.SessaoID, res.UsuarioID, res.ExpiraEm, res.CriadoEm); err != nil {
			return indisponivel(err)
		}

		for _, id := range ids {
			if _, err := tx.Exec(ctx,
				`INSERT INTO reserva_poltronas (reserva_id, poltrona_id) VALUES ($1, $2)`,
				res.ID, id); err != nil {
				return indisponivel(err)
			}
		}

		if _, err := tx.Exec(ctx, `
			UPDATE poltronas
			   SET status = 'RESERVADA', atualizado_em = now()
			 WHERE id = ANY($1)`, ids); err != nil {
			return indisponivel(err)
		}

		return enfileirarFato(ctx, tx, fato)
	})
}

func sessaoProvisionada(ctx context.Context, tx pgx.Tx, sessaoID string) (bool, error) {
	var existe bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM poltronas WHERE sessao_id = $1)`, sessaoID).Scan(&existe)
	if err != nil {
		return false, indisponivel(err)
	}
	return existe, nil
}

func (r *Reservas) aplicarDesfecho(
	ctx context.Context,
	fila, messageID, reservaID string,
	agora time.Time,
	novoStatusReserva reserva.Status,
	novoStatusPoltrona poltrona.Status,
) (usecase.ResultadoTransicao, error) {
	resultado := usecase.TransicaoIgnoradaInexistente

	err := r.banco.EmTransacao(ctx, func(tx pgx.Tx) error {
		if messageID != "" {
			novo, err := registrarProcessada(ctx, tx, fila, messageID)
			if err != nil {
				return err
			}
			if !novo {
				resultado = usecase.TransicaoIgnoradaDuplicata
				return nil
			}
		}

		tag, err := tx.Exec(ctx, `
			UPDATE reservas
			   SET status = $2, finalizado_em = $3
			 WHERE id = $1 AND status = 'PENDENTE'`, reservaID, string(novoStatusReserva), agora)
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
			} else {
				resultado = usecase.TransicaoIgnoradaInexistente
			}
			return nil
		}

		if _, err := tx.Exec(ctx, `
			UPDATE poltronas
			   SET status = $2, atualizado_em = now()
			 WHERE id IN (SELECT poltrona_id FROM reserva_poltronas WHERE reserva_id = $1)`,
			reservaID, string(novoStatusPoltrona)); err != nil {
			return indisponivel(err)
		}

		resultado = usecase.TransicaoAplicada
		return nil
	})

	return resultado, err
}

func (r *Reservas) Confirmar(ctx context.Context, fila, messageID, reservaID string, agora time.Time) (usecase.ResultadoTransicao, error) {
	return r.aplicarDesfecho(ctx, fila, messageID, reservaID, agora, reserva.Confirmada, poltrona.Ocupada)
}

func (r *Reservas) Cancelar(ctx context.Context, fila, messageID, reservaID string, agora time.Time) (usecase.ResultadoTransicao, error) {
	return r.aplicarDesfecho(ctx, fila, messageID, reservaID, agora, reserva.Cancelada, poltrona.Livre)
}
