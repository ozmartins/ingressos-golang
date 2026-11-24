package http

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// O JWKS pode não estar de pé na largada: NovoAutenticador deve esperar com
// retry (e não subir com o conjunto de chaves vazio) até o Keycloak responder.
func TestNovoAutenticadorEsperaOJWKSFicarDisponivel(t *testing.T) {
	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var chamadas atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/certs", func(w http.ResponseWriter, _ *http.Request) {
		if chamadas.Add(1) <= 2 {
			http.Error(w, "ainda subindo", http.StatusServiceUnavailable)
			return
		}
		pub := chave.PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "chave-1", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	auth, err := NovoAutenticador(srv.URL+"/certs", "iss", "aud")
	if err != nil {
		t.Fatalf("deveria ter esperado o JWKS: %v", err)
	}
	if n := chamadas.Load(); n < 3 {
		t.Fatalf("esperava ao menos 3 buscas (2 falhas + 1 sucesso), houve %d", n)
	}

	// As chaves vieram de verdade: um token assinado por ela é aceito.
	tk := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": "iss", "aud": "aud", "sub": "u1", "exp": time.Now().Add(time.Hour).Unix(),
	})
	tk.Header["kid"] = "chave-1"
	assinado, err := tk.SignedString(chave)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+assinado)
	if sub, err := auth.Identificar(r); err != nil || sub != "u1" {
		t.Fatalf("token válido recusado após a espera: sub=%q err=%v", sub, err)
	}
}
