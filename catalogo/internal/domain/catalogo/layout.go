package catalogo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type TipoPoltrona string

const (
	PoltronaNormal      TipoPoltrona = "NORMAL"
	PoltronaPCD         TipoPoltrona = "PCD"
	PoltronaNamoradeira TipoPoltrona = "NAMORADEIRA"
)

var tiposDePoltronaConhecidos = []TipoPoltrona{PoltronaNormal, PoltronaPCD, PoltronaNamoradeira}

func ParseTipoPoltrona(v string) (TipoPoltrona, error) {
	t := TipoPoltrona(v)
	for _, conhecido := range tiposDePoltronaConhecidos {
		if t == conhecido {
			return t, nil
		}
	}
	return "", fmt.Errorf("%w: tipo %q não é reconhecido; valores aceitos: %s",
		shared.ErrValidacao, v, listar(tiposDePoltronaConhecidos))
}

type Fileira struct {
	Letra    string
	Assentos int
	Tipo     TipoPoltrona
}

type LayoutSala struct {
	Fileiras []Fileira
}

func (l LayoutSala) CapacidadeTotal() int {
	total := 0
	for _, f := range l.Fileiras {
		total += f.Assentos
	}
	return total
}

type PoltronaDoLayout struct {
	Fileira string
	Numero  int
	Tipo    TipoPoltrona
}

func (l LayoutSala) Poltronas() []PoltronaDoLayout {
	poltronas := make([]PoltronaDoLayout, 0, l.CapacidadeTotal())
	for _, f := range l.Fileiras {
		for n := 1; n <= f.Assentos; n++ {
			poltronas = append(poltronas, PoltronaDoLayout{Fileira: f.Letra, Numero: n, Tipo: f.Tipo})
		}
	}
	return poltronas
}

func (l LayoutSala) Igual(outra LayoutSala) bool {
	if len(l.Fileiras) != len(outra.Fileiras) {
		return false
	}
	for i, f := range l.Fileiras {
		if f != outra.Fileiras[i] {
			return false
		}
	}
	return true
}

func (l LayoutSala) Dados() []DadosFileira {
	ds := make([]DadosFileira, 0, len(l.Fileiras))
	for _, f := range l.Fileiras {
		ds = append(ds, DadosFileira{Fileira: f.Letra, Assentos: f.Assentos, Tipo: string(f.Tipo)})
	}
	return ds
}

type DadosFileira struct {
	Fileira  string
	Assentos int
	Tipo     string
}

const maxLetraFileira = 5

func NovoLayoutSala(ds []DadosFileira) (LayoutSala, error) {
	if len(ds) == 0 {
		return LayoutSala{}, fmt.Errorf("%w: fileiras deve ter ao menos uma fileira", shared.ErrValidacao)
	}

	fileiras := make([]Fileira, 0, len(ds))
	vistas := make(map[string]struct{}, len(ds))

	for _, d := range ds {
		letra := strings.ToUpper(strings.TrimSpace(d.Fileira))
		switch {
		case letra == "":
			return LayoutSala{}, fmt.Errorf("%w: fileira é obrigatória", shared.ErrValidacao)
		case len(letra) > maxLetraFileira:
			return LayoutSala{}, fmt.Errorf("%w: fileira %q deve ter no máximo %d letras",
				shared.ErrValidacao, d.Fileira, maxLetraFileira)
		case !somenteLetras(letra):
			return LayoutSala{}, fmt.Errorf("%w: fileira %q deve conter apenas letras de A a Z",
				shared.ErrValidacao, d.Fileira)
		case d.Assentos <= 0:
			return LayoutSala{}, fmt.Errorf("%w: assentos da fileira %s deve ser maior que zero",
				shared.ErrValidacao, letra)
		}

		if _, repetida := vistas[letra]; repetida {
			return LayoutSala{}, fmt.Errorf("%w: fileira %s aparece mais de uma vez", shared.ErrValidacao, letra)
		}
		vistas[letra] = struct{}{}

		tipo := PoltronaNormal
		if d.Tipo != "" {
			t, err := ParseTipoPoltrona(d.Tipo)
			if err != nil {
				return LayoutSala{}, err
			}
			tipo = t
		}

		fileiras = append(fileiras, Fileira{Letra: letra, Assentos: d.Assentos, Tipo: tipo})
	}

	sort.Slice(fileiras, func(i, j int) bool { return fileiras[i].Letra < fileiras[j].Letra })

	return LayoutSala{Fileiras: fileiras}, nil
}

func somenteLetras(s string) bool {
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
