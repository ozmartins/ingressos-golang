package catalogo

import (
	"errors"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func fileirasValidas() []DadosFileira {
	return []DadosFileira{
		{Fileira: "A", Assentos: 12},
		{Fileira: "F", Assentos: 8, Tipo: "PCD"},
		{Fileira: "J", Assentos: 6, Tipo: "NAMORADEIRA"},
	}
}

func TestNovoLayoutSalaSomaAsFileiras(t *testing.T) {
	layout, err := NovoLayoutSala(fileirasValidas())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if layout.CapacidadeTotal() != 26 {
		t.Fatalf("capacidade = %d, esperava 26", layout.CapacidadeTotal())
	}
}

func TestNovoLayoutSalaSemTipoNasceNormal(t *testing.T) {
	layout, err := NovoLayoutSala([]DadosFileira{{Fileira: "A", Assentos: 10}})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if layout.Fileiras[0].Tipo != PoltronaNormal {
		t.Fatalf("sem `tipo`, a fileira deveria nascer normal; obteve %q", layout.Fileiras[0].Tipo)
	}
}

func TestNovoLayoutSalaNormalizaALetra(t *testing.T) {
	layout, err := NovoLayoutSala([]DadosFileira{{Fileira: "  a ", Assentos: 10}})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if layout.Fileiras[0].Letra != "A" {
		t.Fatalf("letra = %q, esperava %q", layout.Fileiras[0].Letra, "A")
	}
}

// A ordem da resposta não pode depender da ordem em que o corpo listou as
// fileiras: a mesma planta descrita de duas formas é a mesma planta.
func TestNovoLayoutSalaOrdenaPorLetra(t *testing.T) {
	layout, err := NovoLayoutSala([]DadosFileira{
		{Fileira: "C", Assentos: 4},
		{Fileira: "A", Assentos: 4},
		{Fileira: "B", Assentos: 4},
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	for i, esperada := range []string{"A", "B", "C"} {
		if layout.Fileiras[i].Letra != esperada {
			t.Fatalf("fileira %d = %q, esperava %q", i, layout.Fileiras[i].Letra, esperada)
		}
	}
}

func TestNovoLayoutSalaRecusaEntradasInvalidas(t *testing.T) {
	casos := map[string][]DadosFileira{
		"sem fileira alguma": {},
		"nula":               nil,
		"letra vazia":        {{Fileira: "", Assentos: 10}},
		"letra em branco":    {{Fileira: "   ", Assentos: 10}},
		"letra com dígito":   {{Fileira: "A1", Assentos: 10}},
		"letra longa demais": {{Fileira: "ABCDEF", Assentos: 10}},
		"assentos zero":      {{Fileira: "A", Assentos: 0}},
		"assentos negativos": {{Fileira: "A", Assentos: -1}},
		"fileira repetida":   {{Fileira: "A", Assentos: 10}, {Fileira: "a", Assentos: 8}},
		"tipo desconhecido":  {{Fileira: "A", Assentos: 10, Tipo: "PUFE"}},
		"tipo em minúsculas": {{Fileira: "A", Assentos: 10, Tipo: "pcd"}},
	}
	for nome, fileiras := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := NovoLayoutSala(fileiras); !errors.Is(err, shared.ErrValidacao) {
				t.Fatalf("esperava ErrValidacao, obteve %v", err)
			}
		})
	}
}

func TestParseTipoPoltronaListaOsValoresAceitos(t *testing.T) {
	_, err := ParseTipoPoltrona("PUFE")
	if err == nil {
		t.Fatal("esperava erro para um tipo desconhecido")
	}
	for _, esperado := range []string{"NORMAL", "PCD", "NAMORADEIRA"} {
		if !strings.Contains(err.Error(), esperado) {
			t.Errorf("a mensagem deveria listar %q: %s", esperado, err)
		}
	}
}
