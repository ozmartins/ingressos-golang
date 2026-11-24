// Package retry repete uma operação com backoff exponencial e jitter.
//
// Serve a dois usos: esperar uma dependência de infraestrutura que ainda está
// subindo (broker, JWKS) e re-tentar chamadas a outro serviço. Quem chama
// decide, por meio de Retentavel, quais erros fazem sentido re-tentar: erro de
// negócio (ex.: sessão inexistente) nunca deve ser repetido.
package retry

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"
)

// Politica define quantas vezes e com que pausa re-tentar.
type Politica struct {
	// Tentativas é o total de execuções (1 = sem retry; 0 é tratado como 1).
	Tentativas int
	// Base é a espera antes da 2ª tentativa; dobra a cada falha até Maximo.
	Base   time.Duration
	Maximo time.Duration
	// Retentavel diz se o erro justifica nova tentativa. Nil = todo erro.
	Retentavel func(error) bool
}

// Espera devolve a pausa após a n-ésima falha (n começa em 1): exponencial,
// limitada por Maximo, com jitter total (aleatório em [0, teto]) para que
// várias réplicas não retentem em sincronia.
func (p Politica) Espera(n int) time.Duration {
	teto := p.Base
	for i := 1; i < n && teto < p.Maximo; i++ {
		teto *= 2
	}
	if p.Maximo > 0 && teto > p.Maximo {
		teto = p.Maximo
	}
	if teto <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(teto) + 1))
}

// Executar roda f até ter sucesso, esgotar as tentativas, f devolver um erro
// não retentável ou ctx ser cancelado. Devolve o último erro de f.
func Executar(ctx context.Context, nome string, p Politica, f func() error) error {
	total := max(p.Tentativas, 1)
	var err error
	for n := 1; n <= total; n++ {
		if err = f(); err == nil {
			return nil
		}
		if n == total || ctx.Err() != nil || (p.Retentavel != nil && !p.Retentavel(err)) {
			return err
		}
		espera := p.Espera(n)
		slog.Warn("tentativa falhou, vai re-tentar", "operacao", nome,
			"tentativa", n, "de", total, "espera", espera.String(), "erro", err)
		select {
		case <-time.After(espera):
		case <-ctx.Done():
			return err
		}
	}
	return err
}

// Dependencia é a política para esperar infraestrutura na largada: ~1 min no
// total, o bastante para o broker ou o Keycloak terminarem de subir.
var Dependencia = Politica{Tentativas: 12, Base: 500 * time.Millisecond, Maximo: 8 * time.Second}
