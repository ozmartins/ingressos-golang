//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

const leiturasDePico = 60

func TestVeredictoDaPortariaSobPico(t *testing.T) {
	a := subirAmbiente(t)
	u := a.validador()

	codigos := make([]string, 0, leiturasDePico)
	for k := 0; k < leiturasDePico; k++ {
		codigos = append(codigos, a.emitir(t).CodigoQR)
	}

	duracoes := make([]time.Duration, leiturasDePico)
	pronto := make(chan struct{})
	var wg sync.WaitGroup
	for k, c := range codigos {
		wg.Add(1)
		go func(idx int, codigo string) {
			defer wg.Done()
			<-pronto
			inicio := time.Now()
			r, err := u.Executar(context.Background(), codigo)
			duracoes[idx] = time.Since(inicio)
			if err != nil {
				t.Errorf("validação %d falhou: %v", idx, err)
			}
			if r.Veredito != usecase.Autorizada {
				t.Errorf("validação %d = %v, queria Autorizada", idx, r.Veredito)
			}
		}(k, c)
	}
	close(pronto)
	wg.Wait()

	p99 := percentil(duracoes, 0.99)
	t.Logf("veredito da portaria sob %d leituras simultâneas: p99 = %s", leiturasDePico, p99)
	if p99 > time.Second {
		t.Errorf("p99 = %s, queria menos de 1s (SC-003)", p99)
	}
}

func TestIngressoFicaConsultavelDentroDoPrazo(t *testing.T) {
	a := subirAmbiente(t)
	c := a.consumidor(t, false)

	ctx, parar := context.WithCancel(context.Background())
	defer parar()
	go func() { _ = c.Consumir(ctx) }()

	const amostras = 20
	duracoes := make([]time.Duration, 0, amostras)
	listagem := usecase.ListarIngressos{Ingressos: a.Ingressos}

	for k := 0; k < amostras; k++ {
		usuario := uuid.NewString()
		inicio := time.Now()
		a.publicar(t, uuid.NewString(), usuario)

		esperar(t, 15*time.Second, "o ingresso ficar consultável", func() bool {
			lista, err := listagem.Executar(context.Background(), usuario, "")
			return err == nil && len(lista) == 1
		})
		duracoes = append(duracoes, time.Since(inicio))
	}

	p95 := percentil(duracoes, 0.95)
	t.Logf("do anúncio ao ingresso consultável: p95 = %s", p95)
	if p95 > 5*time.Second {
		t.Errorf("p95 = %s, queria menos de 5s (SC-002)", p95)
	}
}
