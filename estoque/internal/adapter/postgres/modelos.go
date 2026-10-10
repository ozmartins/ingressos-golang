package postgres

import (
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
)

type poltronaRow struct {
	ID           string     `gorm:"column:id;primaryKey"`
	SessaoID     string     `gorm:"column:sessao_id"`
	Fileira      string     `gorm:"column:fileira"`
	Numero       int        `gorm:"column:numero"`
	Rotulo       string     `gorm:"column:rotulo"`
	Tipo         string     `gorm:"column:tipo"`
	Status       string     `gorm:"column:status"`
	AtualizadoEm *time.Time `gorm:"column:atualizado_em;<-:update"`
}

func (poltronaRow) TableName() string { return "poltronas" }

func (r poltronaRow) paraDominio() poltrona.Poltrona {
	return poltrona.Poltrona{
		ID: r.ID, SessaoID: r.SessaoID, Fileira: r.Fileira, Numero: r.Numero,
		Rotulo: r.Rotulo, Tipo: poltrona.Tipo(r.Tipo), Status: poltrona.Status(r.Status),
	}
}

type reservaRow struct {
	ID           string     `gorm:"column:id;primaryKey"`
	SessaoID     string     `gorm:"column:sessao_id"`
	UsuarioID    string     `gorm:"column:usuario_id"`
	ExpiraEm     time.Time  `gorm:"column:expira_em"`
	Status       string     `gorm:"column:status"`
	CriadoEm     time.Time  `gorm:"column:criado_em"`
	FinalizadoEm *time.Time `gorm:"column:finalizado_em"`
	ValorTotal   *string    `gorm:"column:valor_total"`
}

func (reservaRow) TableName() string { return "reservas" }

type reservaPoltronaRow struct {
	ReservaID  string `gorm:"column:reserva_id;primaryKey"`
	PoltronaID string `gorm:"column:poltrona_id;primaryKey"`
}

func (reservaPoltronaRow) TableName() string { return "reserva_poltronas" }

type outboxRow struct {
	ID           int64      `gorm:"column:id;primaryKey;autoIncrement"`
	MessageID    string     `gorm:"column:message_id"`
	RoutingKey   string     `gorm:"column:routing_key"`
	Payload      []byte     `gorm:"column:payload;type:jsonb"`
	TraceContext []byte     `gorm:"column:trace_context;type:jsonb"`
	PublicadoEm  *time.Time `gorm:"column:publicado_em"`
	Tentativas   int        `gorm:"column:tentativas"`
}

func (outboxRow) TableName() string { return "outbox_eventos" }

type mensagemProcessadaRow struct {
	Fila      string `gorm:"column:fila;primaryKey"`
	MessageID string `gorm:"column:message_id;primaryKey"`
}

func (mensagemProcessadaRow) TableName() string { return "mensagens_processadas" }
