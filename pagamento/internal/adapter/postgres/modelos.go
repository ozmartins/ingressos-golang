package postgres

import (
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

// O modelo abaixo existe só para o GORM mapear a tabela. Não carrega regra de
// negócio nem sai do pacote; o domínio continua sem tags.
//
// Forma, código do gateway e motivo da falha são ponteiros porque o banco
// distingue NULL de vazio (as invariantes forma_coerente_com_estado e
// forma_valida), enquanto o domínio usa string vazia para "ausente".
type transacaoRow struct {
	ID                     string     `gorm:"column:id;primaryKey"`
	ReservaID              string     `gorm:"column:reserva_id"`
	UsuarioID              string     `gorm:"column:usuario_id"`
	ValorTotal             string     `gorm:"column:valor_total"`
	FormaPagamento         *string    `gorm:"column:forma_pagamento"`
	Status                 string     `gorm:"column:status"`
	CodigoTransacaoGateway *string    `gorm:"column:codigo_transacao_gateway"`
	MotivoFalha            *string    `gorm:"column:motivo_falha"`
	CobrancaEmitida        bool       `gorm:"column:cobranca_emitida"`
	ResultadoAnunciado     bool       `gorm:"column:resultado_anunciado"`
	ExpiraEm               time.Time  `gorm:"column:expira_em"`
	PagoEm                 *time.Time `gorm:"column:pago_em"`
	CriadoEm               time.Time  `gorm:"column:criado_em"`
	AtualizadoEm           time.Time  `gorm:"column:atualizado_em"`
}

func (transacaoRow) TableName() string { return "transacoes_pagamento" }

func nuloSeVazio(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func vazioSeNulo(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func paraLinha(t transacao.Transacao) transacaoRow {
	return transacaoRow{
		ID: t.ID, ReservaID: t.ReservaID, UsuarioID: t.UsuarioID, ValorTotal: t.ValorTotal,
		FormaPagamento:         nuloSeVazio(string(t.FormaPagamento)),
		Status:                 string(t.Status),
		CodigoTransacaoGateway: nuloSeVazio(t.CodigoTransacaoGateway),
		MotivoFalha:            nuloSeVazio(string(t.MotivoFalha)),
		CobrancaEmitida:        t.CobrancaEmitida, ResultadoAnunciado: t.ResultadoAnunciado,
		ExpiraEm: t.ExpiraEm, PagoEm: t.PagoEm, CriadoEm: t.CriadoEm, AtualizadoEm: t.AtualizadoEm,
	}
}

func (r transacaoRow) paraDominio() transacao.Transacao {
	return transacao.Transacao{
		ID: r.ID, ReservaID: r.ReservaID, UsuarioID: r.UsuarioID, ValorTotal: r.ValorTotal,
		FormaPagamento:         transacao.FormaPagamento(vazioSeNulo(r.FormaPagamento)),
		Status:                 transacao.Status(r.Status),
		CodigoTransacaoGateway: vazioSeNulo(r.CodigoTransacaoGateway),
		MotivoFalha:            transacao.Motivo(vazioSeNulo(r.MotivoFalha)),
		CobrancaEmitida:        r.CobrancaEmitida, ResultadoAnunciado: r.ResultadoAnunciado,
		ExpiraEm: r.ExpiraEm, PagoEm: r.PagoEm, CriadoEm: r.CriadoEm, AtualizadoEm: r.AtualizadoEm,
	}
}
