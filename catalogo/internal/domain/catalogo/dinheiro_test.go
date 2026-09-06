package catalogo

import "testing"

// A multiplicação por quantidade inteira precisa ser exata: o resultado vira
// cobrança, e um centavo perdido aqui é um centavo cobrado errado.
func TestDinheiroMultiplicar(t *testing.T) {
	casos := []struct {
		nome     string
		centavos int64
		fator    int
		esperado string
	}{
		{"preço com centavos", 4250, 3, "127.50"},
		{"uma poltrona não muda nada", 4250, 1, "42.50"},
		{"o centavo mais baixo", 1, 3, "0.03"},
		{"resultado atravessa o real", 99, 2, "1.98"},
		{"dez poltronas, o teto do bloqueio", 3899, 10, "389.90"},
		{"nenhuma poltrona", 4250, 0, "0.00"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := DinheiroDeCentavos(c.centavos).Multiplicar(c.fator).String(); got != c.esperado {
				t.Fatalf("%d centavos x %d = %s, esperado %s", c.centavos, c.fator, got, c.esperado)
			}
		})
	}
}

// O tipo é imutável: multiplicar devolve outro valor e não altera o original.
func TestDinheiroMultiplicarNaoAlteraOOriginal(t *testing.T) {
	preco := DinheiroDeCentavos(4250)
	_ = preco.Multiplicar(3)
	if preco.String() != "42.50" {
		t.Fatalf("o original mudou para %s", preco)
	}
}
