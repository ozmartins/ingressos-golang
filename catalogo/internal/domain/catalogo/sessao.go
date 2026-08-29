package catalogo

import "time"

// StatusSessao é a situação de uma exibição no seu ciclo de vida.
type StatusSessao string

const (
	SessaoAgendada    StatusSessao = "AGENDADA"
	SessaoEmAndamento StatusSessao = "EM_ANDAMENTO"
	SessaoFinalizada  StatusSessao = "FINALIZADA"
	SessaoCancelada   StatusSessao = "CANCELADA"
)

// StatusVisiveisNaGrade são as situações oferecidas ao cliente (FR-016).
var StatusVisiveisNaGrade = []StatusSessao{SessaoAgendada, SessaoEmAndamento}

// Idioma da exibição.
type Idioma string

const (
	Dublado   Idioma = "DUBLADO"
	Legendado Idioma = "LEGENDADO"
)

// Sessao é a exibição de um filme numa sala em um instante específico.
type Sessao struct {
	ID             string
	FilmeID        string
	SalaID         string
	DataHoraInicio time.Time
	Idioma         Idioma
	PrecoBase      Dinheiro
	Status         StatusSessao
}

// AceitaReserva informa se a sessão ainda pode receber bloqueio de poltronas.
//
// Exige situação AGENDADA *e* início no futuro. A segunda condição existe porque
// a transição para EM_ANDAMENTO é feita por um processo externo e pode atrasar:
// confiar só na situação deixaria uma janela em que sessões já iniciadas
// aceitariam reservas.
func (s Sessao) AceitaReserva(agora time.Time) bool {
	return s.Status == SessaoAgendada && s.DataHoraInicio.After(agora)
}

// SessaoDetalhada é a vista de leitura da grade: consolida quatro tabelas numa
// linha. Não é um agregado com identidade própria.
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
