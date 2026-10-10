package identidade

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

var ErrCredencialInvalida = errors.New("credencial inválida")

// caminhoJWKS é o endpoint de chaves públicas de um realm do Keycloak, relativo
// ao emissor.
const caminhoJWKS = "/protocol/openid-connect/certs"

type Verificador struct {
	chaves    jwt.Keyfunc
	emissor   string
	audiencia string
}

// NovoVerificador carrega as chaves públicas do realm (emissor + caminhoJWKS).
func NovoVerificador(emissor, audiencia string) (*Verificador, error) {
	jwksURL := strings.TrimRight(emissor, "/") + caminhoJWKS
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("carregar JWKS de %s: %w", jwksURL, err)
	}
	return NovoVerificadorComChave(k.Keyfunc, emissor, audiencia), nil
}

func NovoVerificadorComChave(chaves jwt.Keyfunc, emissor, audiencia string) *Verificador {
	return &Verificador{chaves: chaves, emissor: emissor, audiencia: audiencia}
}

type Identidade struct {
	UsuarioID string
}

func (v *Verificador) Verificar(_ context.Context, tokenBruto string) (Identidade, error) {
	token, err := jwt.Parse(tokenBruto, v.chaves,
		jwt.WithIssuer(v.emissor),
		jwt.WithAudience(v.audiencia),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !token.Valid {
		return Identidade{}, fmt.Errorf("%w: %v", ErrCredencialInvalida, err)
	}
	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return Identidade{}, fmt.Errorf("%w: credencial sem identificação da pessoa usuária", ErrCredencialInvalida)
	}
	return Identidade{UsuarioID: sub}, nil
}
