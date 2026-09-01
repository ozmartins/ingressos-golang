//go:build integration

package integration

import (
	"context"
	"testing"
)

// TestSchemaProprio prova que o serviço não usa `public`: as consultas dos
// repositórios não qualificam a tabela, então é o `search_path` do pool que
// decide onde elas caem. Se alguém remover essa configuração ou desqualificar
// as migrações, as tabelas voltam para `public` e este teste falha.
func TestSchemaProprio(t *testing.T) {
	cenario := montarCenario(t, false)
	ctx := context.Background()

	// O `search_path` efetivo, não o schema resolvido: como o usuário do banco
	// também se chama `estoque`, o padrão `"$user", public` acertaria o schema
	// por coincidência de nome e esconderia a falta da configuração no pool.
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
