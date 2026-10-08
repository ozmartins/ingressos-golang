package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

type Reservas struct{ banco *Banco }

func NovoRepositorioReservas(b *Banco) *Reservas { return &Reservas{banco: b} }

func (r *Reservas) Conceder(ctx context.Context, sol reserva.Solicitacao, res reserva.Reserva, fato usecase.FatoPendente) error {
	return r.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		var encontradas []poltronaRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
			Select("id", "rotulo", "status").
			Where("sessao_id = ? AND rotulo IN ?", sol.SessaoID, sol.Rotulos).
			Order("rotulo").
			Find(&encontradas).Error
		if err != nil {
			if ehConflitoDeTravamento(err) {
				return fmt.Errorf("%w: disputa simultânea", shared.ErrPoltronasIndisponiveis)
			}
			return indisponivel(err)
		}

		if len(encontradas) != len(sol.Rotulos) {
			if len(encontradas) == 0 {
				provisionada, err := sessaoProvisionada(tx, sol.SessaoID)
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
			if poltrona.Status(t.Status) != poltrona.Livre {
				return fmt.Errorf("%w: poltrona %s está %s", shared.ErrPoltronasIndisponiveis, t.Rotulo, t.Status)
			}
			ids = append(ids, t.ID)
		}

		if err := tx.Create(&reservaRow{
			ID: res.ID, SessaoID: res.SessaoID, UsuarioID: res.UsuarioID,
			ExpiraEm: res.ExpiraEm, Status: string(reserva.Pendente), CriadoEm: res.CriadoEm,
			ValorTotal: &res.ValorTotal,
		}).Error; err != nil {
			return indisponivel(err)
		}

		vinculos := make([]reservaPoltronaRow, 0, len(ids))
		for _, id := range ids {
			vinculos = append(vinculos, reservaPoltronaRow{ReservaID: res.ID, PoltronaID: id})
		}
		if err := tx.Create(&vinculos).Error; err != nil {
			return indisponivel(err)
		}

		if err := tx.Model(&poltronaRow{}).Where("id IN ?", ids).
			Updates(map[string]any{"status": string(poltrona.Reservada), "atualizado_em": gorm.Expr("now()")}).Error; err != nil {
			return indisponivel(err)
		}

		return enfileirarFato(tx, fato)
	})
}

func sessaoProvisionada(tx *gorm.DB, sessaoID string) (bool, error) {
	var existe bool
	err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM poltronas WHERE sessao_id = ?)`, sessaoID).Scan(&existe).Error
	if err != nil {
		return false, indisponivel(err)
	}
	return existe, nil
}

func reservaExiste(tx *gorm.DB, reservaID string) (bool, error) {
	var existe bool
	err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM reservas WHERE id = ?)`, reservaID).Scan(&existe).Error
	if err != nil {
		return false, indisponivel(err)
	}
	return existe, nil
}

// mudarPoltronasDas leva ao `novo` status todas as poltronas presas às reservas
// dadas. Quem chama já decidiu que a transição das reservas aconteceu.
func mudarPoltronasDas(tx *gorm.DB, reservaIDs []string, novo poltrona.Status) error {
	presas := tx.Model(&reservaPoltronaRow{}).Select("poltrona_id").Where("reserva_id IN ?", reservaIDs)
	err := tx.Model(&poltronaRow{}).Where("id IN (?)", presas).
		Updates(map[string]any{"status": string(novo), "atualizado_em": gorm.Expr("now()")}).Error
	if err != nil {
		return indisponivel(err)
	}
	return nil
}

func (r *Reservas) aplicarDesfecho(
	ctx context.Context,
	fila, messageID, reservaID string,
	agora time.Time,
	novoStatusReserva reserva.Status,
	novoStatusPoltrona poltrona.Status,
) (usecase.ResultadoTransicao, error) {
	resultado := usecase.TransicaoIgnoradaInexistente

	err := r.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		if messageID != "" {
			novo, err := registrarProcessada(tx, fila, messageID)
			if err != nil {
				return err
			}
			if !novo {
				resultado = usecase.TransicaoIgnoradaDuplicata
				return nil
			}
		}

		res := tx.Model(&reservaRow{}).
			Where("id = ? AND status = ?", reservaID, string(reserva.Pendente)).
			Updates(map[string]any{"status": string(novoStatusReserva), "finalizado_em": agora})
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
			} else {
				resultado = usecase.TransicaoIgnoradaInexistente
			}
			return nil
		}

		if err := mudarPoltronasDas(tx, []string{reservaID}, novoStatusPoltrona); err != nil {
			return err
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
