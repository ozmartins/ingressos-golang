//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestConsultasUsamOsIndices(t *testing.T) {
	carregarVolume(t, 10)
	ctx := context.Background()

	var cinemaID string
	if err := pool.QueryRow(ctx, `SELECT id FROM cinemas LIMIT 1`).Scan(&cinemaID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `ANALYZE`); err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nome   string
		sql    string
		args   []any
		indice string
		tabela string
	}{
		{
			nome: "filmes por situação",
			sql: `SELECT id, titulo FROM filmes WHERE status = ANY($1)
			      ORDER BY titulo, id LIMIT 20 OFFSET 0`,
			args:   []any{[]string{"EM_CARTAZ", "BREVE"}},
			indice: "idx_filmes_titulo_id",
			tabela: "filmes",
		},
		{
			nome:   "salas de um cinema",
			sql:    `SELECT id, numero FROM salas WHERE cinema_id = $1 ORDER BY numero, id LIMIT 20 OFFSET 0`,
			args:   []any{cinemaID},
			indice: "idx_salas_cinema_numero_id",
			tabela: "salas",
		},
		{
			nome: "grade de sessões",
			sql: `SELECT s.id FROM sessoes s
			      JOIN filmes f ON f.id = s.filme_id
			      JOIN salas sa ON sa.id = s.sala_id
			      JOIN cinemas c ON c.id = sa.cinema_id
			      WHERE s.status = ANY($1) ORDER BY s.data_hora_inicio, s.id LIMIT 20 OFFSET 0`,
			args:   []any{[]string{"AGENDADA", "EM_ANDAMENTO"}},
			indice: "idx_sessoes_inicio_id",
			tabela: "sessoes",
		},
		{
			nome:   "grade filtrada por filme",
			sql:    `SELECT s.id FROM sessoes s WHERE s.filme_id = (SELECT id FROM filmes LIMIT 1) ORDER BY s.data_hora_inicio, s.id LIMIT 20 OFFSET 0`,
			indice: "idx_sessoes_filme_inicio",
			tabela: "sessoes",
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			plano := explicar(t, c.sql, c.args...)
			if !strings.Contains(plano, c.indice) {
				t.Errorf("a consulta não usou %s.\nPlano:\n%s", c.indice, plano)
			}
			if strings.Contains(plano, "Seq Scan on "+c.tabela) {
				t.Errorf("varredura sequencial em %s com volume alto.\nPlano:\n%s", c.tabela, plano)
			}
		})
	}
}

func TestContagemDoTotalCabeNoOrcamento(t *testing.T) {
	carregarVolume(t, 10)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `ANALYZE`); err != nil {
		t.Fatal(err)
	}

	const limite = 200 * time.Millisecond
	p95 := medirP95(t, 30, func() {
		var total int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM filmes WHERE status = ANY($1)`,
			[]string{"EM_CARTAZ", "BREVE"}).Scan(&total); err != nil {
			t.Fatal(err)
		}
	})
	t.Logf("contagem com 5.000 filmes: p95=%v", p95.Round(time.Microsecond))
	if p95 > limite {
		t.Errorf("contagem levou p95=%v, acima do orçamento de %v", p95, limite)
	}
}

func explicar(t *testing.T, sql string, args ...any) string {
	t.Helper()
	rows, err := pool.Query(context.Background(), "EXPLAIN (ANALYZE, BUFFERS) "+sql, args...)
	if err != nil {
		t.Fatalf("EXPLAIN falhou: %v", err)
	}
	defer rows.Close()

	var linhas []string
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			t.Fatal(err)
		}
		linhas = append(linhas, l)
	}
	return strings.Join(linhas, "\n")
}
