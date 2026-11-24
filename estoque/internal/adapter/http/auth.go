package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/retry"
)

var ErrCredencial = errors.New("http: credencial inválida")

type Autenticador struct {
	chaves   jwt.Keyfunc
	issuer   string
	audience string
}

func NovoAutenticador(jwksURL, issuer, audience string) (*Autenticador, error) {
	var k keyfunc.Keyfunc
	err := retry.Executar(context.Background(), "carregar JWKS", retry.Dependencia, func() (e error) {
		k, e = keyfunc.NewDefault([]string{jwksURL})
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("carregar JWKS de %s: %w", jwksURL, err)
	}
	return &Autenticador{chaves: k.Keyfunc, issuer: issuer, audience: audience}, nil
}

func NovoAutenticadorComChave(kf jwt.Keyfunc, issuer, audience string) *Autenticador {
	return &Autenticador{chaves: kf, issuer: issuer, audience: audience}
}

func (a *Autenticador) autenticar(r *http.Request) (Identidade, error) {
	bruto, err := tokenDoCabecalho(r)
	if err != nil {
		return Identidade{}, ErrCredencial
	}

	t, err := jwt.Parse(bruto, a.chaves,
		jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.audience),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !t.Valid {
		return Identidade{}, ErrCredencial
	}

	sub, err := t.Claims.GetSubject()
	if err != nil || sub == "" {
		return Identidade{}, ErrCredencial
	}
	return Identidade{UsuarioID: sub, Permissoes: permissoesDe(t.Claims)}, nil
}

// PermissaoCliente é o papel de realm que o Keycloak concede a toda pessoa
// usuária (reservar, pagar, ver ingressos). Tokens de service account não o
// têm: autenticados (401 não se aplica), mas sem permissão (403).
const PermissaoCliente = "cliente"

// ErrSemPermissao: credencial válida, mas sem a permissão exigida (403).
var ErrSemPermissao = errors.New("http: credencial sem permissão")

// Identidade é o que o token comprova: quem é e o que pode fazer.
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

// Identificar valida o token e devolve só o sub (qualquer pessoa autenticada).
func (a *Autenticador) Identificar(r *http.Request) (string, error) {
	id, err := a.autenticar(r)
	return id.UsuarioID, err
}

// Exigir valida o token e confere a permissão: ErrCredencial (401) se o token
// é ausente/inválido/expirado, ErrSemPermissao (403) se é válido mas não a concede.
func (a *Autenticador) Exigir(r *http.Request, permissao string) (string, error) {
	id, err := a.autenticar(r)
	if err != nil {
		return "", err
	}
	if !id.Tem(permissao) {
		return "", ErrSemPermissao
	}
	return id.UsuarioID, nil
}

// permissoesDe junta papéis de realm (realm_access.roles) e escopos (scope).
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

var errSemCredencial = errors.New("http: cabeçalho Authorization ausente ou malformado")

func tokenDoCabecalho(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", errSemCredencial
	}
	partes := strings.SplitN(h, " ", 2)
	if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") {
		return "", errSemCredencial
	}
	return strings.TrimSpace(partes[1]), nil
}
