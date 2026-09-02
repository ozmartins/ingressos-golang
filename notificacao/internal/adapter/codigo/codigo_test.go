package codigo

import (
	"errors"
	"strings"
	"testing"
)

func assinador(t *testing.T, segredo string) *Assinador {
	t.Helper()
	a, err := NovoAssinador(segredo)
	if err != nil {
		t.Fatalf("NovoAssinador: %v", err)
	}
	return a
}

func TestIdaEVolta(t *testing.T) {
	a := assinador(t, "segredo-de-teste")
	const id = "771a9210-9981-42a1-b882-102938471290"

	c := a.Gerar(id)
	if !strings.HasPrefix(c, Prefixo+".") {
		t.Errorf("código %q não começa com o prefixo", c)
	}
	if len(c) > 255 {
		t.Errorf("código com %d caracteres não cabe no VARCHAR(255)", len(c))
	}
	if strings.Contains(c, id) {
		t.Error("o identificador aparece em texto legível dentro do código")
	}

	volta, err := a.Verificar(c)
	if err != nil {
		t.Fatalf("Verificar recusou código legítimo: %v", err)
	}
	if volta != id {
		t.Errorf("id = %q, queria %q", volta, id)
	}
}

func TestCodigosDeIngressosDiferentesSaoDistintos(t *testing.T) {
	a := assinador(t, "segredo-de-teste")
	if a.Gerar("ing-1") == a.Gerar("ing-2") {
		t.Error("ingressos diferentes geraram o mesmo código (FR-005)")
	}
}

func TestGerarEDeterministico(t *testing.T) {
	a := assinador(t, "segredo-de-teste")
	if a.Gerar("ing-1") != a.Gerar("ing-1") {
		t.Error("o mesmo ingresso gerou códigos diferentes")
	}
}

func TestVerificarRecusa(t *testing.T) {
	a := assinador(t, "segredo-de-teste")
	legitimo := a.Gerar("ing-1")

	adulterado := legitimo[:len(legitimo)-1] + trocaUltimo(legitimo)
	corpoTrocado := Prefixo + "." + strings.Split(a.Gerar("ing-2"), ".")[1] + "." + strings.Split(legitimo, ".")[2]

	casos := map[string]string{
		"vazio":                 "",
		"lixo":                  "lixo",
		"prefixo errado":        strings.Replace(legitimo, Prefixo, "XXX9", 1),
		"duas partes":           Prefixo + "." + strings.Split(legitimo, ".")[1],
		"quatro partes":         legitimo + ".extra",
		"assinatura adulterada": adulterado,
		"corpo trocado":         corpoTrocado,
		"base64 quebrada":       Prefixo + ".!!!.###",
		"id vazio":              Prefixo + ".." + strings.Split(legitimo, ".")[2],
		"absurdamente longo":    Prefixo + "." + strings.Repeat("A", 300),
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := a.Verificar(c); !errors.Is(err, ErrInvalido) {
				t.Errorf("Verificar(%q) = %v, queria ErrInvalido", c, err)
			}
		})
	}
}

func TestSegredoDiferenteNaoValida(t *testing.T) {
	deles := assinador(t, "segredo-do-atacante").Gerar("ing-1")
	if _, err := assinador(t, "segredo-de-teste").Verificar(deles); !errors.Is(err, ErrInvalido) {
		t.Errorf("código assinado com outro segredo foi aceito: %v", err)
	}
}

func TestSegredoVazioRecusado(t *testing.T) {
	if _, err := NovoAssinador(""); err == nil {
		t.Error("NovoAssinador aceitou segredo vazio")
	}
}

func trocaUltimo(s string) string {
	if s[len(s)-1] == 'A' {
		return "B"
	}
	return "A"
}
