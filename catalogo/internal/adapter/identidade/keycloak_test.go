package identidade

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

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
		pub := chave.PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "chave-1", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}}})
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
	tk := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(claims))
	tk.Header["kid"] = "chave-1"
	assinado, err := tk.SignedString(chave)
	if err != nil {
		t.Fatal(err)
	}
	return assinado
}

func (e *emissorDeTeste) verificador(t *testing.T) *Verificador {
	t.Helper()
	k, err := keyfunc.NewDefault([]string{e.server.URL + "/jwks"})
	if err != nil {
		t.Fatal(err)
	}
	return NovoVerificadorComChave(k.Keyfunc, e.issuer, "cinema-app")
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
	id, err := e.verificador(t).Verificar(context.Background(), e.token(t, claimsValidas(e.issuer), nil))
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
	if _, err := e.verificador(t).Verificar(context.Background(), e.token(t, c, nil)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de token expirado, obteve %v", err)
	}
}

func TestVerificarRecusaAssinaturaDeOutraChave(t *testing.T) {
	e := novoEmissor(t)
	outra, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.verificador(t).Verificar(context.Background(), e.token(t, claimsValidas(e.issuer), outra)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de assinatura inválida, obteve %v", err)
	}
}

func TestVerificarRecusaEmissorDesconhecido(t *testing.T) {
	e := novoEmissor(t)
	c := claimsValidas("https://emissor-que-nao-confiamos.example")
	if _, err := e.verificador(t).Verificar(context.Background(), e.token(t, c, nil)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de emissor desconhecido, obteve %v", err)
	}
}

func TestVerificarRecusaAudienciaErrada(t *testing.T) {
	e := novoEmissor(t)
	c := claimsValidas(e.issuer)
	c["aud"] = "outro-aplicativo"
	if _, err := e.verificador(t).Verificar(context.Background(), e.token(t, c, nil)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de audiência errada, obteve %v", err)
	}
}

func TestVerificarRecusaTokenSemSub(t *testing.T) {
	e := novoEmissor(t)
	c := claimsValidas(e.issuer)
	delete(c, "sub")
	_, err := e.verificador(t).Verificar(context.Background(), e.token(t, c, nil))
	if !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de token sem sub, obteve %v", err)
	}
}

func TestVerificarRecusaTokenMalformado(t *testing.T) {
	e := novoEmissor(t)
	if _, err := e.verificador(t).Verificar(context.Background(), "isto.nao.e-um-jwt"); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de token malformado, obteve %v", err)
	}
}

func TestVerificarAceitaTokenDeServiceAccount(t *testing.T) {
	e := novoEmissor(t)
	const subDaServiceAccount = "service-account-cinema-m2m-0000-000000000001"

	c := map[string]any{
		"iss": e.issuer,
		"aud": "cinema-app",
		"azp": "cinema-m2m",
		"typ": "Bearer",
		"sub": subDaServiceAccount,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	id, err := e.verificador(t).Verificar(context.Background(), e.token(t, c, nil))
	if err != nil {
		t.Fatalf("token de service account deveria ser aceito: %v", err)
	}
	if id.UsuarioID != subDaServiceAccount {
		t.Fatalf("usuario_id = %q, esperava o sub da service account", id.UsuarioID)
	}
}

func TestVerificarRecusaTokenM2MSemMapperDeAudiencia(t *testing.T) {
	e := novoEmissor(t)
	c := claimsValidas(e.issuer)
	c["aud"] = "account"
	c["azp"] = "cinema-m2m"

	if _, err := e.verificador(t).Verificar(context.Background(), e.token(t, c, nil)); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("esperava recusa de token com audiência 'account', obteve %v", err)
	}
}

func TestVerificarExtraiPapeisEEscopos(t *testing.T) {
	e := novoEmissor(t)
	v := e.verificador(t)
	c := claimsValidas(e.issuer)
	c["realm_access"] = map[string]any{"roles": []string{"catalogo-admin", "offline_access"}}
	c["scope"] = "profile catalogo-admin leitura"

	id, err := v.Verificar(context.Background(), e.token(t, c, nil))
	if err != nil {
		t.Fatal(err)
	}
	if !id.Tem("catalogo-admin") || !id.Tem("leitura") || id.Tem("outra") {
		t.Errorf("permissões = %v", id.Permissoes)
	}
	if n := len(id.Permissoes); n != 4 { // catalogo-admin, offline_access, profile, leitura (sem repetir)
		t.Errorf("esperava 4 permissões distintas, veio %d: %v", n, id.Permissoes)
	}
}

func TestVerificarTokenSemPapeisNaoTemPermissao(t *testing.T) {
	e := novoEmissor(t)
	id, err := e.verificador(t).Verificar(context.Background(), e.token(t, claimsValidas(e.issuer), nil))
	if err != nil {
		t.Fatal(err)
	}
	if id.Tem(PermissaoEscrita) {
		t.Error("token sem papéis não deveria ter permissão de escrita")
	}
}

// O JWKS pode não estar de pé na largada: NovoVerificador deve esperar com
// retry (e não subir com o conjunto de chaves vazio) até o Keycloak responder.
func TestNovoVerificadorEsperaOJWKSFicarDisponivel(t *testing.T) {
	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var chamadas atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc(caminhoJWKS, func(w http.ResponseWriter, _ *http.Request) {
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

	v, err := NovoVerificador(srv.URL, "cinema-app")
	if err != nil {
		t.Fatalf("deveria ter esperado o JWKS: %v", err)
	}
	if n := chamadas.Load(); n < 3 {
		t.Fatalf("esperava ao menos 3 buscas (2 falhas + 1 sucesso), houve %d", n)
	}
	// As chaves vieram de verdade: um token assinado por ela é aceito.
	e := &emissorDeTeste{chave: chave, issuer: srv.URL}
	if _, err := v.Verificar(context.Background(), e.token(t, claimsValidas(srv.URL), nil)); err != nil {
		t.Fatalf("token válido recusado após a espera: %v", err)
	}
}
