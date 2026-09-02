package openapi

import (
	"os"
	"testing"
)

const caminhoContrato = "../../../../specs/001-catalogo-sessoes-reserva/contracts/openapi.yaml"

func TestCopiaEmbutidaEstaSincronizadaComOContrato(t *testing.T) {
	origem, err := os.ReadFile(caminhoContrato)
	if err != nil {
		t.Fatalf("lendo o contrato de origem: %v", err)
	}

	if string(origem) != string(Especificacao) {
		t.Errorf("a cópia embutida divergiu do contrato em %s.\n"+
			"O contrato é a fonte da verdade; para ressincronizar: make openapi-sync",
			caminhoContrato)
	}
}

func TestEspecificacaoNaoEstaVazia(t *testing.T) {
	if len(Especificacao) == 0 {
		t.Fatal("a especificação embutida está vazia")
	}
}
