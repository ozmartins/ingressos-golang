package catalogo

import (
	"fmt"
	"strings"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type StatusFilme string

const (
	StatusEmCartaz     StatusFilme = "EM_CARTAZ"
	StatusBreve        StatusFilme = "BREVE"
	StatusForaDeCartaz StatusFilme = "FORA_DE_CARTAZ"
)

var statusFilmeConhecidos = []StatusFilme{StatusEmCartaz, StatusBreve, StatusForaDeCartaz}

var StatusPublicos = []StatusFilme{StatusEmCartaz, StatusBreve}

func ParseStatusFilme(v string) (StatusFilme, error) {
	s := StatusFilme(v)
	for _, conhecido := range statusFilmeConhecidos {
		if s == conhecido {
			return s, nil
		}
	}
	return "", fmt.Errorf("%w: status %q não é reconhecido; valores aceitos: %s",
		shared.ErrValidacao, v, listar(statusFilmeConhecidos))
}

func (s StatusFilme) Valido() bool {
	for _, conhecido := range statusFilmeConhecidos {
		if s == conhecido {
			return true
		}
	}
	return false
}

type Filme struct {
	ID                  string
	Titulo              string
	Sinopse             *string
	DuracaoMinutos      int
	ClassificacaoEtaria string
	Genero              string
	ImagemURL           *string
	Status              StatusFilme
}

func listar[T ~string](vs []T) string {
	s := ""
	for i, v := range vs {
		if i > 0 {
			s += ", "
		}
		s += string(v)
	}
	return s
}

type DadosFilme struct {
	Titulo              string
	Sinopse             *string
	DuracaoMinutos      int
	ClassificacaoEtaria string
	Genero              string
	ImagemURL           *string
	Status              string
}

func NovoFilme(id string, d DadosFilme) (Filme, error) {
	titulo := strings.TrimSpace(d.Titulo)
	classificacao := strings.TrimSpace(d.ClassificacaoEtaria)
	genero := strings.TrimSpace(d.Genero)

	switch {
	case titulo == "":
		return Filme{}, fmt.Errorf("%w: titulo é obrigatório", shared.ErrValidacao)
	case len(titulo) > 255:
		return Filme{}, fmt.Errorf("%w: titulo deve ter no máximo 255 caracteres", shared.ErrValidacao)
	case d.DuracaoMinutos <= 0:
		return Filme{}, fmt.Errorf("%w: duracao_minutos deve ser maior que zero", shared.ErrValidacao)
	case classificacao == "":
		return Filme{}, fmt.Errorf("%w: classificacao_etaria é obrigatória", shared.ErrValidacao)
	case len(classificacao) > 50:
		return Filme{}, fmt.Errorf("%w: classificacao_etaria deve ter no máximo 50 caracteres", shared.ErrValidacao)
	case genero == "":
		return Filme{}, fmt.Errorf("%w: genero é obrigatório", shared.ErrValidacao)
	case len(genero) > 100:
		return Filme{}, fmt.Errorf("%w: genero deve ter no máximo 100 caracteres", shared.ErrValidacao)
	case d.ImagemURL != nil && len(*d.ImagemURL) > 500:
		return Filme{}, fmt.Errorf("%w: imagem_url deve ter no máximo 500 caracteres", shared.ErrValidacao)
	}

	// Sem `status` no corpo, o filme nasce em cartaz — mesmo padrão da coluna.
	status := StatusEmCartaz
	if d.Status != "" {
		s, err := ParseStatusFilme(d.Status)
		if err != nil {
			return Filme{}, err
		}
		status = s
	}

	return Filme{
		ID:                  id,
		Titulo:              titulo,
		Sinopse:             d.Sinopse,
		DuracaoMinutos:      d.DuracaoMinutos,
		ClassificacaoEtaria: classificacao,
		Genero:              genero,
		ImagemURL:           d.ImagemURL,
		Status:              status,
	}, nil
}
