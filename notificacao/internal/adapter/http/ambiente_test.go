package http

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/health"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

const (
	emissor  = "http://localhost:8081/realms/cinema"
	publico  = "conta-cinema"
	chaveAPI = "chave-de-portaria-de-teste"
	usuario1 = "c394c8b3-76a1-4328-b803-02f5923b7a15"
	usuario2 = "00000000-0000-4000-8000-000000000000"
)

var instanteFixo = time.Date(2026, 8, 29, 21, 35, 12, 0, time.UTC)

type relogioFixo struct{}

func (relogioFixo) Agora() time.Time { return instanteFixo }

type assinadorFalso struct{}

func (assinadorFalso) Gerar(id string) string { return "CIN1." + id + ".assinatura" }
func (assinadorFalso) Verificar(c string) (string, error) {
	partes := strings.Split(c, ".")
	if len(partes) != 3 || partes[0] != "CIN1" || partes[2] != "assinatura" || partes[1] == "" {
		return "", usecase.ErrNaoEncontrado
	}
	return partes[1], nil
}

type repoMemoria struct {
	mu    sync.Mutex
	porID map[string]ingresso.Ingresso
}

func novoRepo() *repoMemoria { return &repoMemoria{porID: map[string]ingresso.Ingresso{}} }

func (r *repoMemoria) CriarSeAusente(context.Context, ingresso.Ingresso) (bool, ingresso.Ingresso, error) {
	return false, ingresso.Ingresso{}, nil
}

func (r *repoMemoria) Utilizar(_ context.Context, id string, agora time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.porID[id]
	if !ok || i.Status != ingresso.Valido {
		return false, nil
	}
	u, err := i.Utilizar(agora)
	if err != nil {
		return false, nil
	}
	r.porID[id] = u
	return true, nil
}

func (r *repoMemoria) BuscarPorID(_ context.Context, id string) (ingresso.Ingresso, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.porID[id]
	if !ok {
		return ingresso.Ingresso{}, usecase.ErrNaoEncontrado
	}
	return i, nil
}

func (r *repoMemoria) ListarPorUsuario(_ context.Context, usuarioID string, filtro ingresso.Status) ([]ingresso.Ingresso, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []ingresso.Ingresso
	for _, i := range r.porID {
		if i.UsuarioID != usuarioID || (filtro != "" && i.Status != filtro) {
			continue
		}
		out = append(out, i)
	}
	for a := 1; a < len(out); a++ {
		for b := a; b > 0 && menor(out[b-1], out[b]); b-- {
			out[b-1], out[b] = out[b], out[b-1]
		}
	}
	return out, nil
}

func menor(x, y ingresso.Ingresso) bool {
	if x.CriadoEm.Equal(y.CriadoEm) {
		return x.ID < y.ID
	}
	return x.CriadoEm.Before(y.CriadoEm)
}

func (r *repoMemoria) semear(t *testing.T, id, usuarioID string, status ingresso.Status, criadoEm time.Time) ingresso.Ingresso {
	t.Helper()
	i, err := ingresso.Novo(id, "res-"+id, usuarioID, assinadorFalso{}.Gerar(id), criadoEm)
	if err != nil {
		t.Fatalf("preparação: %v", err)
	}
	switch status {
	case ingresso.Utilizado:
		i, _ = i.Utilizar(criadoEm.Add(time.Hour))
	case ingresso.Cancelado:
		i, _ = i.Cancelar()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.porID[id] = i
	return i
}

type ambiente struct {
	srv   *httptest.Server
	repo  *repoMemoria
	chave *rsa.PrivateKey
}

func montarAmbiente(t *testing.T) *ambiente {
	t.Helper()
	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gerar chave: %v", err)
	}
	kf := func(*jwt.Token) (any, error) { return &chave.PublicKey, nil }

	repo := novoRepo()
	portaria, err := NovaChavePortaria(chaveAPI)
	if err != nil {
		t.Fatalf("chave de portaria: %v", err)
	}
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))

	api := &API{
		Listagem:  usecase.ListarIngressos{Ingressos: repo},
		Validacao: usecase.ValidarIngresso{Ingressos: repo, Assinador: assinadorFalso{}, Relogio: relogioFixo{}, Log: log},
		Auth:      NovoAutenticadorComChave(kf, emissor, publico),
		Chave:     portaria,
		Prontidao: health.NovaProntidao(),
		Log:       log,
	}
	srv := httptest.NewServer(api.Rotas())
	t.Cleanup(srv.Close)
	return &ambiente{srv: srv, repo: repo, chave: chave}
}

func (a *ambiente) token(t *testing.T, sub string, ajustes ...func(jwt.MapClaims)) string {
	t.Helper()
	c := jwt.MapClaims{
		"sub": sub, "iss": emissor, "aud": publico,
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	for _, f := range ajustes {
		f(c)
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodRS256, c).SignedString(a.chave)
	if err != nil {
		t.Fatalf("assinar token: %v", err)
	}
	return s
}

func (a *ambiente) get(t *testing.T, caminho string, cab map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, a.srv.URL+caminho, nil)
	if err != nil {
		t.Fatalf("montar requisição: %v", err)
	}
	return a.enviar(t, req, cab)
}

func (a *ambiente) postValidar(t *testing.T, corpo string, cab map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, a.srv.URL+"/api/v1/ingressos/validar", strings.NewReader(corpo))
	if err != nil {
		t.Fatalf("montar requisição: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return a.enviar(t, req, cab)
}

func (a *ambiente) enviar(t *testing.T, req *http.Request, cab map[string]string) (*http.Response, []byte) {
	t.Helper()
	for k, v := range cab {
		req.Header.Set(k, v)
	}
	res, err := a.srv.Client().Do(req)
	if err != nil {
		t.Fatalf("enviar: %v", err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("ler corpo: %v", err)
	}
	return res, b
}

func decodificarLista(t *testing.T, b []byte) []map[string]any {
	t.Helper()
	var lista []map[string]any
	if err := json.Unmarshal(b, &lista); err != nil {
		t.Fatalf("decodificar lista: %v (corpo: %s)", err, b)
	}
	return lista
}

func decodificarEm(t *testing.T, b []byte, destino any) {
	t.Helper()
	if err := json.Unmarshal(b, destino); err != nil {
		t.Fatalf("decodificar: %v (corpo: %s)", err, b)
	}
}
