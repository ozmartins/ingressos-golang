package catalogo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type TipoPoltrona string

// Os três tipos são os que o estoque aceita ao provisionar a matriz de uma
// sessão; o catálogo não pode inventar um quarto, porque quem materializa a
// poltrona é do outro lado.
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

// Uma fileira é uniforme: todos os seus assentos são do mesmo tipo. Um assento
// PCD no meio de uma fileira comum se declara como fileira própria.
type Fileira struct {
	Letra    string
	Assentos int
	Tipo     TipoPoltrona
}

// A planta da sala. É sempre lida e escrita inteira, junto da sala.
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

// Uma poltrona da planta: a expansão de uma fileira em assentos numerados de 1
// em diante. O catálogo não guarda poltrona — quem as materializa é o estoque —,
// mas precisa enumerá-las para anunciar a sessão.
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

// Igual responde se duas plantas descrevem a mesma sala. A comparação é
// posicional porque `NovoLayoutSala` sempre ordena as fileiras: a mesma planta
// descrita em ordens diferentes chega aqui na mesma ordem.
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

// Dados desfaz a planta na forma de entrada, para quem precisa reapresentá-la a
// `NovaSala` sem tê-la recebido do cliente.
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

// O teto de 5 caracteres e o alfabeto sem dígito vêm do que o estoque aceita
// numa fileira: o rótulo da poltrona é a letra seguida do número, e um dígito na
// letra tornaria o rótulo ambíguo.
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

		// Sem `tipo` na fileira, ela nasce normal — mesmo padrão da coluna do
		// estoque, onde a maioria das poltronas é comum.
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

	// A ordem alfabética torna a resposta e a ida e volta pelo banco
	// determinísticas, independentemente da ordem em que o corpo as listou.
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
