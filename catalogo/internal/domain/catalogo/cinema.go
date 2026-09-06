package catalogo

import (
	"fmt"
	"strings"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type Cinema struct {
	ID       string
	Nome     string
	Cidade   string
	Estado   string
	Endereco string
	Ativo    bool
}

type DadosCinema struct {
	Nome     string
	Cidade   string
	Estado   string
	Endereco string
	Ativo    *bool
}

func NovoCinema(id string, d DadosCinema) (Cinema, error) {
	nome := strings.TrimSpace(d.Nome)
	cidade := strings.TrimSpace(d.Cidade)
	estado := strings.ToUpper(strings.TrimSpace(d.Estado))
	endereco := strings.TrimSpace(d.Endereco)

	switch {
	case nome == "":
		return Cinema{}, fmt.Errorf("%w: nome é obrigatório", shared.ErrValidacao)
	case len(nome) > 255:
		return Cinema{}, fmt.Errorf("%w: nome deve ter no máximo 255 caracteres", shared.ErrValidacao)
	case cidade == "":
		return Cinema{}, fmt.Errorf("%w: cidade é obrigatória", shared.ErrValidacao)
	case len(cidade) > 100:
		return Cinema{}, fmt.Errorf("%w: cidade deve ter no máximo 100 caracteres", shared.ErrValidacao)
	case len(estado) != 2:
		return Cinema{}, fmt.Errorf("%w: estado deve ter exatamente 2 letras (sigla da unidade federativa)", shared.ErrValidacao)
	case endereco == "":
		return Cinema{}, fmt.Errorf("%w: endereco é obrigatório", shared.ErrValidacao)
	}

	// Sem `ativo` no corpo, o cinema nasce ativo — mesmo padrão da coluna.
	ativo := true
	if d.Ativo != nil {
		ativo = *d.Ativo
	}

	return Cinema{
		ID:       id,
		Nome:     nome,
		Cidade:   cidade,
		Estado:   estado,
		Endereco: endereco,
		Ativo:    ativo,
	}, nil
}
