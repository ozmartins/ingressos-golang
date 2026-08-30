package postgres

import (
	"context"
	"time"
)

// RotinaPeriodica roda fn a cada intervalo até o contexto ser cancelado.
// É o que sustenta a varredura de expiração e a limpeza de idempotência.
func RotinaPeriodica(ctx context.Context, nome string, intervalo time.Duration, fn func(context.Context) error, aoFalhar func(string, error)) {
	go func() {
		tique := time.NewTicker(intervalo)
		defer tique.Stop()

		// Uma passada imediata na largada: é ela que invalida as reservas cujo
		// prazo venceu enquanto o serviço estava fora do ar (FR-013, SC-008).
		if err := fn(ctx); err != nil {
			aoFalhar(nome, err)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-tique.C:
				if err := fn(ctx); err != nil {
					aoFalhar(nome, err)
				}
			}
		}
	}()
}
