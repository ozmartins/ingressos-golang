//go:build integration

package integration

import (
	"context"
	"testing"
)

// TestSchemaProprio prova que o serviço não usa `public`: as consultas do
// repositório não qualificam a tabela, então é o `search_path` do pool que
// decide onde elas caem. Se alguém remover essa configuração ou desqualificar
// a migração, a tabela volta para `public` e este teste falha.
func TestSchemaProprio(t *testing.T) {
	amb := subirAmbiente(t)
	ctx := context.Background()

	// O `search_path` efetivo, não o schema resolvido: como o usuário do banco
	// também se chama `pagamento`, o padrão `"$user", public` acertaria o
	// schema por coincidência de nome e esconderia a falta da configuração no
	// pool.
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
