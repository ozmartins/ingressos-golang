package postgres

import (
	"context"
	"time"
)

// RotinaPeriodica roda `fn` na largada e a cada intervalo, até o contexto
// encerrar. É a mesma função do Servico-Estoque: duplicá-la custa menos que
// criar um pacote compartilhado entre dois módulos Go independentes, e ela é
// curta o suficiente para que a duplicação não esconda comportamento.
//
// Uma falha não interrompe a rotina — ela é registrada e o próximo tique tenta
// de novo, porque o que a rotina persegue é estado durável no banco.
func RotinaPeriodica(ctx context.Context, nome string, intervalo time.Duration,
	fn func(context.Context) error, aoFalhar func(string, error)) {
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
