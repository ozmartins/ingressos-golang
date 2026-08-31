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

// leiturasDePico é o volume que este teste toma por "horário de pico".
//
// A spec (SC-003) não quantifica o pico, e a T052 mandou fixar o valor aqui e
// registrar a escolha. 60 leituras simultâneas correspondem a uma sala grande
// esvaziando a fila de entrada em poucos minutos, com várias catracas
// disparando ao mesmo tempo. Se a operação real for maior, é este número que
// muda — e o teste passa a cobrar o novo patamar.
const leiturasDePico = 60

// SC-003: a portaria recebe o veredito em menos de 1 s no percentil 99, com o
// cinema em pico.
func TestVeredictoDaPortariaSobPico(t *testing.T) {
	a := subirAmbiente(t)
	u := a.validador()

	// Cada leitura tem o próprio ingresso: medimos o caminho de autorização,
	// que é o mais caro (verifica assinatura, grava a baixa e relê a linha).
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

// SC-002: o ingresso fica consultável em menos de 5 s da confirmação do
// pagamento, no percentil 95.
//
// A medição vai do instante da publicação do anúncio até o ingresso aparecer na
// listagem da pessoa — que é o que a spec promete, e não o tempo interno da
// gravação.
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
