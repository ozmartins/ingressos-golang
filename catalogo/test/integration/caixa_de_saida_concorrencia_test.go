//go:build integration

package integration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
)

// Duas réplicas drenando a mesma caixa não podem tomar o mesmo fato nem esperar
// uma pela outra: é o que `FOR UPDATE SKIP LOCKED` garante e o que o princípio VI
// exige para o "ao menos uma vez" não virar "várias vezes ao mesmo tempo". Cada
// publicação demora um pouco, para que as transações se sobreponham de verdade.
func TestCaixaDeSaidaComReplicasConcorrentesNaoDuplicaNemBloqueia(t *testing.T) {
	carregarFixtures(t)
	ctx := context.Background()
	const fatos, replicas, lote = 50, 3, 10

	if _, err := pool.ExecContext(ctx, `INSERT INTO outbox_eventos (message_id, routing_key, payload)
	                             SELECT 'concorrencia-' || g, 'sessao.criada', '{}'::jsonb
	                               FROM generate_series(1, $1::int) g`, fatos); err != nil {
		t.Fatalf("enfileirando fatos: %v", err)
	}

	var (
		mu       sync.Mutex
		vistos   = map[string]int{}
		porRepl  = make([]int, replicas)
		falhas   []error
		aguardar sync.WaitGroup

		// Quantas réplicas estão, ao mesmo tempo, dentro de `publicar` — isto é,
		// com a transação e os locks abertos. Sem SKIP LOCKED a segunda réplica
		// esperaria a primeira commitar e este máximo nunca passaria de 1.
		emVoo, maxEmVoo atomic.Int32
	)
	for r := range replicas {
		aguardar.Add(1)
		go func() {
			defer aguardar.Done()
			caixa := pgadapter.NovaCaixaDeSaida(banco)
			for {
				n, err := caixa.Drenar(ctx, lote, func(f pgadapter.FatoNaCaixa) error {
					atual := emVoo.Add(1)
					for {
						m := maxEmVoo.Load()
						if atual <= m || maxEmVoo.CompareAndSwap(m, atual) {
							break
						}
					}
					time.Sleep(20 * time.Millisecond)
					emVoo.Add(-1)
					mu.Lock()
					vistos[f.MessageID]++
					mu.Unlock()
					return nil
				})
				if err != nil {
					mu.Lock()
					falhas = append(falhas, err)
					mu.Unlock()
					return
				}
				if n == 0 {
					return
				}
				mu.Lock()
				porRepl[r] += n
				mu.Unlock()
			}
		}()
	}
	aguardar.Wait()

	for _, err := range falhas {
		t.Errorf("Drenar: %v", err)
	}
	// Quem chegou depois de as outras pegarem tudo sai com zero; o resto é
	// recolhido aqui, como o próximo tique do publicador faria.
	if _, err := pgadapter.NovaCaixaDeSaida(banco).Drenar(ctx, fatos, func(f pgadapter.FatoNaCaixa) error {
		mu.Lock()
		vistos[f.MessageID]++
		mu.Unlock()
		return nil
	}); err != nil {
		t.Fatalf("drenagem final: %v", err)
	}

	if len(vistos) != fatos {
		t.Errorf("foram publicados %d fatos distintos, esperava %d", len(vistos), fatos)
	}
	for id, vezes := range vistos {
		if vezes != 1 {
			t.Errorf("o fato %s foi entregue %d vezes, esperava 1", id, vezes)
		}
	}

	// Sem bloqueio: mais de uma réplica esteve publicando ao mesmo tempo, cada uma
	// com o seu lote.
	if m := maxEmVoo.Load(); m < 2 {
		t.Errorf("no máximo %d réplica publicou por vez (%v por réplica): uma esperou a outra", m, porRepl)
	}
}
