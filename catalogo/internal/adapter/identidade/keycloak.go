package identidade

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/retry"
)

var ErrCredencialInvalida = errors.New("credencial inválida")

const caminhoJWKS = "/protocol/openid-connect/certs"

type Verificador struct {
	chaves    jwt.Keyfunc
	emissor   string
	audiencia string
}

func NovoVerificador(emissor, audiencia string) (*Verificador, error) {
	jwksURL := strings.TrimRight(emissor, "/") + caminhoJWKS
	var k keyfunc.Keyfunc
	err := retry.Executar(context.Background(), "carregar JWKS", retry.Dependencia, func() (e error) {
		k, e = keyfunc.NewDefault([]string{jwksURL})
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("carregar JWKS de %s: %w", jwksURL, err)
	}
	return NovoVerificadorComChave(k.Keyfunc, emissor, audiencia), nil
}

func NovoVerificadorComChave(chaves jwt.Keyfunc, emissor, audiencia string) *Verificador {
	return &Verificador{chaves: chaves, emissor: emissor, audiencia: audiencia}
}

// Permissões que o catálogo exige. Vêm do token: papéis de realm
// (`realm_access.roles`) e/ou escopos OAuth (`scope`) do Keycloak.
const (
	// PermissaoEscrita autoriza criar, alterar e remover cinemas, salas, filmes e sessões.
	PermissaoEscrita = "catalogo-admin"
	// PermissaoLerSessao autoriza a consulta de sessão por outros serviços (gRPC).
	PermissaoLerSessao = "catalogo-leitura-sessao"
)

type Identidade struct {
	UsuarioID  string
	Permissoes []string
}

// Tem diz se o token concedeu a permissão.
func (i Identidade) Tem(permissao string) bool {
	for _, p := range i.Permissoes {
		if p == permissao {
			return true
		}
	}
	return false
}

// permissoesDe junta os papéis de realm e os escopos do token, sem repetição.
func permissoesDe(claims jwt.Claims) []string {
	mc, ok := claims.(jwt.MapClaims)
	if !ok {
		return nil
	}
	vistas := map[string]bool{}
	var out []string
	add := func(p string) {
		if p != "" && !vistas[p] {
			vistas[p] = true
			out = append(out, p)
		}
	}
	if ra, ok := mc["realm_access"].(map[string]any); ok {
		if roles, ok := ra["roles"].([]any); ok {
			for _, r := range roles {
				if s, ok := r.(string); ok {
					add(s)
				}
			}
		}
	}
	if sc, ok := mc["scope"].(string); ok {
		for _, s := range strings.Fields(sc) {
			add(s)
		}
	}
	return out
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
	return Identidade{UsuarioID: sub, Permissoes: permissoesDe(token.Claims)}, nil
}
