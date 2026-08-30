package reserva

import (
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

var instante = time.Date(2026, 8, 29, 21, 33, 0, 0, time.UTC)

func TestNovaCriaPendenteComPrazo(t *testing.T) {
	r := Nova("sessao", "usuario", []string{"A1", "A2"}, instante, 10*time.Minute)

	if r.Status != Pendente {
		t.Errorf("status = %s, esperado PENDENTE", r.Status)
	}
	if !r.ExpiraEm.Equal(instante.Add(10 * time.Minute)) {
		t.Errorf("expira_em = %v, esperado %v", r.ExpiraEm, instante.Add(10*time.Minute))
	}
	if r.ID == "" {
		t.Error("reserva sem identificador")
	}
}

func TestExpirou(t *testing.T) {
	r := Nova("sessao", "usuario", []string{"A1"}, instante, 10*time.Minute)

	if r.Expirou(instante.Add(9*time.Minute + 59*time.Second)) {
		t.Error("não devia expirar antes do prazo")
	}
	// O vencimento é inclusivo: no instante exato, a reserva já venceu.
	if !r.Expirou(instante.Add(10 * time.Minute)) {
		t.Error("devia expirar no instante do prazo")
	}
	if !r.Expirou(instante.Add(time.Hour)) {
		t.Error("devia expirar depois do prazo")
	}

	// FR-014: reserva confirmada nunca expira, por mais que o prazo passe.
	confirmada, err := r.Transicionar(Confirmada)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if confirmada.Expirou(instante.Add(24 * time.Hour)) {
		t.Error("reserva confirmada não pode expirar")
	}
}

func TestTransicaoSoAceitaAPartirDePendente(t *testing.T) {
	base := Nova("sessao", "usuario", []string{"A1"}, instante, time.Minute)

	for _, destino := range []Status{Confirmada, Cancelada, Expirada} {
		t.Run(string(destino), func(t *testing.T) {
			finalizada, err := base.Transicionar(destino)
			if err != nil {
				t.Fatalf("PENDENTE→%s devia ser aceita: %v", destino, err)
			}
			if finalizada.Status != destino {
				t.Fatalf("status = %s, esperado %s", finalizada.Status, destino)
			}

			// Estado final é imutável: o segundo desfecho é ignorado (FR-011).
			for _, outro := range []Status{Confirmada, Cancelada, Expirada} {
				novamente, err := finalizada.Transicionar(outro)
				if err == nil {
					t.Errorf("%s→%s devia ser recusada", destino, outro)
				}
				if !errors.Is(err, shared.ErrTransicaoInvalida) {
					t.Errorf("erro = %v, esperado ErrTransicaoInvalida", err)
				}
				if novamente.Status != destino {
					t.Errorf("transição recusada alterou o estado para %s", novamente.Status)
				}
			}
		})
	}
}

func TestTransicaoRecusaEstadoNaoFinal(t *testing.T) {
	r := Nova("sessao", "usuario", []string{"A1"}, instante, time.Minute)
	if _, err := r.Transicionar(Pendente); err == nil {
		t.Error("PENDENTE→PENDENTE devia ser recusada")
	}
}
