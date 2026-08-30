package poltrona

import (
	"errors"
	"testing"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

func TestTransicoesPermitidas(t *testing.T) {
	casos := []struct {
		nome     string
		de, para Status
		aceita   bool
	}{
		{"livre bloqueada", Livre, Reservada, true},
		{"reservada confirmada", Reservada, Ocupada, true},
		{"reservada liberada", Reservada, Livre, true},
		{"livre direto para ocupada", Livre, Ocupada, false},
		{"ocupada de volta para livre", Ocupada, Livre, false},
		{"ocupada rebloqueada", Ocupada, Reservada, false},
		{"livre para livre", Livre, Livre, false},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			p := Poltrona{Rotulo: "A1", Status: caso.de}
			resultado, err := p.Transicionar(caso.para)

			if caso.aceita {
				if err != nil {
					t.Fatalf("esperava transição %s→%s aceita, veio: %v", caso.de, caso.para, err)
				}
				if resultado.Status != caso.para {
					t.Errorf("status = %s, esperado %s", resultado.Status, caso.para)
				}
				return
			}
			if err == nil {
				t.Fatalf("esperava recusa da transição %s→%s", caso.de, caso.para)
			}
			if !errors.Is(err, shared.ErrTransicaoInvalida) {
				t.Errorf("erro = %v, esperado ErrTransicaoInvalida", err)
			}
			if resultado.Status != caso.de {
				t.Errorf("transição recusada não pode alterar o estado: %s", resultado.Status)
			}
		})
	}
}

func TestPodeSerBloqueada(t *testing.T) {
	for status, esperado := range map[Status]bool{Livre: true, Reservada: false, Ocupada: false} {
		if got := (Poltrona{Status: status}).PodeSerBloqueada(); got != esperado {
			t.Errorf("PodeSerBloqueada(%s) = %v, esperado %v", status, got, esperado)
		}
	}
}

func TestNovaDerivaIdentidadeDeFormaDeterministica(t *testing.T) {
	const sessao = "f781a9b2-11e2-4f81-a901-8890bc123456"

	p1, err := Nova(sessao, "a", 1, Normal)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	p2, err := Nova(sessao, "A", 1, Normal)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if p1.ID != p2.ID {
		t.Errorf("mesma poltrona gerou ids diferentes: %s != %s", p1.ID, p2.ID)
	}
	if p1.Rotulo != "A1" {
		t.Errorf("rótulo = %q, esperado A1", p1.Rotulo)
	}
	if p1.Status != Livre {
		t.Errorf("poltrona nasce %s, esperado LIVRE", p1.Status)
	}

	outra, _ := Nova(sessao, "A", 2, Normal)
	if outra.ID == p1.ID {
		t.Error("poltronas distintas não podem compartilhar identificador")
	}
	deOutraSessao, _ := Nova("outra-sessao", "A", 1, Normal)
	if deOutraSessao.ID == p1.ID {
		t.Error("mesma fileira/número em sessões distintas não pode colidir")
	}
}

func TestNovaRecusaEntradaInvalida(t *testing.T) {
	casos := map[string]struct {
		fileira string
		numero  int
		tipo    Tipo
	}{
		"fileira vazia":     {"", 1, Normal},
		"número zero":       {"A", 0, Normal},
		"número negativo":   {"A", -3, Normal},
		"tipo desconhecido": {"A", 1, Tipo("POLTRONA_VIP")},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := Nova("sessao", caso.fileira, caso.numero, caso.tipo); err == nil {
				t.Fatal("esperava recusa")
			} else if !errors.Is(err, shared.ErrSolicitacaoInvalida) {
				t.Errorf("erro = %v, esperado ErrSolicitacaoInvalida", err)
			}
		})
	}
}

func TestLerRotulo(t *testing.T) {
	validos := map[string]struct {
		fileira string
		numero  int
	}{
		"A1":      {"A", 1},
		"a1":      {"A", 1},
		"  B12  ": {"B", 12},
		"AA7":     {"AA", 7},
	}
	for entrada, esperado := range validos {
		fileira, numero, err := LerRotulo(entrada)
		if err != nil {
			t.Errorf("LerRotulo(%q) devolveu erro: %v", entrada, err)
			continue
		}
		if fileira != esperado.fileira || numero != esperado.numero {
			t.Errorf("LerRotulo(%q) = (%s,%d), esperado (%s,%d)", entrada, fileira, numero, esperado.fileira, esperado.numero)
		}
	}

	for _, invalido := range []string{"", "1", "A", "A0", "A-1", "1A", "A1B", "ABCDEF1"} {
		if _, _, err := LerRotulo(invalido); err == nil {
			t.Errorf("LerRotulo(%q) devia recusar", invalido)
		}
	}
}
