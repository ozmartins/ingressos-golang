package catalogo

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type Cinema struct {
	ID       string
	Nome     string
	Cidade   string
	Estado   string
	Endereco string
	LogoURL  string // vazio = cinema sem logo
	Ativo    bool
}

type DadosCinema struct {
	Nome     string
	Cidade   string
	Estado   string
	Endereco string
	LogoURL  string
	Ativo    *bool
}

const tamanhoMaximoLogoURL = 2048

func NovoCinema(id string, d DadosCinema) (Cinema, error) {
	nome := strings.TrimSpace(d.Nome)
	cidade := strings.TrimSpace(d.Cidade)
	estado := strings.ToUpper(strings.TrimSpace(d.Estado))
	endereco := strings.TrimSpace(d.Endereco)
	logoURL := strings.TrimSpace(d.LogoURL)

	switch {
	case nome == "":
		return Cinema{}, fmt.Errorf("%w: nome é obrigatório", shared.ErrValidacao)
	case len(nome) > 255:
		return Cinema{}, fmt.Errorf("%w: nome deve ter no máximo 255 caracteres", shared.ErrValidacao)
	case cidade == "":
		return Cinema{}, fmt.Errorf("%w: cidade é obrigatória", shared.ErrValidacao)
	case len(cidade) > 100:
		return Cinema{}, fmt.Errorf("%w: cidade deve ter no máximo 100 caracteres", shared.ErrValidacao)
	case len(estado) != 2:
		return Cinema{}, fmt.Errorf("%w: estado deve ter exatamente 2 letras (sigla da unidade federativa)", shared.ErrValidacao)
	case endereco == "":
		return Cinema{}, fmt.Errorf("%w: endereco é obrigatório", shared.ErrValidacao)
	}
	if err := validarLogoURL(logoURL); err != nil {
		return Cinema{}, err
	}

	ativo := true
	if d.Ativo != nil {
		ativo = *d.Ativo
	}

	return Cinema{
		ID:       id,
		Nome:     nome,
		Cidade:   cidade,
		Estado:   estado,
		Endereco: endereco,
		LogoURL:  logoURL,
		Ativo:    ativo,
	}, nil
}

// validarLogoURL aceita vazio (cinema sem logo) ou uma URL http/https absoluta,
// sem espaços e de até 2048 caracteres. Esquemas como javascript: e data: são
// recusados porque a URL será renderizada por clientes.
func validarLogoURL(valor string) error {
	if valor == "" {
		return nil
	}
	if len(valor) > tamanhoMaximoLogoURL {
		return fmt.Errorf("%w: logo_url deve ter no máximo %d caracteres", shared.ErrValidacao, tamanhoMaximoLogoURL)
	}
	if strings.ContainsAny(valor, " \t\r\n") {
		return fmt.Errorf("%w: logo_url não pode conter espaços", shared.ErrValidacao)
	}
	u, err := url.Parse(valor)
	if err != nil {
		return fmt.Errorf("%w: logo_url deve ser uma URL absoluta válida", shared.ErrValidacao)
	}
	if esquema := strings.ToLower(u.Scheme); esquema != "" && esquema != "http" && esquema != "https" {
		return fmt.Errorf("%w: logo_url deve usar o esquema http ou https", shared.ErrValidacao)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%w: logo_url deve ser uma URL absoluta válida", shared.ErrValidacao)
	}
	return nil
}
