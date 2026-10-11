package sessao

import (
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

var agora = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

func TestAceitaReserva(t *testing.T) {
	casos := map[string]struct {
		s        Sessao
		esperado bool
	}{
		"agendada futura":         {Sessao{Status: Agendada, Inicio: agora.Add(time.Hour)}, true},
		"agendada que já começou": {Sessao{Status: Agendada, Inicio: agora.Add(-time.Minute)}, false},
		"cancelada":               {Sessao{Status: "CANCELADA", Inicio: agora.Add(time.Hour)}, false},
		"em andamento":            {Sessao{Status: "EM_ANDAMENTO", Inicio: agora.Add(time.Hour)}, false},
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			if got := c.s.AceitaReserva(agora); got != c.esperado {
				t.Errorf("AceitaReserva = %v, esperado %v", got, c.esperado)
			}
		})
	}
}

func TestValorTotal(t *testing.T) {
	casos := []struct {
		preco string
		qtd   int
		valor string
	}{
		{"42.00", 2, "84.00"},
		{"32", 3, "96.00"},
		{"32.5", 3, "97.50"},
		{"0.01", 1, "0.01"},
		{"19.99", 10, "199.90"},
	}
	for _, c := range casos {
		got, err := Sessao{PrecoBase: c.preco}.ValorTotal(c.qtd)
		if err != nil || got != c.valor {
			t.Errorf("ValorTotal(%q × %d) = %q, %v; esperado %q", c.preco, c.qtd, got, err, c.valor)
		}
	}
}

func TestValorTotalRecusaPrecoInvalido(t *testing.T) {
	for _, preco := range []string{"", "abc", "-1.00", "1.234", "1.", ".50", "0", "0.00"} {
		_, err := Sessao{PrecoBase: preco}.ValorTotal(1)
		if !errors.Is(err, shared.ErrRespostaInvalidaDoCatalogo) {
			t.Errorf("preço %q: err = %v, esperado ErrRespostaInvalidaDoCatalogo", preco, err)
		}
	}
}
