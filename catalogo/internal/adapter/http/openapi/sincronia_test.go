package openapi

import (
	"os"
	"testing"
)

// caminhoContrato é a fonte da verdade do contrato: o artefato versionado do
// spec-kit. O arquivo embutido neste pacote é uma cópia de runtime, exigida
// porque go:embed não alcança diretórios acima do pacote.
const caminhoContrato = "../../../../specs/001-catalogo-sessoes-reserva/contracts/openapi.yaml"

// Uma cópia que ninguém verifica é uma cópia que diverge. Este teste é o que
// mantém o documento servido em /openapi.yaml igual ao contrato acordado.
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
