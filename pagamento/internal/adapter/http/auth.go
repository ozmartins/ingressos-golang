package http

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// ErrCredencial cobre toda recusa de autenticação. É uma só de propósito: a
// resposta 401 não diferencia token ausente, expirado ou mal assinado
// (contracts/erros.md).
var ErrCredencial = errors.New("http: credencial inválida")

// Autenticador valida o JWT do Keycloak de forma stateless, por JWKS: assinatura,
// emissor, público e validade. Não há introspecção — a assinatura já garante o
// que a introspecção diria, sem chamada de rede no caminho de leitura (D8).
type Autenticador struct {
	chaves   jwt.Keyfunc
	issuer   string
	audience string
}

// NovoAutenticador busca o conjunto de chaves e o mantém atualizado.
func NovoAutenticador(jwksURL, issuer, audience string) (*Autenticador, error) {
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("carregar JWKS de %s: %w", jwksURL, err)
	}
	return &Autenticador{chaves: k.Keyfunc, issuer: issuer, audience: audience}, nil
}

// NovoAutenticadorComChave monta o autenticador sobre uma função de chave
// arbitrária. Existe para os testes de contrato assinarem com chave própria.
func NovoAutenticadorComChave(kf jwt.Keyfunc, issuer, audience string) *Autenticador {
	return &Autenticador{chaves: kf, issuer: issuer, audience: audience}
}

// Identificar devolve o `sub` do token, que é a identidade comparada ao dono da
// transação (FR-017).
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
