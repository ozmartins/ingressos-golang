package catalogo

import (
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
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

func dadosSessaoValidos() DadosSessao {
	return DadosSessao{
		FilmeID:        "a1a2a3a4-0000-4000-8000-000000000001",
		SalaID:         "b1b2c3d4-0000-4000-8000-000000000001",
		DataHoraInicio: time.Date(2026, 9, 20, 19, 30, 0, 0, time.UTC),
		Idioma:         "LEGENDADO",
		PrecoBase:      "42.50",
	}
}

func TestNovaSessaoSemStatusNasceAgendada(t *testing.T) {
	sessao, err := NovaSessao("sessao-1", dadosSessaoValidos())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if sessao.Status != SessaoAgendada {
		t.Fatalf("status = %q, esperava AGENDADA", sessao.Status)
	}
	if sessao.PrecoBase.Centavos() != 4250 {
		t.Fatalf("preco_base = %d centavos", sessao.PrecoBase.Centavos())
	}
}

func TestNovaSessaoRecusaEntradasInvalidas(t *testing.T) {
	casos := map[string]func(*DadosSessao){
		"sem filme":                    func(d *DadosSessao) { d.FilmeID = "" },
		"sem sala":                     func(d *DadosSessao) { d.SalaID = "" },
		"sem início":                   func(d *DadosSessao) { d.DataHoraInicio = time.Time{} },
		"sem idioma":                   func(d *DadosSessao) { d.Idioma = "" },
		"idioma desconhecido":          func(d *DadosSessao) { d.Idioma = "ORIGINAL" },
		"status desconhecido":          func(d *DadosSessao) { d.Status = "PAUSADA" },
		"preço ausente":                func(d *DadosSessao) { d.PrecoBase = "" },
		"preço zero":                   func(d *DadosSessao) { d.PrecoBase = "0.00" },
		"preço negativo":               func(d *DadosSessao) { d.PrecoBase = "-10.00" },
		"preço com três casas":         func(d *DadosSessao) { d.PrecoBase = "42.505" },
		"preço com vírgula":            func(d *DadosSessao) { d.PrecoBase = "42,50" },
		"preço como fração":            func(d *DadosSessao) { d.PrecoBase = "85/2" },
		"preço em notação exponencial": func(d *DadosSessao) { d.PrecoBase = "4.25e1" },
	}
	for nome, quebrar := range casos {
		t.Run(nome, func(t *testing.T) {
			dados := dadosSessaoValidos()
			quebrar(&dados)
			if _, err := NovaSessao("sessao-1", dados); !errors.Is(err, shared.ErrValidacao) {
				t.Fatalf("esperava ErrValidacao, obteve %v", err)
			}
		})
	}
}

func TestNovaSessaoNormalizaOInicioParaUTC(t *testing.T) {
	fusoDeSaoPaulo := time.FixedZone("-03", -3*60*60)
	dados := dadosSessaoValidos()
	dados.DataHoraInicio = time.Date(2026, 9, 20, 19, 30, 0, 0, fusoDeSaoPaulo)

	sessao, err := NovaSessao("sessao-1", dados)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if hora := sessao.DataHoraInicio.Hour(); hora != 22 {
		t.Fatalf("o início deveria virar 22h UTC, obteve %dh", hora)
	}
}

func TestFimPrevistoSomaADuracaoDoFilme(t *testing.T) {
	sessao, err := NovaSessao("sessao-1", dadosSessaoValidos())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	fim := sessao.FimPrevisto(148)
	if esperado := time.Date(2026, 9, 20, 21, 58, 0, 0, time.UTC); !fim.Equal(esperado) {
		t.Fatalf("fim = %s, esperava %s", fim, esperado)
	}
}
