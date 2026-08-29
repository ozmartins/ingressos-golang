package catalogo

import (
	"fmt"
	"math/big"
)

// Dinheiro é um valor monetário exato, em centavos.
//
// preco_base é DECIMAL(10,2) no banco. Convertê-lo para float64 introduziria
// erro de representação binária em valores decimais — 42.00 vira 42.000000000001
// e a diferença aparece em soma de carrinho. Aqui o valor é inteiro e a
// serialização é textual.
type Dinheiro struct {
	centavos int64
}

func DinheiroDeCentavos(c int64) Dinheiro { return Dinheiro{centavos: c} }

func (d Dinheiro) Centavos() int64 { return d.centavos }

// String devolve a representação decimal com duas casas, para serialização.
func (d Dinheiro) String() string {
	sinal := ""
	c := d.centavos
	if c < 0 {
		sinal, c = "-", -c
	}
	return fmt.Sprintf("%s%d.%02d", sinal, c/100, c%100)
}

// DinheiroDeRat converte um racional exato (como o vindo de NUMERIC) para
// centavos, recusando valores que não caibam em duas casas decimais.
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
