package openapi

import "testing"

func TestEspecificacaoNaoEstaVazia(t *testing.T) {
	if len(Especificacao) == 0 {
		t.Fatal("a especificação embutida está vazia")
	}
}
