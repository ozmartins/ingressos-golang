package reserva

import (
	"errors"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const limitePadrao = 10

func TestNovaSolicitacaoNormalizaRotulos(t *testing.T) {
	s, err := NovaSolicitacao("sessao", "usuario", []string{"a1", " B2 ", "C10"}, limitePadrao)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	esperado := []string{"A1", "B2", "C10"}
	if strings.Join(s.Rotulos, ",") != strings.Join(esperado, ",") {
		t.Errorf("rótulos = %v, esperado %v", s.Rotulos, esperado)
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
			if _, err := NovaSolicitacao(caso.sessao, caso.usuario, caso.rotulos, limitePadrao); err == nil {
				t.Fatal("esperava recusa")
			} else if !errors.Is(err, caso.erro) {
				t.Errorf("erro = %v, esperado %v", err, caso.erro)
			}
		})
	}
}

func TestNovaSolicitacaoAplicaLimiteConfiguravel(t *testing.T) {
	dez := []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10"}

	if _, err := NovaSolicitacao("sessao", "usuario", dez, limitePadrao); err != nil {
		t.Fatalf("exatamente no limite devia ser aceito: %v", err)
	}

	onze := append(append([]string{}, dez...), "A11")
	_, err := NovaSolicitacao("sessao", "usuario", onze, limitePadrao)
	if !errors.Is(err, shared.ErrLimiteExcedido) {
		t.Fatalf("erro = %v, esperado ErrLimiteExcedido", err)
	}

	// O limite é configurável: com teto de 2, três poltronas já estouram.
	if _, err := NovaSolicitacao("sessao", "usuario", []string{"A1", "A2", "A3"}, 2); !errors.Is(err, shared.ErrLimiteExcedido) {
		t.Errorf("erro = %v, esperado ErrLimiteExcedido com limite 2", err)
	}
}
