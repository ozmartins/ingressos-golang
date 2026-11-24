package ratelimit

import (
	"net/http"
	"testing"
	"time"
)

func TestPermitirEsgotaORajadaEReporComOTempo(t *testing.T) {
	agora := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := Novo(1, 2)
	l.agora = func() time.Time { return agora }

	if !l.Permitir("a") || !l.Permitir("a") {
		t.Fatal("a rajada de 2 deveria passar")
	}
	if l.Permitir("a") {
		t.Fatal("a 3ª requisição imediata deveria ser barrada")
	}
	if !l.Permitir("b") {
		t.Fatal("outra chave não deve ser afetada")
	}
	agora = agora.Add(1100 * time.Millisecond)
	if !l.Permitir("a") {
		t.Fatal("após 1s uma ficha deveria ter sido reposta")
	}
}

func TestPermitirDescartaEntradasOciosas(t *testing.T) {
	agora := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := Novo(1, 1)
	l.agora = func() time.Time { return agora }
	l.Permitir("a")
	agora = agora.Add(10 * time.Minute)
	l.Permitir("b")
	if _, ok := l.entradas["a"]; ok {
		t.Error("entrada ociosa deveria ter sido descartada")
	}
}

func TestLimitadorNilNaoLimita(t *testing.T) {
	var l *Limitador
	if !l.Permitir("x") {
		t.Fatal("nil deveria permitir")
	}
}

func TestIPDeIgnoraXForwardedFor(t *testing.T) {
	r := &http.Request{RemoteAddr: "10.0.0.7:5555", Header: http.Header{"X-Forwarded-For": {"1.2.3.4"}}}
	if got := IPDe(r); got != "10.0.0.7" {
		t.Errorf("IP = %q", got)
	}
}
