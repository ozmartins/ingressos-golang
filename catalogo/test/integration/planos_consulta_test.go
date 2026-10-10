//go:build integration

package integration

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

func TestConsultasUsamOsIndices(t *testing.T) {
	carregarVolume(t, 10)
	ctx := context.Background()

	var cinemaID, filmeID string
	if err := pool.QueryRowContext(ctx, `SELECT id FROM cinemas LIMIT 1`).Scan(&cinemaID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRowContext(ctx, `SELECT id FROM filmes LIMIT 1`).Scan(&filmeID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx, `ANALYZE`); err != nil {
		t.Fatal(err)
	}

	captura := &capturaSQL{}
	observado, err := pgadapter.Abrir(ctx, urlBanco, func(cfg *gorm.Config) { cfg.Logger = captura })
	if err != nil {
		t.Fatal(err)
	}
	defer observado.Fechar()

	req, err := shared.NovoPageRequest(1, 20, 20, 100)
	if err != nil {
		t.Fatal(err)
	}
	publicos := []catalogo.StatusFilme{catalogo.StatusEmCartaz, catalogo.StatusBreve}

	casos := []struct {
		nome   string
		acao   func() error
		indice string
		tabela string
	}{
		{
			nome: "filmes por situação",
			acao: func() error {
				_, err := pgadapter.NovoFilmeRepository(observado).Listar(ctx, usecase.FiltroFilmes{}, publicos, req)
				return err
			},
			indice: "idx_filmes_titulo_id",
			tabela: "filmes",
		},
		{
			nome: "salas de um cinema",
			acao: func() error {
				_, err := pgadapter.NovoSalaRepository(observado).Listar(ctx, usecase.FiltroSalas{CinemaID: cinemaID}, req)
				return err
			},
			indice: "idx_salas_cinema_numero_id",
			tabela: "salas",
		},
		{
			nome: "grade de sessões",
			acao: func() error {
				_, err := pgadapter.NovoSessaoRepository(observado).Consultar(ctx, usecase.FiltroSessoes{}, req)
				return err
			},
			indice: "idx_sessoes_inicio_id",
			tabela: "sessoes",
		},
		{
			nome: "grade filtrada por filme",
			acao: func() error {
				_, err := pgadapter.NovoSessaoRepository(observado).Consultar(ctx, usecase.FiltroSessoes{FilmeID: filmeID}, req)
				return err
			},
			indice: "idx_sessoes_filme_inicio",
			tabela: "sessoes",
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			captura.limpar()
			if err := c.acao(); err != nil {
				t.Fatalf("operação do adaptador falhou: %v", err)
			}
			consulta := captura.paginaDaListagem(t)
			t.Logf("SQL emitido: %s", consulta)
			plano := explicar(t, consulta)
			if !strings.Contains(plano, c.indice) {
				t.Errorf("a consulta não usou %s.\nSQL emitido: %s\nPlano:\n%s", c.indice, consulta, plano)
			}
			if strings.Contains(plano, "Seq Scan on "+c.tabela) {
				t.Errorf("varredura sequencial em %s com volume alto.\nSQL emitido: %s\nPlano:\n%s", c.tabela, consulta, plano)
			}
		})
	}
}

type capturaSQL struct {
	mu   sync.Mutex
	sqls []string
}

func (c *capturaSQL) LogMode(logger.LogLevel) logger.Interface      { return c }
func (c *capturaSQL) Info(context.Context, string, ...interface{})  {}
func (c *capturaSQL) Warn(context.Context, string, ...interface{})  {}
func (c *capturaSQL) Error(context.Context, string, ...interface{}) {}
func (c *capturaSQL) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	c.mu.Lock()
	c.sqls = append(c.sqls, sql)
	c.mu.Unlock()
}

func (c *capturaSQL) limpar() {
	c.mu.Lock()
	c.sqls = nil
	c.mu.Unlock()
}

func (c *capturaSQL) paginaDaListagem(t *testing.T) string {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range c.sqls {
		if strings.Contains(s, " LIMIT ") && !strings.Contains(strings.ToUpper(s), "COUNT(") {
			return s
		}
	}
	t.Fatalf("nenhuma consulta de página capturada entre: %v", c.sqls)
	return ""
}

func TestContagemDoTotalCabeNoOrcamento(t *testing.T) {
	carregarVolume(t, 10)
	ctx := context.Background()
	if _, err := pool.ExecContext(ctx, `ANALYZE`); err != nil {
		t.Fatal(err)
	}

	const limite = 200 * time.Millisecond
	p95 := medirP95(t, 30, func() {
		var total int
		if err := pool.QueryRowContext(ctx,
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
	rows, err := pool.QueryContext(context.Background(), "EXPLAIN (ANALYZE, BUFFERS) "+sql, args...)
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
