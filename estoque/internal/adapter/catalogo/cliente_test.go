package catalogo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const corpoValido = `{"id":"s1","filme_id":"f","sala_id":"x","data_hora_inicio":"2026-09-20T19:30:00Z","idioma":"DUBLADO","preco_base":"42.00","status":"AGENDADA"}`

func servidor(t *testing.T, h http.HandlerFunc) *Cliente {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NovoCliente(Opcoes{URLBase: srv.URL, Timeout: 200 * time.Millisecond, FalhasParaAbrir: 2, IntervaloAberto: time.Hour})
}

func TestBuscarSessaoLeOsCamposDoCatalogo(t *testing.T) {
	var caminho string
	c := servidor(t, func(w http.ResponseWriter, r *http.Request) {
		caminho = r.URL.Path
		_, _ = w.Write([]byte(corpoValido))
	})

	s, err := c.BuscarSessao(context.Background(), "s1")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if caminho != "/api/v1/sessoes/s1" {
		t.Errorf("caminho = %q", caminho)
	}
	if s.Status != "AGENDADA" || s.PrecoBase != "42.00" || s.Inicio.Format(time.RFC3339) != "2026-09-20T19:30:00Z" {
		t.Errorf("sessão = %+v", s)
	}
}

func TestBuscarSessaoTraduzErros(t *testing.T) {
	casos := map[string]struct {
		h        http.HandlerFunc
		esperado error
	}{
		"404":               {func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }, shared.ErrSessaoNaoEncontrada},
		"503":               {func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(503) }, shared.ErrCatalogoIndisponivel},
		"400":               {func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(400) }, shared.ErrRespostaInvalidaDoCatalogo},
		"json quebrado":     {func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{")) }, shared.ErrRespostaInvalidaDoCatalogo},
		"sessão incompleta": {func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"status":"AGENDADA"}`)) }, shared.ErrRespostaInvalidaDoCatalogo},
		"lento":             {func(w http.ResponseWriter, _ *http.Request) { time.Sleep(500 * time.Millisecond) }, shared.ErrCatalogoIndisponivel},
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := servidor(t, c.h).BuscarSessao(context.Background(), "s1")
			if !errors.Is(err, c.esperado) {
				t.Errorf("err = %v, esperado %v", err, c.esperado)
			}
		})
	}
}

func TestBreakerAbreComFalhasConsecutivasENaoComSessaoInexistente(t *testing.T) {
	var chamadas atomic.Int32
	status := http.StatusNotFound
	c := servidor(t, func(w http.ResponseWriter, _ *http.Request) {
		chamadas.Add(1)
		w.WriteHeader(status)
	})

	for range 5 {
		_, _ = c.BuscarSessao(context.Background(), "s1")
	}
	if chamadas.Load() != 5 {
		t.Fatalf("404 não deve abrir o breaker; chamadas = %d", chamadas.Load())
	}

	status = http.StatusInternalServerError
	for range 5 {
		_, err := c.BuscarSessao(context.Background(), "s1")
		if !errors.Is(err, shared.ErrCatalogoIndisponivel) {
			t.Fatalf("err = %v, esperado ErrCatalogoIndisponivel", err)
		}
	}
	if got := chamadas.Load(); got != 7 {
		t.Errorf("chamadas = %d, esperado 7 (5 + 2 falhas até abrir; o resto recusado sem rede)", got)
	}
}
