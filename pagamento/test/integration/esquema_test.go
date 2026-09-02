//go:build integration

package integration

import (
	"context"
	"testing"
)

func TestSchemaProprio(t *testing.T) {
	amb := subirAmbiente(t)
	ctx := context.Background()

	var searchPath string
	if err := amb.Pool.QueryRow(ctx, `SELECT current_setting('search_path')`).Scan(&searchPath); err != nil {
		t.Fatalf("consultando search_path: %v", err)
	}
	if searchPath != "pagamento" {
		t.Errorf("search_path = %q, esperado \"pagamento\"", searchPath)
	}

	var emPublic *string
	if err := amb.Pool.QueryRow(ctx,
		`SELECT to_regclass('public.transacoes_pagamento')::text`).Scan(&emPublic); err != nil {
		t.Fatalf("consultando public.transacoes_pagamento: %v", err)
	}
	if emPublic != nil {
		t.Errorf("tabela existe em public (%s); deveria existir só em pagamento", *emPublic)
	}
}
