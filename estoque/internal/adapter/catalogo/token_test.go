package catalogo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func servidorDeToken(t *testing.T, status int, expiresIn int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		_ = r.ParseForm()
		if r.PostForm.Get("grant_type") != "client_credentials" || r.PostForm.Get("client_id") != "estoque-svc" ||
			r.PostForm.Get("client_secret") != "s3" {
			http.Error(w, "x", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":` + itoa(expiresIn) + `}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func itoa(n int) string {
	return string(rune('0'+n/100%10)) + string(rune('0'+n/10%10)) + string(rune('0'+n%10))
}

func TestFonteDeTokenUsaCacheERenovaPertoDoVencimento(t *testing.T) {
	srv, chamadas := servidorDeToken(t, 200, 300)
	f := NovaFonteDeToken(srv.URL, "estoque-svc", "s3", time.Second)
	agora := time.Now()
	f.agora = func() time.Time { return agora }

	md, err := f.GetRequestMetadata(context.Background())
	if err != nil || md["authorization"] != "Bearer tok" {
		t.Fatalf("md=%v err=%v", md, err)
	}
	_, _ = f.Token(context.Background())
	if chamadas.Load() != 1 {
		t.Fatalf("esperava cache (1 chamada), veio %d", chamadas.Load())
	}
	agora = agora.Add(280 * time.Second) // dentro da margem de 30s do vencimento
	_, _ = f.Token(context.Background())
	if chamadas.Load() != 2 {
		t.Fatalf("esperava renovação (2 chamadas), veio %d", chamadas.Load())
	}
}

func TestFonteDeTokenFalhaComCredencialErradaOuStatusRuim(t *testing.T) {
	srv, _ := servidorDeToken(t, 200, 300)
	if _, err := NovaFonteDeToken(srv.URL, "estoque-svc", "errado", time.Second).Token(context.Background()); err == nil {
		t.Error("esperava erro com segredo errado")
	}
	srv2, _ := servidorDeToken(t, 500, 300)
	if _, err := NovaFonteDeToken(srv2.URL, "estoque-svc", "s3", time.Second).Token(context.Background()); err == nil {
		t.Error("esperava erro com status 500")
	}
}
