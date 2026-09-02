package identidade

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

var ErrCredencialInvalida = errors.New("credencial inválida")

type Verificador struct {
	verificador *oidc.IDTokenVerifier
}

func NovoVerificador(ctx context.Context, issuerURL, audiencia string) (*Verificador, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("descobrindo emissor de credenciais em %s: %w", issuerURL, err)
	}
	return &Verificador{
		verificador: provider.Verifier(&oidc.Config{
			ClientID:             audiencia,
			SupportedSigningAlgs: []string{oidc.RS256},
		}),
	}, nil
}

func NovoVerificadorComKeySet(ks oidc.KeySet, issuerURL, audiencia string) *Verificador {
	return &Verificador{
		verificador: oidc.NewVerifier(issuerURL, ks, &oidc.Config{
			ClientID:             audiencia,
			SupportedSigningAlgs: []string{oidc.RS256},
		}),
	}
}

type Identidade struct {
	UsuarioID string
}

func (v *Verificador) Verificar(ctx context.Context, tokenBruto string) (Identidade, error) {
	token, err := v.verificador.Verify(ctx, tokenBruto)
	if err != nil {
		return Identidade{}, fmt.Errorf("%w: %v", ErrCredencialInvalida, err)
	}
	if token.Subject == "" {
		return Identidade{}, fmt.Errorf("%w: credencial sem identificação da pessoa usuária", ErrCredencialInvalida)
	}
	return Identidade{UsuarioID: token.Subject}, nil
}
