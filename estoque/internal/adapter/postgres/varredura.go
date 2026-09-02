package postgres

import (
	"context"
	"time"
)

func RotinaPeriodica(ctx context.Context, nome string, intervalo time.Duration, fn func(context.Context) error, aoFalhar func(string, error)) {
	go func() {
		tique := time.NewTicker(intervalo)
		defer tique.Stop()

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
