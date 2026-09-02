package catalogo

import (
	"fmt"
	"math/big"
)

type Dinheiro struct {
	centavos int64
}

func DinheiroDeCentavos(c int64) Dinheiro { return Dinheiro{centavos: c} }

func (d Dinheiro) Centavos() int64 { return d.centavos }

func (d Dinheiro) String() string {
	sinal := ""
	c := d.centavos
	if c < 0 {
		sinal, c = "-", -c
	}
	return fmt.Sprintf("%s%d.%02d", sinal, c/100, c%100)
}

func DinheiroDeRat(r *big.Rat) (Dinheiro, error) {
	if r == nil {
		return Dinheiro{}, fmt.Errorf("valor monetário ausente")
	}
	cem := new(big.Rat).SetInt64(100)
	escalado := new(big.Rat).Mul(r, cem)
	if !escalado.IsInt() {
		return Dinheiro{}, fmt.Errorf("valor monetário %s tem mais de duas casas decimais", r.FloatString(4))
	}
	return Dinheiro{centavos: escalado.Num().Int64()}, nil
}
