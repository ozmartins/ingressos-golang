package catalogo

import (
	"errors"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func dadosCinemaValidos(logo string) DadosCinema {
	return DadosCinema{
		Nome: "CineMark - Beiramar", Cidade: "Florianópolis",
		Estado: "SC", Endereco: "Rua Y, 200", LogoURL: logo,
	}
}

func TestNovoCinemaAceitaLogoURL(t *testing.T) {
	casos := map[string]struct{ entrada, esperado string }{
		"http":                  {"http://cdn.exemplo.com/logo.png", "http://cdn.exemplo.com/logo.png"},
		"https":                 {"https://cdn.exemplo.com/logo.png", "https://cdn.exemplo.com/logo.png"},
		"esquema em maiúsculas": {"HTTPS://cdn.exemplo.com/logo.png", "HTTPS://cdn.exemplo.com/logo.png"},
		"espaços nas pontas":    {"  https://cdn.exemplo.com/logo.png \n", "https://cdn.exemplo.com/logo.png"},
		"vazia":                 {"", ""},
		"só espaços":            {"   ", ""},
		"2048 caracteres":       {"https://a.co/" + strings.Repeat("x", 2048-len("https://a.co/")), "https://a.co/" + strings.Repeat("x", 2048-len("https://a.co/"))},
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			cinema, err := NovoCinema("id-1", dadosCinemaValidos(c.entrada))
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if cinema.LogoURL != c.esperado {
				t.Fatalf("LogoURL = %q, esperava %q", cinema.LogoURL, c.esperado)
			}
		})
	}
}

func TestNovoCinemaRecusaLogoURLInvalida(t *testing.T) {
	casos := map[string]struct{ entrada, motivo string }{
		"javascript":      {"javascript:alert(1)", "http ou https"},
		"ftp":             {"ftp://x.com/logo.png", "http ou https"},
		"sem host":        {"https://", "URL absoluta"},
		"espaço interno":  {"https://a b.com/logo.png", "espaços"},
		"texto livre":     {"isto não é uma url", "espaços"},
		"sem esquema":     {"cdn.exemplo.com/logo.png", "URL absoluta"},
		"2049 caracteres": {"https://a.co/" + strings.Repeat("x", 2049-len("https://a.co/")), "2048"},
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := NovoCinema("id-1", dadosCinemaValidos(c.entrada))
			if !errors.Is(err, shared.ErrValidacao) {
				t.Fatalf("esperava ErrValidacao, obteve %v", err)
			}
			if !strings.Contains(err.Error(), "logo_url") || !strings.Contains(err.Error(), c.motivo) {
				t.Fatalf("a mensagem deveria citar logo_url e %q: %v", c.motivo, err)
			}
		})
	}
}
