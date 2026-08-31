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

// ErrCredencial cobre toda recusa de autenticação. É uma só de propósito: a
// resposta 401 não diferencia token ausente, expirado ou mal assinado
// (contracts/erros.md §2).
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

// Identificar devolve o `sub` do token, que é a identidade que recorta a
// listagem (FR-014).
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

// ChavePortaria valida a credencial do dispositivo de portaria. É um esquema
// separado do token da pessoa, sem hierarquia entre os dois: a rota de
// validação não olha Authorization, e a de listagem não olha X-API-Key
// (research.md D7).
type ChavePortaria struct{ esperada []byte }

// NovaChavePortaria exige chave não vazia.
func NovaChavePortaria(chave string) (*ChavePortaria, error) {
	if chave == "" {
		return nil, errors.New("http: chave de portaria vazia")
	}
	return &ChavePortaria{esperada: []byte(chave)}, nil
}

// Autorizar devolve ErrCredencial para chave ausente e para chave errada — a
// mesma recusa, indistinguível (FR-012).
//
// A comparação é em tempo constante: comparação ingênua vazaria o prefixo
// correto pelo tempo de resposta.
func (c *ChavePortaria) Autorizar(r *http.Request) error {
	apresentada := []byte(r.Header.Get("X-API-Key"))
	if subtle.ConstantTimeCompare(apresentada, c.esperada) != 1 {
		return ErrCredencial
	}
	return nil
}
