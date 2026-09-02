package http

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
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

type ChavePortaria struct{ esperada []byte }

func NovaChavePortaria(chave string) (*ChavePortaria, error) {
	if chave == "" {
		return nil, errors.New("http: chave de portaria vazia")
	}
	return &ChavePortaria{esperada: []byte(chave)}, nil
}

func (c *ChavePortaria) Autorizar(r *http.Request) error {
	apresentada := []byte(r.Header.Get("X-API-Key"))
	if subtle.ConstantTimeCompare(apresentada, c.esperada) != 1 {
		return ErrCredencial
	}
	return nil
}
