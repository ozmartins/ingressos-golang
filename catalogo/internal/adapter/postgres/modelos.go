package postgres

import "time"

type filmeRow struct {
	ID                  string  `gorm:"column:id;primaryKey"`
	Titulo              string  `gorm:"column:titulo"`
	Sinopse             *string `gorm:"column:sinopse"`
	DuracaoMinutos      int     `gorm:"column:duracao_minutos"`
	ClassificacaoEtaria string  `gorm:"column:classificacao_etaria"`
	Genero              string  `gorm:"column:genero"`
	ImagemURL           *string `gorm:"column:imagem_url"`
	Status              string  `gorm:"column:status"`
}

func (filmeRow) TableName() string { return "filmes" }

type cinemaRow struct {
	ID       string `gorm:"column:id;primaryKey"`
	Nome     string `gorm:"column:nome"`
	Cidade   string `gorm:"column:cidade"`
	Estado   string `gorm:"column:estado"`
	Endereco string `gorm:"column:endereco"`
	Ativo    bool   `gorm:"column:ativo"`
}

func (cinemaRow) TableName() string { return "cinemas" }

type salaRow struct {
	ID       string `gorm:"column:id;primaryKey"`
	CinemaID string `gorm:"column:cinema_id"`
	Numero   int    `gorm:"column:numero"`
	TipoTela string `gorm:"column:tipo_tela"`
	Layout   []byte `gorm:"column:layout;type:jsonb"`
	Ativo    bool   `gorm:"column:ativo"`
}

func (salaRow) TableName() string { return "salas" }

type sessaoRow struct {
	ID             string    `gorm:"column:id;primaryKey"`
	FilmeID        string    `gorm:"column:filme_id"`
	SalaID         string    `gorm:"column:sala_id"`
	DataHoraInicio time.Time `gorm:"column:data_hora_inicio"`
	Idioma         string    `gorm:"column:idioma"`
	PrecoBase      string    `gorm:"column:preco_base"`
	Status         string    `gorm:"column:status"`
}

func (sessaoRow) TableName() string { return "sessoes" }

type sessaoDetalhadaRow struct {
	ID             string    `gorm:"column:id"`
	FilmeID        string    `gorm:"column:filme_id"`
	FilmeTitulo    string    `gorm:"column:filme_titulo"`
	CinemaID       string    `gorm:"column:cinema_id"`
	CinemaNome     string    `gorm:"column:cinema_nome"`
	SalaNumero     int       `gorm:"column:sala_numero"`
	TipoTela       string    `gorm:"column:tipo_tela"`
	DataHoraInicio time.Time `gorm:"column:data_hora_inicio"`
	Idioma         string    `gorm:"column:idioma"`
	PrecoBase      string    `gorm:"column:preco_base"`
}

const colunasSessaoDetalhada = `s.id, s.filme_id, f.titulo AS filme_titulo,
	c.id AS cinema_id, c.nome AS cinema_nome, sa.numero AS sala_numero,
	sa.tipo_tela, s.data_hora_inicio, s.idioma, s.preco_base`

type outboxRow struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	MessageID    string `gorm:"column:message_id"`
	RoutingKey   string `gorm:"column:routing_key"`
	Payload      []byte `gorm:"column:payload;type:jsonb"`
	TraceContext []byte `gorm:"column:trace_context;type:jsonb"`
}

func (outboxRow) TableName() string { return "outbox_eventos" }
