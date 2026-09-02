//go:build integration

package integration

import (
	"context"
	"testing"
)

func TestSchemaProprio(t *testing.T) {
	cenario := montarCenario(t, false)
	ctx := context.Background()

	var searchPath string
	if err := cenario.Pool.QueryRow(ctx, `SELECT current_setting('search_path')`).Scan(&searchPath); err != nil {
		t.Fatalf("consultando search_path: %v", err)
	}
	if searchPath != "estoque" {
		t.Errorf("search_path = %q, esperado \"estoque\"", searchPath)
	}

	tabelas := []string{
		"poltronas", "reservas", "reserva_poltronas",
		"outbox_eventos", "mensagens_processadas",
	}
	for _, tabela := range tabelas {
		var emPublic *string
		if err := cenario.Pool.QueryRow(ctx, `SELECT to_regclass('public.'|| $1)::text`, tabela).Scan(&emPublic); err != nil {
			t.Fatalf("consultando public.%s: %v", tabela, err)
		}
		if emPublic != nil {
			t.Errorf("tabela %q existe em public (%s); deveria existir só em estoque", tabela, *emPublic)
		}
	}
}
