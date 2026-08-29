package identidade

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
)

// emissorDeTeste é um provedor de JWKS local. Existe para exercitar a
// verificação sem depender de um Keycloak em pé — a lógica sob teste é a nossa,
// não a do provedor.
type emissorDeTeste struct {
	chave  *rsa.PrivateKey
	server *httptest.Server
	issuer string
}

func novoEmissor(t *testing.T) *emissorDeTeste {
	t.Helper()
	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	e := &emissorDeTeste{chave: chave}

	mux := http.NewServeMux()
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: chave.Public(), KeyID: "chave-1", Algorithm: "RS256", Use: "sig",
		}}}
		_ = json.NewEncoder(w).Encode(jwks)
	})
	e.server = httptest.NewServer(mux)
	e.issuer = e.server.URL
	t.Cleanup(e.server.Close)
	return e
}

func (e *emissorDeTeste) token(t *testing.T, claims map[string]any, chave *rsa.PrivateKey) string {
	t.Helper()
	if chave == nil {
		chave = e.chave
	}
	assinador, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: chave},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "chave-1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	corpo, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	assinado, err := assinador.Sign(corpo)
	if err != nil {
		t.Fatal(err)
	}
	compacto, err := assinado.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return compacto
}

func (e *emissorDeTeste) verificador() *Verificador {
	return NovoVerificadorComKeySet(oidc.NewRemoteKeySet(context.Background(), e.server.URL+"/jwks"), e.issuer, "cinema-app")
}

func claimsValidas(issuer string) map[string]any {
	return map[string]any{
		"iss": issuer,
		"aud": "cinema-app",
		"sub": "9982a1b3-44c1-4221-a123-902183120192",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
}

func TestVerificarAceitaTokenValido(t *testing.T) {
	e := novoEmissor(t)
	id, err := e.verificador().Verificar(context.Background(), e.token(t, claimsValidas(e.issuer), nil))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if id.UsuarioID != "9982a1b3-44c1-4221-a123-902183120192" {
		t.Fatalf("usuario_id veio de onde não devia: %s", id.UsuarioID)
	}
}

func TestVerificarRecusaTokenExpirado(t *testing.T) {
	e := novoEmissor(t)
	c := claimsValidas(e.issuer)
	c["exp"] = time.Now().Add(-time.Hour).Unix()
	if _, err := e.verificador().Verificar(context.Background(), e.token(t, c, nil)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de token expirado, obteve %v", err)
	}
}

func TestVerificarRecusaAssinaturaDeOutraChave(t *testing.T) {
	e := novoEmissor(t)
	outra, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.verificador().Verificar(context.Background(), e.token(t, claimsValidas(e.issuer), outra)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de assinatura inválida, obteve %v", err)
	}
}

func TestVerificarRecusaEmissorDesconhecido(t *testing.T) {
	e := novoEmissor(t)
	c := claimsValidas("https://emissor-que-nao-confiamos.example")
	if _, err := e.verificador().Verificar(context.Background(), e.token(t, c, nil)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de emissor desconhecido, obteve %v", err)
	}
}

func TestVerificarRecusaAudienciaErrada(t *testing.T) {
	e := novoEmissor(t)
	c := claimsValidas(e.issuer)
	c["aud"] = "outro-aplicativo"
	if _, err := e.verificador().Verificar(context.Background(), e.token(t, c, nil)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de audiência errada, obteve %v", err)
	}
}

// Assinatura válida, mas sem identidade: não há a quem atribuir a reserva.
func TestVerificarRecusaTokenSemSub(t *testing.T) {
	e := novoEmissor(t)
	c := claimsValidas(e.issuer)
	delete(c, "sub")
	_, err := e.verificador().Verificar(context.Background(), e.token(t, c, nil))
	if !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de token sem sub, obteve %v", err)
	}
}

func TestVerificarRecusaTokenMalformado(t *testing.T) {
	e := novoEmissor(t)
	if _, err := e.verificador().Verificar(context.Background(), "isto.nao.e-um-jwt"); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de token malformado, obteve %v", err)
	}
}
