package postgres

import (
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

// dinheiroDeNumeric converte NUMERIC(10,2) para o tipo monetário exato do
// domínio, sem passar por ponto flutuante em momento algum.
func dinheiroDeNumeric(n pgtype.Numeric) (catalogo.Dinheiro, error) {
	if !n.Valid {
		return catalogo.Dinheiro{}, fmt.Errorf("preco_base nulo")
	}
	if n.NaN || n.InfinityModifier != pgtype.Finite {
		return catalogo.Dinheiro{}, fmt.Errorf("preco_base não é um número finito")
	}
	// valor = Int * 10^Exp, exato.
	r := new(big.Rat).SetInt(n.Int)
	dez := big.NewInt(10)
	if n.Exp >= 0 {
		r.Mul(r, new(big.Rat).SetInt(new(big.Int).Exp(dez, big.NewInt(int64(n.Exp)), nil)))
	} else {
		r.Quo(r, new(big.Rat).SetInt(new(big.Int).Exp(dez, big.NewInt(int64(-n.Exp)), nil)))
	}
	return catalogo.DinheiroDeRat(r)
}
