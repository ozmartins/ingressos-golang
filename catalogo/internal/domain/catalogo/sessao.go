package catalogo

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type StatusSessao string

const (
	SessaoAgendada    StatusSessao = "AGENDADA"
	SessaoEmAndamento StatusSessao = "EM_ANDAMENTO"
	SessaoFinalizada  StatusSessao = "FINALIZADA"
	SessaoCancelada   StatusSessao = "CANCELADA"
)

var statusSessaoConhecidos = []StatusSessao{
	SessaoAgendada, SessaoEmAndamento, SessaoFinalizada, SessaoCancelada,
}

var StatusVisiveisNaGrade = []StatusSessao{SessaoAgendada, SessaoEmAndamento}

func ParseStatusSessao(v string) (StatusSessao, error) {
	s := StatusSessao(v)
	for _, conhecido := range statusSessaoConhecidos {
		if s == conhecido {
			return s, nil
		}
	}
	return "", fmt.Errorf("%w: status %q não é reconhecido; valores aceitos: %s",
		shared.ErrValidacao, v, listar(statusSessaoConhecidos))
}

type Idioma string

const (
	Dublado   Idioma = "DUBLADO"
	Legendado Idioma = "LEGENDADO"
)

var idiomasConhecidos = []Idioma{Dublado, Legendado}

func ParseIdioma(v string) (Idioma, error) {
	i := Idioma(v)
	for _, conhecido := range idiomasConhecidos {
		if i == conhecido {
			return i, nil
		}
	}
	return "", fmt.Errorf("%w: idioma %q não é reconhecido; valores aceitos: %s",
		shared.ErrValidacao, v, listar(idiomasConhecidos))
}

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

// A sessão não guarda a própria duração: ela é a do filme em cartaz naquela
// sala. Quem sabe a duração é quem chama.
func (s Sessao) FimPrevisto(duracaoMinutos int) time.Time {
	return s.DataHoraInicio.Add(time.Duration(duracaoMinutos) * time.Minute)
}

type DadosSessao struct {
	FilmeID        string
	SalaID         string
	DataHoraInicio time.Time
	Idioma         string
	PrecoBase      string
	Status         string
}

func NovaSessao(id string, d DadosSessao) (Sessao, error) {
	switch {
	case d.FilmeID == "":
		return Sessao{}, fmt.Errorf("%w: filme_id é obrigatório", shared.ErrValidacao)
	case d.SalaID == "":
		return Sessao{}, fmt.Errorf("%w: sala_id é obrigatório", shared.ErrValidacao)
	case d.DataHoraInicio.IsZero():
		return Sessao{}, fmt.Errorf("%w: data_hora_inicio é obrigatória", shared.ErrValidacao)
	case d.Idioma == "":
		return Sessao{}, fmt.Errorf("%w: idioma é obrigatório", shared.ErrValidacao)
	}

	idioma, err := ParseIdioma(d.Idioma)
	if err != nil {
		return Sessao{}, err
	}

	preco, err := parsePreco(d.PrecoBase)
	if err != nil {
		return Sessao{}, err
	}

	// Sem `status` no corpo, a sessão nasce agendada — mesmo padrão da coluna.
	status := SessaoAgendada
	if d.Status != "" {
		s, err := ParseStatusSessao(d.Status)
		if err != nil {
			return Sessao{}, err
		}
		status = s
	}

	return Sessao{
		ID:             id,
		FilmeID:        d.FilmeID,
		SalaID:         d.SalaID,
		DataHoraInicio: d.DataHoraInicio.UTC(),
		Idioma:         idioma,
		PrecoBase:      preco,
		Status:         status,
	}, nil
}

// O preço chega como texto — é assim que ele sai em `paraSessaoDTO`, e é o único
// formato que atravessa o JSON sem o arredondamento binário do float. O formato
// é conferido à mão porque `big.Rat` também aceitaria "1/3" e "1e2".
func parsePreco(v string) (Dinheiro, error) {
	invalido := fmt.Errorf("%w: preco_base deve ser um valor como \"32.00\", com até duas casas decimais", shared.ErrValidacao)

	inteiro, decimais, temPonto := strings.Cut(v, ".")
	if inteiro == "" || (temPonto && (decimais == "" || len(decimais) > 2)) {
		return Dinheiro{}, invalido
	}
	for _, parte := range []string{inteiro, decimais} {
		for _, c := range parte {
			if c < '0' || c > '9' {
				return Dinheiro{}, invalido
			}
		}
	}

	r, ok := new(big.Rat).SetString(v)
	if !ok {
		return Dinheiro{}, invalido
	}
	preco, err := DinheiroDeRat(r)
	if err != nil {
		return Dinheiro{}, invalido
	}
	if preco.Centavos() <= 0 {
		return Dinheiro{}, fmt.Errorf("%w: preco_base deve ser maior que zero", shared.ErrValidacao)
	}
	return preco, nil
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
