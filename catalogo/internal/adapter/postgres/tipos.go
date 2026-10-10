package postgres

import (
	"fmt"
	"math/big"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

func dinheiroDeTexto(texto string) (catalogo.Dinheiro, error) {
	if texto == "" {
		return catalogo.Dinheiro{}, fmt.Errorf("preco_base nulo")
	}
	r, ok := new(big.Rat).SetString(texto)
	if !ok {
		return catalogo.Dinheiro{}, fmt.Errorf("preco_base %q não é um número finito", texto)
	}
	return catalogo.DinheiroDeRat(r)
}
