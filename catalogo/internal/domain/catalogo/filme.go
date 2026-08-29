// Package catalogo reúne as entidades estruturais expostas pelo serviço.
// Nada aqui conhece banco, transporte ou framework (constituição, princípio I).
package catalogo

import (
	"fmt"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// StatusFilme é a situação de exibição de uma obra.
type StatusFilme string

const (
	StatusEmCartaz     StatusFilme = "EM_CARTAZ"
	StatusBreve        StatusFilme = "BREVE"
	StatusForaDeCartaz StatusFilme = "FORA_DE_CARTAZ"
)

// statusFilmeConhecidos é a fonte da verdade para validação e para a mensagem
// de erro que lista os valores aceitos (FR-009).
var statusFilmeConhecidos = []StatusFilme{StatusEmCartaz, StatusBreve, StatusForaDeCartaz}

// StatusPublicos são as situações que a vitrine mostra quando nenhum filtro é
// informado (FR-008): fora de cartaz não é oferta.
var StatusPublicos = []StatusFilme{StatusEmCartaz, StatusBreve}

// ParseStatusFilme valida um valor recebido de fora.
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

// Valido informa se a situação lida do armazenamento é conhecida. Valor
// desconhecido é erro de dados, não valor a repassar ao cliente.
func (s StatusFilme) Valido() bool {
	for _, conhecido := range statusFilmeConhecidos {
		if s == conhecido {
			return true
		}
	}
	return false
}

// Filme é uma obra em exibição ou prevista.
//
// Sinopse e ImagemURL são ponteiros porque a ausência é significativa: um filme
// sem material de apoio continua listável, e o campo é omitido da resposta em
// vez de virar string vazia.
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
