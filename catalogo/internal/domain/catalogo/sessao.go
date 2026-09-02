package catalogo

import "time"

type StatusSessao string

const (
	SessaoAgendada    StatusSessao = "AGENDADA"
	SessaoEmAndamento StatusSessao = "EM_ANDAMENTO"
	SessaoFinalizada  StatusSessao = "FINALIZADA"
	SessaoCancelada   StatusSessao = "CANCELADA"
)

var StatusVisiveisNaGrade = []StatusSessao{SessaoAgendada, SessaoEmAndamento}

type Idioma string

const (
	Dublado   Idioma = "DUBLADO"
	Legendado Idioma = "LEGENDADO"
)

type Sessao struct {
	ID             string
	FilmeID        string
	SalaID         string
	DataHoraInicio time.Time
	Idioma         Idioma
	PrecoBase      Dinheiro
	Status         StatusSessao
}

func (s Sessao) AceitaReserva(agora time.Time) bool {
	return s.Status == SessaoAgendada && s.DataHoraInicio.After(agora)
}

type SessaoDetalhada struct {
	ID             string
	FilmeID        string
	FilmeTitulo    string
	CinemaID       string
	CinemaNome     string
	SalaNumero     int
	TipoTela       TipoTela
	DataHoraInicio time.Time
	Idioma         Idioma
	PrecoBase      Dinheiro
}
