package catalogo

import (
	"fmt"

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
