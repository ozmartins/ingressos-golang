package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repositorio struct{ db *gorm.DB }

func NovoRepositorio(db *gorm.DB) *Repositorio { return &Repositorio{db: db} }

var estadosAnunciaveis = []string{
	string(transacao.Pago), string(transacao.Recusado), string(transacao.Cancelado),
}

func (r *Repositorio) CriarSeAusente(ctx context.Context, t transacao.Transacao) (bool, transacao.Transacao, error) {
	linha := paraLinha(t)
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "reserva_id"}}, DoNothing: true}, clause.Returning{}).
		Create(&linha)
	if res.Error != nil {
		return false, transacao.Transacao{}, falhaInfra(res.Error)
	}
	if res.RowsAffected == 1 {
		return true, linha.paraDominio(), nil
	}

	atual, err := r.BuscarPorReserva(ctx, t.ReservaID)
	if err != nil {
		return false, transacao.Transacao{}, err
	}
	return false, atual, nil
}

func (r *Repositorio) BuscarPorReserva(ctx context.Context, reservaID string) (transacao.Transacao, error) {
	var linha transacaoRow
	err := r.db.WithContext(ctx).Where("reserva_id = ?", reservaID).Take(&linha).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return transacao.Transacao{}, usecase.ErrNaoEncontrada
	}
	if err != nil {
		return transacao.Transacao{}, falhaInfra(err)
	}
	return linha.paraDominio(), nil
}

func (r *Repositorio) Finalizar(ctx context.Context, t transacao.Transacao) error {
	res := r.db.WithContext(ctx).Model(&transacaoRow{}).
		Where("id = ? AND status = ?", t.ID, string(transacao.Processando)).
		Updates(map[string]any{
			"status":                   string(t.Status),
			"codigo_transacao_gateway": nuloSeVazio(t.CodigoTransacaoGateway),
			"motivo_falha":             nuloSeVazio(string(t.MotivoFalha)),
			"pago_em":                  t.PagoEm,
			"atualizado_em":            t.AtualizadoEm,
		})
	if res.Error != nil {
		return falhaInfra(res.Error)
	}
	if res.RowsAffected == 0 {
		return usecase.ErrJaFinalizada
	}
	return nil
}

func (r *Repositorio) MarcarAnunciado(ctx context.Context, id string, agora time.Time) error {
	res := r.db.WithContext(ctx).Model(&transacaoRow{}).
		Where("id = ? AND status IN ?", id, estadosAnunciaveis).
		Updates(map[string]any{"resultado_anunciado": true, "atualizado_em": agora})
	return falhaInfra(res.Error)
}

func (r *Repositorio) ReivindicarCobranca(ctx context.Context, id string, agora time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&transacaoRow{}).
		Where("id = ? AND status = ? AND cobranca_emitida = false", id, string(transacao.Processando)).
		Updates(map[string]any{"cobranca_emitida": true, "atualizado_em": agora})
	if res.Error != nil {
		return false, falhaInfra(res.Error)
	}
	return res.RowsAffected == 1, nil
}

func (r *Repositorio) LiberarCobranca(ctx context.Context, id string, agora time.Time) error {
	res := r.db.WithContext(ctx).Model(&transacaoRow{}).
		Where("id = ? AND status = ?", id, string(transacao.Processando)).
		Updates(map[string]any{"cobranca_emitida": false, "atualizado_em": agora})
	return falhaInfra(res.Error)
}

func (r *Repositorio) RegistrarEscolha(ctx context.Context, t transacao.Transacao) error {
	res := r.db.WithContext(ctx).Model(&transacaoRow{}).
		Where("id = ? AND status = ?", t.ID, string(transacao.AguardandoForma)).
		Updates(map[string]any{
			"forma_pagamento": nuloSeVazio(string(t.FormaPagamento)),
			"status":          string(t.Status),
			"motivo_falha":    nuloSeVazio(string(t.MotivoFalha)),
			"atualizado_em":   t.AtualizadoEm,
		})
	if res.Error != nil {
		return falhaInfra(res.Error)
	}
	if res.RowsAffected == 0 {
		return usecase.ErrJaFinalizada
	}
	return nil
}

func (r *Repositorio) AguardandoCobranca(ctx context.Context, limite int) ([]transacao.Transacao, error) {
	var linhas []transacaoRow
	err := r.db.WithContext(ctx).
		Where("status = ? AND NOT cobranca_emitida", string(transacao.Processando)).
		Order("criado_em").Limit(limite).
		Find(&linhas).Error
	return paraDominioLista(linhas), falhaInfra(err)
}

func (r *Repositorio) CancelarEsperasVencidas(ctx context.Context, agora time.Time, limite int) ([]transacao.Transacao, error) {
	vencidas := r.db.WithContext(ctx).Model(&transacaoRow{}).Select("id").
		Where("status = ? AND expira_em <= ?", string(transacao.AguardandoForma), agora).
		Order("expira_em").Limit(limite).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})

	var linhas []transacaoRow
	err := r.db.WithContext(ctx).Model(&linhas).
		Clauses(clause.Returning{}).
		Where("id IN (?)", vencidas).
		Updates(map[string]any{
			"status":        string(transacao.Cancelado),
			"motivo_falha":  string(transacao.MotivoReservaExpirada),
			"atualizado_em": agora,
		}).Error
	return paraDominioLista(linhas), falhaInfra(err)
}

func (r *Repositorio) AnunciosPendentes(ctx context.Context, limite int) ([]transacao.Transacao, error) {
	var linhas []transacaoRow
	err := r.db.WithContext(ctx).
		Where("status IN ? AND NOT resultado_anunciado", estadosAnunciaveis).
		Order("atualizado_em").Limit(limite).
		Find(&linhas).Error
	return paraDominioLista(linhas), falhaInfra(err)
}

func paraDominioLista(linhas []transacaoRow) []transacao.Transacao {
	var lista []transacao.Transacao
	for _, l := range linhas {
		lista = append(lista, l.paraDominio())
	}
	return lista
}
