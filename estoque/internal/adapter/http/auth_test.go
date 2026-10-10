package http

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// O algoritmo vem do token; aceitar HS256 permitiria validar a assinatura com
// um segredo simétrico em vez da chave pública do emissor.
func TestIdentificarRecusaAlgoritmoDiferenteDeRS256(t *testing.T) {
	segredo := []byte("segredo-simetrico")
	auth := NovoAutenticadorComChave(func(*jwt.Token) (any, error) { return segredo, nil }, "iss", "aud")

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "iss", "aud": "aud", "sub": "u1", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(segredo)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)

	if _, err := auth.Identificar(r); err == nil {
		t.Fatal("token HS256 devia ser recusado")
	}
}
