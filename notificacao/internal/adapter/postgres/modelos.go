package postgres

import (
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

// Os modelos abaixo existem só para o GORM mapear as tabelas. Não carregam
// regra de negócio nem saem do pacote; o domínio continua sem tags.

type ingressoRow struct {
	ID          string     `gorm:"column:id;primaryKey"`
	ReservaID   string     `gorm:"column:reserva_id"`
	UsuarioID   string     `gorm:"column:usuario_id"`
	CodigoQR    string     `gorm:"column:codigo_qr"`
	Status      string     `gorm:"column:status"`
	UtilizadoEm *time.Time `gorm:"column:utilizado_em"`
	CriadoEm    time.Time  `gorm:"column:criado_em"`
}

func (ingressoRow) TableName() string { return "ingressos_emitidos" }

func paraLinha(i ingresso.Ingresso) ingressoRow {
	return ingressoRow{
		ID: i.ID, ReservaID: i.ReservaID, UsuarioID: i.UsuarioID, CodigoQR: i.CodigoQR,
		Status: string(i.Status), UtilizadoEm: i.UtilizadoEm, CriadoEm: i.CriadoEm,
	}
}

func (r ingressoRow) paraDominio() ingresso.Ingresso {
	return ingresso.Ingresso{
		ID: r.ID, ReservaID: r.ReservaID, UsuarioID: r.UsuarioID, CodigoQR: r.CodigoQR,
		Status: ingresso.Status(r.Status), UtilizadoEm: r.UtilizadoEm, CriadoEm: r.CriadoEm,
	}
}

type avisoRow struct {
	ID         string    `gorm:"column:id;primaryKey"`
	IngressoID string    `gorm:"column:ingresso_id"`
	UsuarioID  string    `gorm:"column:usuario_id"`
	Canal      string    `gorm:"column:canal"`
	Status     string    `gorm:"column:status"`
	Detalhes   *string   `gorm:"column:detalhes"`
	EnviadoEm  time.Time `gorm:"column:enviado_em"`
}

func (avisoRow) TableName() string { return "registros_notificacao" }

func avisoParaLinha(r aviso.Registro) avisoRow {
	var detalhes *string
	if r.Detalhes != "" {
		detalhes = &r.Detalhes
	}
	return avisoRow{
		ID: r.ID, IngressoID: r.IngressoID, UsuarioID: r.UsuarioID,
		Canal: string(r.Canal), Status: string(r.Desfecho), Detalhes: detalhes, EnviadoEm: r.EnviadoEm,
	}
}
