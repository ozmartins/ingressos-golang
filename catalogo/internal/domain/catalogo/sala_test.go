package catalogo

import (
	"errors"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func dadosSalaValidos() DadosSala {
	return DadosSala{
		CinemaID: "b1b2c3d4-0000-4000-8000-000000000001",
		Numero:   3,
		TipoTela: "IMAX",
		Fileiras: []DadosFileira{
			{Fileira: "A", Assentos: 12},
			{Fileira: "B", Assentos: 12},
		},
	}
}

func TestNovaSalaSemAtivoNasceAtiva(t *testing.T) {
	sala, err := NovaSala("sala-1", dadosSalaValidos())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !sala.Ativo {
		t.Fatal("sem `ativo` no corpo, a sala deveria nascer ativa")
	}
	if sala.TipoTela != TelaIMAX {
		t.Fatalf("tipo_tela = %q", sala.TipoTela)
	}
	if sala.CapacidadeTotal() != 24 {
		t.Fatalf("a capacidade deveria ser a soma das fileiras; obteve %d", sala.CapacidadeTotal())
	}
}

func TestNovaSalaRespeitaAtivoExplicito(t *testing.T) {
	inativa := false
	dados := dadosSalaValidos()
	dados.Ativo = &inativa

	sala, err := NovaSala("sala-1", dados)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if sala.Ativo {
		t.Fatal("`ativo: false` no corpo deveria valer")
	}
}

func TestNovaSalaRecusaEntradasInvalidas(t *testing.T) {
	casos := map[string]func(*DadosSala){
		"sem cinema":         func(d *DadosSala) { d.CinemaID = "" },
		"numero zero":        func(d *DadosSala) { d.Numero = 0 },
		"numero negativo":    func(d *DadosSala) { d.Numero = -1 },
		"sem fileiras":       func(d *DadosSala) { d.Fileiras = nil },
		"sem tipo de tela":   func(d *DadosSala) { d.TipoTela = "" },
		"tela desconhecida":  func(d *DadosSala) { d.TipoTela = "4DX" },
		"tela em minúsculas": func(d *DadosSala) { d.TipoTela = "imax" },
	}
	for nome, quebrar := range casos {
		t.Run(nome, func(t *testing.T) {
			dados := dadosSalaValidos()
			quebrar(&dados)
			if _, err := NovaSala("sala-1", dados); !errors.Is(err, shared.ErrValidacao) {
				t.Fatalf("esperava ErrValidacao, obteve %v", err)
			}
		})
	}
}

func TestParseTipoTelaListaOsValoresAceitos(t *testing.T) {
	_, err := ParseTipoTela("4DX")
	if err == nil {
		t.Fatal("esperava erro para um tipo desconhecido")
	}
	for _, esperado := range []string{"2D", "3D", "IMAX", "VIP"} {
		if !strings.Contains(err.Error(), esperado) {
			t.Errorf("a mensagem deveria listar %q: %s", esperado, err)
		}
	}
}
