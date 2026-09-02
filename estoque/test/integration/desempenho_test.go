//go:build integration

package integration

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
)

func percentil(amostras []time.Duration, p float64) time.Duration {
	if len(amostras) == 0 {
		return 0
	}
	ordenadas := append([]time.Duration(nil), amostras...)
	sort.Slice(ordenadas, func(i, j int) bool { return ordenadas[i] < ordenadas[j] })
	indice := int(float64(len(ordenadas)-1) * p)
	return ordenadas[indice]
}

func TestDesempenhoDoBloqueio(t *testing.T) {
	c := montarCenario(t, false)
	ctx := context.Background()

	fileiras := make([]string, 0, 25)
	for i := 0; i < 25; i++ {
		fileiras = append(fileiras, string(rune('A'+i)))
	}
	sessao := c.novaSessao(t, fileiras, 20)

	var amostras []time.Duration
	for i, fileira := range fileiras {
		for n := 1; n <= 20; n++ {
			rotulo := poltrona.MontarRotulo(fileira, n)

			inicio := time.Now()
			resultado, err := c.Bloquear.Executar(ctx, sessao, usuario, []string{rotulo})
			decorrido := time.Since(inicio)

			if err != nil {
				t.Fatalf("bloqueio de %s: %v", rotulo, err)
			}
			if !resultado.Concedido {
				t.Fatalf("bloqueio de %s recusado em sala vazia", rotulo)
			}
			amostras = append(amostras, decorrido)
		}
		_ = i
	}

	p99 := percentil(amostras, 0.99)
	mediana := percentil(amostras, 0.50)
	t.Logf("bloqueio: n=%d mediana=%v p99=%v", len(amostras), mediana, p99)

	if p99 > 100*time.Millisecond {
		t.Errorf("p99 do bloqueio = %v, orçamento é 100ms (SC-001)", p99)
	}
}

func TestDesempenhoDaConsulta(t *testing.T) {
	c := montarCenario(t, false)
	ctx := context.Background()

	fileiras := make([]string, 0, 25)
	for i := 0; i < 25; i++ {
		fileiras = append(fileiras, string(rune('A'+i)))
	}
	sessao := c.novaSessao(t, fileiras, 20)

	for i := 0; i < 250; i++ {
		rotulo := poltrona.MontarRotulo(fileiras[i/20], i%20+1)
		if _, err := c.Bloquear.Executar(ctx, sessao, usuario, []string{rotulo}); err != nil {
			t.Fatalf("preparar sala: %v", err)
		}
	}

	var amostras []time.Duration
	for i := 0; i < 100; i++ {
		inicio := time.Now()
		mapa, err := c.Consultar.Executar(ctx, sessao)
		amostras = append(amostras, time.Since(inicio))

		if err != nil {
			t.Fatalf("consulta: %v", err)
		}
		if len(mapa) != 500 {
			t.Fatalf("mapa = %d poltronas, esperado 500", len(mapa))
		}
	}

	p99 := percentil(amostras, 0.99)
	t.Logf("consulta do mapa (500 lugares): mediana=%v p99=%v", percentil(amostras, 0.50), p99)

	if p99 > 200*time.Millisecond {
		t.Errorf("p99 da consulta = %v, orçamento é 200ms (SC-013)", p99)
	}
}
