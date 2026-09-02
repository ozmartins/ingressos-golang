package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

var ErrCredencial = errors.New("http: credencial inválida")

type Autenticador struct {
	chaves   jwt.Keyfunc
	issuer   string
	audience string
}

func NovoAutenticador(jwksURL, issuer, audience string) (*Autenticador, error) {
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("carregar JWKS de %s: %w", jwksURL, err)
	}
	return &Autenticador{chaves: k.Keyfunc, issuer: issuer, audience: audience}, nil
}

func NovoAutenticadorComChave(kf jwt.Keyfunc, issuer, audience string) *Autenticador {
	return &Autenticador{chaves: kf, issuer: issuer, audience: audience}
}

func (a *Autenticador) Identificar(r *http.Request) (string, error) {
	bruto, err := tokenDoCabecalho(r)
	if err != nil {
		return "", ErrCredencial
	}

	t, err := jwt.Parse(bruto, a.chaves,
		jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.audience),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256", "ES256", "HS256"}),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !t.Valid {
		return "", ErrCredencial
	}

	sub, err := t.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", ErrCredencial
	}
	return sub, nil
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
