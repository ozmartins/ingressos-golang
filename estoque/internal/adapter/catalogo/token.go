package catalogo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// FonteDeToken obtém e guarda em cache o token de service account do estoque
// (OAuth2 client credentials, emitido pelo Keycloak). Cada chamada ao catálogo
// leva esse token: o catálogo sabe quem chama e o que essa identidade pode
// fazer. Implementa credentials.PerRPCCredentials do gRPC.
type FonteDeToken struct {
	url      string
	clientID string
	segredo  string
	http     *http.Client
	// ExigeTLS faz o gRPC recusar enviar o token em canal sem TLS.
	ExigeTLS bool
	margem   time.Duration // renova com esta antecedência do vencimento
	agora    func() time.Time

	mu     sync.Mutex
	token  string
	expira time.Time
}

// NovaFonteDeToken cria a fonte. O cliente HTTP tem timeout explícito.
func NovaFonteDeToken(tokenURL, clientID, segredo string, timeout time.Duration) *FonteDeToken {
	return &FonteDeToken{
		url: tokenURL, clientID: clientID, segredo: segredo,
		http:   &http.Client{Timeout: timeout},
		margem: 30 * time.Second,
		agora:  time.Now,
	}
}

// Token devolve um access token válido, renovando-o se estiver perto de vencer.
func (f *FonteDeToken) Token(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.token != "" && f.agora().Add(f.margem).Before(f.expira) {
		return f.token, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {f.clientID},
		"client_secret": {f.segredo},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.url, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("montar requisição de token: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("obter token do Keycloak: %w", err)
	}
	defer resp.Body.Close()
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("ler resposta de token: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("obter token do Keycloak: status %d", resp.StatusCode)
	}

	var r struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(corpo, &r); err != nil || r.AccessToken == "" {
		return "", errors.New("resposta de token do Keycloak inválida")
	}
	f.token = r.AccessToken
	f.expira = f.agora().Add(time.Duration(r.ExpiresIn) * time.Second)
	return f.token, nil
}

// GetRequestMetadata implementa credentials.PerRPCCredentials.
func (f *FonteDeToken) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	t, err := f.Token(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]string{"authorization": "Bearer " + t}, nil
}

// RequireTransportSecurity é true quando o canal usa TLS (ExigeTLS): assim o
// token nunca sai em texto puro. Sem TLS (só dev/testes) fica false.
func (f *FonteDeToken) RequireTransportSecurity() bool { return f.ExigeTLS }
