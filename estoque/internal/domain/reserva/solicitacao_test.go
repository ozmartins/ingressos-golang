package reserva

import (
	"errors"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const limitePadrao = 10

func TestNovaSolicitacaoNormalizaRotulos(t *testing.T) {
	s, err := NovaSolicitacao("sessao", "usuario", []string{"a1", " B2 ", "C10"}, "84.00", limitePadrao)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	esperado := []string{"A1", "B2", "C10"}
	if strings.Join(s.Rotulos, ",") != strings.Join(esperado, ",") {
		t.Errorf("rótulos = %v, esperado %v", s.Rotulos, esperado)
	}
}

// O valor não é decidido aqui, mas o formato é conferido: ele vira cobrança, e
// um valor malformado que passe daqui só apareceria na hora de cobrar.
func TestNovaSolicitacaoRecusaValorInvalido(t *testing.T) {
	casos := map[string]string{
		"ausente":            "",
		"em branco":          "   ",
		"zero":               "0",
		"zero com decimais":  "0.00",
		"negativo":           "-84.00",
		"três casas":         "84.000",
		"ponto sem decimais": "84.",
		"expoente":           "1e2",
		"fração":             "1/3",
		"com moeda":          "R$ 84,00",
		"vírgula decimal":    "84,00",
		"texto":              "de graça",
	}
	for nome, valor := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := NovaSolicitacao("sessao", "usuario", []string{"A1"}, valor, limitePadrao)
			if !errors.Is(err, shared.ErrSolicitacaoInvalida) {
				t.Fatalf("erro = %v, esperado ErrSolicitacaoInvalida", err)
			}
		})
	}
}

func TestNovaSolicitacaoAceitaValoresBemFormados(t *testing.T) {
	for _, valor := range []string{"84", "84.0", "84.00", "0.01", "1234567.89"} {
		t.Run(valor, func(t *testing.T) {
			s, err := NovaSolicitacao("sessao", "usuario", []string{"A1"}, " "+valor+" ", limitePadrao)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if s.ValorTotal != valor {
				t.Errorf("valor_total = %q, esperado %q (com as bordas aparadas)", s.ValorTotal, valor)
			}
		})
	}
}

func TestNovaSolicitacaoRecusaEntradaInvalida(t *testing.T) {
	casos := map[string]struct {
		sessao, usuario string
		rotulos         []string
		erro            error
	}{
		"lista vazia":       {"sessao", "usuario", nil, shared.ErrSolicitacaoInvalida},
		"rótulo repetido":   {"sessao", "usuario", []string{"A1", "a1"}, shared.ErrSolicitacaoInvalida},
		"usuário ausente":   {"sessao", "  ", []string{"A1"}, shared.ErrSolicitacaoInvalida},
		"sessão ausente":    {"", "usuario", []string{"A1"}, shared.ErrSolicitacaoInvalida},
		"rótulo malformado": {"sessao", "usuario", []string{"cadeira-boa"}, shared.ErrSolicitacaoInvalida},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := NovaSolicitacao(caso.sessao, caso.usuario, caso.rotulos, "84.00", limitePadrao); err == nil {
				t.Fatal("esperava recusa")
			} else if !errors.Is(err, caso.erro) {
				t.Errorf("erro = %v, esperado %v", err, caso.erro)
			}
		})
	}
}

func TestNovaSolicitacaoAplicaLimiteConfiguravel(t *testing.T) {
	dez := []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10"}

	if _, err := NovaSolicitacao("sessao", "usuario", dez, "84.00", limitePadrao); err != nil {
		t.Fatalf("exatamente no limite devia ser aceito: %v", err)
	}

	onze := append(append([]string{}, dez...), "A11")
	_, err := NovaSolicitacao("sessao", "usuario", onze, "84.00", limitePadrao)
	if !errors.Is(err, shared.ErrLimiteExcedido) {
		t.Fatalf("erro = %v, esperado ErrLimiteExcedido", err)
	}

	if _, err := NovaSolicitacao("sessao", "usuario", []string{"A1", "A2", "A3"}, "84.00", 2); !errors.Is(err, shared.ErrLimiteExcedido) {
		t.Errorf("erro = %v, esperado ErrLimiteExcedido com limite 2", err)
	}
}
