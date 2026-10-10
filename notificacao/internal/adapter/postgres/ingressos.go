package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

type Ingressos struct{ DB *gorm.DB }

func (r Ingressos) CriarSeAusente(ctx context.Context, i ingresso.Ingresso) (bool, ingresso.Ingresso, error) {
	linha := paraLinha(i)

	res := r.DB.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "reserva_id"}}, DoNothing: true}).
		Create(&linha)
	if res.Error != nil {
		return false, ingresso.Ingresso{}, fmt.Errorf("inserir ingresso: %w", res.Error)
	}
	if res.RowsAffected == 1 {
		return true, linha.paraDominio(), nil
	}

	atual, err := r.buscarPorReserva(ctx, i.ReservaID)
	if err != nil {
		return false, ingresso.Ingresso{}, err
	}
	return false, atual, nil
}

func (r Ingressos) Utilizar(ctx context.Context, id string, agora time.Time) (bool, error) {
	res := r.DB.WithContext(ctx).
		Model(&ingressoRow{}).
		Where("id = ? AND status = ?", id, string(ingresso.Valido)).
		Updates(map[string]any{"status": string(ingresso.Utilizado), "utilizado_em": agora})
	if res.Error != nil {
		return false, fmt.Errorf("dar baixa no ingresso: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

func (r Ingressos) BuscarPorID(ctx context.Context, id string) (ingresso.Ingresso, error) {
	var linha ingressoRow
	err := r.DB.WithContext(ctx).Where("id = ?", id).Take(&linha).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ingresso.Ingresso{}, usecase.ErrNaoEncontrado
	}
	if err != nil {
		return ingresso.Ingresso{}, fmt.Errorf("buscar ingresso: %w", err)
	}
	return linha.paraDominio(), nil
}

func (r Ingressos) ListarPorUsuario(ctx context.Context, usuarioID string, filtro ingresso.Status) ([]ingresso.Ingresso, error) {
	consulta := r.DB.WithContext(ctx).Where("usuario_id = ?", usuarioID)
	if filtro != "" {
		consulta = consulta.Where("status = ?", string(filtro))
	}

	var linhas []ingressoRow
	if err := consulta.Order("criado_em DESC, id DESC").Find(&linhas).Error; err != nil {
		return nil, fmt.Errorf("listar ingressos: %w", err)
	}

	lista := make([]ingresso.Ingresso, 0, len(linhas))
	for _, l := range linhas {
		lista = append(lista, l.paraDominio())
	}
	return lista, nil
}

func (r Ingressos) buscarPorReserva(ctx context.Context, reservaID string) (ingresso.Ingresso, error) {
	var linha ingressoRow
	err := r.DB.WithContext(ctx).Where("reserva_id = ?", reservaID).Take(&linha).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ingresso.Ingresso{}, usecase.ErrNaoEncontrado
	}
	if err != nil {
		return ingresso.Ingresso{}, fmt.Errorf("buscar ingresso por reserva: %w", err)
	}
	return linha.paraDominio(), nil
}
