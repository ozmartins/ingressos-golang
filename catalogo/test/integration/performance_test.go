//go:build integration

package integration

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

func carregarVolume(t *testing.T, fator int) {
	t.Helper()
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `TRUNCATE sessoes, salas, cinemas, filmes CASCADE`); err != nil {
		t.Fatal(err)
	}
	cargas := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO filmes (id, titulo, duracao_minutos, classificacao_etaria, genero, status)
		  SELECT gen_random_uuid()::text, 'Filme ' || lpad(i::text, 6, '0'), 90 + (i % 90),
		         'Livre', 'Drama', CASE WHEN i % 10 = 0 THEN 'FORA_DE_CARTAZ' ELSE 'EM_CARTAZ' END
		  FROM generate_series(1, $1) AS i`, []any{500 * fator}},

		{`INSERT INTO cinemas (id, nome, cidade, estado, endereco)
		  SELECT gen_random_uuid()::text, 'Cinema ' || lpad(i::text, 4, '0'), 'Cidade', 'SC', 'Rua X'
		  FROM generate_series(1, $1) AS i`, []any{50 * fator}},

		{`INSERT INTO salas (id, cinema_id, numero, tipo_tela, capacidade_total)
		  SELECT gen_random_uuid()::text, c.id, s.n, '2D', 100
		  FROM cinemas c, generate_series(1, $1) AS s(n)`, []any{6}},

		{`INSERT INTO sessoes (id, filme_id, sala_id, data_hora_inicio, idioma, preco_base, status)
		  SELECT gen_random_uuid()::text, f.id, sa.id,
		         TIMESTAMPTZ '2026-09-01 12:00:00+00' + (i % 30) * INTERVAL '1 day' + (i % 12) * INTERVAL '1 hour',
		         CASE WHEN i % 2 = 0 THEN 'DUBLADO' ELSE 'LEGENDADO' END,
		         30.00 + (i % 40), 'AGENDADA'
		  FROM generate_series(1, $1) AS i
		  CROSS JOIN LATERAL (SELECT id FROM filmes OFFSET (i % (SELECT COUNT(*) FROM filmes)) LIMIT 1) AS f
		  CROSS JOIN LATERAL (SELECT id FROM salas  OFFSET (i % (SELECT COUNT(*) FROM salas))  LIMIT 1) AS sa`,
			[]any{5000 * fator}},
	}

	for _, c := range cargas {
		if _, err := pool.Exec(ctx, c.sql, c.args...); err != nil {
			t.Fatalf("carregando volume (fator %d): %v", fator, err)
		}
	}
}

func medirP95(t *testing.T, n int, consulta func()) time.Duration {
	t.Helper()
	amostras := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		inicio := time.Now()
		consulta()
		amostras = append(amostras, time.Since(inicio))
	}
	sort.Slice(amostras, func(i, j int) bool { return amostras[i] < amostras[j] })
	return amostras[int(float64(len(amostras))*0.95)]
}

func TestDesempenhoDasConsultasPaginadas(t *testing.T) {
	if testing.Short() {
		t.Skip("teste de volume")
	}
	ctx := context.Background()

	casos := []struct {
		fator  int
		limite time.Duration
	}{
		{1, time.Second},
		{10, 2 * time.Second},
	}

	for _, c := range casos {
		t.Run(fmt.Sprintf("fator-%d", c.fator), func(t *testing.T) {
			carregarVolume(t, c.fator)

			filmes := pgadapter.NovoFilmeRepository(pool)
			cinemas := pgadapter.NovoCinemaRepository(pool)
			salas := pgadapter.NovoSalaRepository(pool)
			sessoes := pgadapter.NovoSessaoRepository(pool)

			var cinemaID string
			if err := pool.QueryRow(ctx, `SELECT id FROM cinemas LIMIT 1`).Scan(&cinemaID); err != nil {
				t.Fatal(err)
			}
			req := pagina(t, 1, 20)

			consultas := map[string]func(){
				"filmes":  func() { _, _ = filmes.Listar(ctx, usecase.FiltroFilmes{}, catalogo.StatusPublicos, req) },
				"cinemas": func() { _, _ = cinemas.Listar(ctx, req) },
				"salas":   func() { _, _ = salas.ListarPorCinema(ctx, cinemaID, req) },
				"sessoes": func() { _, _ = sessoes.Consultar(ctx, usecase.FiltroSessoes{}, req) },
				"sessoes-filtradas": func() {
					_, _ = sessoes.Consultar(ctx, usecase.FiltroSessoes{
						CinemaID: cinemaID, Data: &usecase.DataDoDia{Ano: 2026, Mes: 9, Dia: 5},
					}, req)
				},
			}
			for nome, consulta := range consultas {
				p95 := medirP95(t, 40, consulta)
				t.Logf("fator %2d | %-18s p95=%v", c.fator, nome, p95.Round(time.Microsecond))
				if p95 > c.limite {
					t.Errorf("%s: p95=%v acima do limite de %v (SC-003)", nome, p95, c.limite)
				}
			}
		})
	}
}
