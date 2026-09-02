package catalogo

import (
	"testing"
	"time"
)

func TestAceitaReserva(t *testing.T) {
	agora := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	futuro := agora.Add(2 * time.Hour)
	passado := agora.Add(-2 * time.Hour)

	casos := []struct {
		nome     string
		status   StatusSessao
		inicio   time.Time
		esperado bool
	}{
		{"agendada e futura", SessaoAgendada, futuro, true},
		{"agendada mas já começou", SessaoAgendada, passado, false},
		{"agendada começando exatamente agora", SessaoAgendada, agora, false},
		{"em andamento", SessaoEmAndamento, passado, false},
		{"em andamento com início futuro (dado inconsistente)", SessaoEmAndamento, futuro, false},
		{"finalizada", SessaoFinalizada, passado, false},
		{"cancelada", SessaoCancelada, futuro, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			s := Sessao{Status: c.status, DataHoraInicio: c.inicio}
			if got := s.AceitaReserva(agora); got != c.esperado {
				t.Fatalf("esperava %v, obteve %v", c.esperado, got)
			}
		})
	}
}

func TestParseStatusFilme(t *testing.T) {
	for _, v := range []string{"EM_CARTAZ", "BREVE", "FORA_DE_CARTAZ"} {
		if _, err := ParseStatusFilme(v); err != nil {
			t.Errorf("%s deveria ser aceito: %v", v, err)
		}
	}
	_, err := ParseStatusFilme("EM_BREVE")
	if err == nil {
		t.Fatal("esperava recusa de status desconhecido")
	}
	for _, v := range []string{"EM_CARTAZ", "BREVE", "FORA_DE_CARTAZ"} {
		if !contains(err.Error(), v) {
			t.Errorf("mensagem não lista %s: %s", v, err)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestDinheiroPreservaExatidao(t *testing.T) {
	d := DinheiroDeCentavos(4200)
	if d.String() != "42.00" {
		t.Fatalf("esperava 42.00, obteve %s", d.String())
	}
	if DinheiroDeCentavos(5).String() != "0.05" {
		t.Fatalf("centavos isolados: %s", DinheiroDeCentavos(5).String())
	}
	if DinheiroDeCentavos(-150).String() != "-1.50" {
		t.Fatalf("negativo: %s", DinheiroDeCentavos(-150).String())
	}
}
