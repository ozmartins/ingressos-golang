package catalogo

import (
	"fmt"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type TipoTela string

const (
	Tela2D   TipoTela = "2D"
	Tela3D   TipoTela = "3D"
	TelaIMAX TipoTela = "IMAX"
	TelaVIP  TipoTela = "VIP"
)

var tiposDeTelaConhecidos = []TipoTela{Tela2D, Tela3D, TelaIMAX, TelaVIP}

func ParseTipoTela(v string) (TipoTela, error) {
	t := TipoTela(v)
	for _, conhecido := range tiposDeTelaConhecidos {
		if t == conhecido {
			return t, nil
		}
	}
	return "", fmt.Errorf("%w: tipo_tela %q não é reconhecido; valores aceitos: %s",
		shared.ErrValidacao, v, listar(tiposDeTelaConhecidos))
}

type Sala struct {
	ID              string
	CinemaID        string
	Numero          int
	TipoTela        TipoTela
	CapacidadeTotal int
	Ativo           bool
}

type DadosSala struct {
	CinemaID        string
	Numero          int
	TipoTela        string
	CapacidadeTotal int
	Ativo           *bool
}

func NovaSala(id string, d DadosSala) (Sala, error) {
	switch {
	case d.CinemaID == "":
		return Sala{}, fmt.Errorf("%w: cinema_id é obrigatório", shared.ErrValidacao)
	case d.Numero <= 0:
		return Sala{}, fmt.Errorf("%w: numero deve ser maior que zero", shared.ErrValidacao)
	case d.CapacidadeTotal <= 0:
		return Sala{}, fmt.Errorf("%w: capacidade_total deve ser maior que zero", shared.ErrValidacao)
	case d.TipoTela == "":
		return Sala{}, fmt.Errorf("%w: tipo_tela é obrigatório", shared.ErrValidacao)
	}

	tipo, err := ParseTipoTela(d.TipoTela)
	if err != nil {
		return Sala{}, err
	}

	// Sem `ativo` no corpo, a sala nasce ativa — mesmo padrão da coluna.
	ativo := true
	if d.Ativo != nil {
		ativo = *d.Ativo
	}

	return Sala{
		ID:              id,
		CinemaID:        d.CinemaID,
		Numero:          d.Numero,
		TipoTela:        tipo,
		CapacidadeTotal: d.CapacidadeTotal,
		Ativo:           ativo,
	}, nil
}
