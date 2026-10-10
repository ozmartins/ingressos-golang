package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

const chaveVarredura int64 = 8_201_477_301

func (r *Reservas) ExpirarVencidas(ctx context.Context, agora time.Time, limite int) ([]string, error) {
	var ids []string

	err := r.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		var obtido bool
		if err := tx.Raw(`SELECT pg_try_advisory_xact_lock(?)`, chaveVarredura).Scan(&obtido).Error; err != nil {
			return indisponivel(err)
		}
		if !obtido {
			return nil
		}

		vencidas := tx.Model(&reservaRow{}).Select("id").
			Where("status = ? AND expira_em <= ?", string(reserva.Pendente), agora).
			Order("expira_em").
			Limit(limite).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})

		var expiradas []reservaRow
		if err := tx.Model(&expiradas).
			Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).
			Where("id IN (?)", vencidas).
			Updates(map[string]any{"status": string(reserva.Expirada), "finalizado_em": agora}).Error; err != nil {
			return indisponivel(err)
		}
		if len(expiradas) == 0 {
			return nil
		}
		for _, e := range expiradas {
			ids = append(ids, e.ID)
		}

		return mudarPoltronasDas(tx, ids, poltrona.Livre)
	})

	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Reservas) ExpirarUma(ctx context.Context, reservaID string, agora time.Time) (usecase.ResultadoTransicao, error) {
	resultado := usecase.TransicaoIgnoradaInexistente

	err := r.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		res := tx.Model(&reservaRow{}).
			Where("id = ? AND status = ? AND expira_em <= ?", reservaID, string(reserva.Pendente), agora).
			Updates(map[string]any{"status": string(reserva.Expirada), "finalizado_em": agora})
		if res.Error != nil {
			return indisponivel(res.Error)
		}
		if res.RowsAffected == 0 {
			existe, err := reservaExiste(tx, reservaID)
			if err != nil {
				return err
			}
			if existe {
				resultado = usecase.TransicaoIgnoradaEstadoFinal
			}
			return nil
		}

		if err := mudarPoltronasDas(tx, []string{reservaID}, poltrona.Livre); err != nil {
			return err
		}
		resultado = usecase.TransicaoAplicada
		return nil
	})

	return resultado, err
}

func (r *Reservas) CancelarPendentesDaSessao(
	ctx context.Context,
	fila, messageID, sessaoID string,
	agora time.Time,
) (usecase.DesfechoCancelamentoDeSessao, error) {
	desfecho := usecase.DesfechoCancelamentoDeSessao{Resultado: usecase.TransicaoIgnoradaInexistente}

	err := r.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		if messageID != "" {
			novo, err := registrarProcessada(tx, fila, messageID)
			if err != nil {
				return err
			}
			if !novo {
				desfecho.Resultado = usecase.TransicaoIgnoradaDuplicata
				return nil
			}
		}

		var confirmadas int64
		if err := tx.Model(&reservaRow{}).
			Where("sessao_id = ? AND status = ?", sessaoID, string(reserva.Confirmada)).
			Count(&confirmadas).Error; err != nil {
			return indisponivel(err)
		}
		desfecho.Confirmadas = int(confirmadas)

		pendentes := tx.Model(&reservaRow{}).Select("id").
			Where("sessao_id = ? AND status = ?", sessaoID, string(reserva.Pendente)).
			Order("criado_em").
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})

		var canceladas []reservaRow
		if err := tx.Model(&canceladas).
			Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).
			Where("id IN (?)", pendentes).
			Updates(map[string]any{"status": string(reserva.Cancelada), "finalizado_em": agora}).Error; err != nil {
			return indisponivel(err)
		}
		if len(canceladas) == 0 {
			return nil
		}
		for _, c := range canceladas {
			desfecho.Canceladas = append(desfecho.Canceladas, c.ID)
		}

		if err := mudarPoltronasDas(tx, desfecho.Canceladas, poltrona.Livre); err != nil {
			return err
		}

		desfecho.Resultado = usecase.TransicaoAplicada
		return nil
	})

	return desfecho, err
}
