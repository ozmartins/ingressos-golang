//go:build integration

package integration

import (
	"context"
	"testing"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

// FR-010 e SC-002: toda consulta reflete o estado atual.
//
// É um requisito negativo — a ausência de cache não se manifesta sozinha. Sem
// este teste, alguém poderia introduzir um cache de leitura amanhã e nenhuma
// suíte acusaria.
func TestConsultasRefletemAlteracaoImediatamente(t *testing.T) {
	carregarFixtures(t)
	ctx := context.Background()

	filmes := usecase.ListarFilmes{Repo: pgadapter.NovoFilmeRepository(pool)}

	// Primeira leitura: aquece qualquer cache que porventura exista.
	antes, err := filmes.Executar(ctx, usecase.FiltroFilmes{}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}

	// O processo administrativo altera o título de um filme.
	if _, err := pool.Exec(ctx,
		`UPDATE filmes SET titulo = 'Duna: Parte 2 (Reestreia)' WHERE id = $1`, filmeDuna); err != nil {
		t.Fatal(err)
	}

	depois, err := filmes.Executar(ctx, usecase.FiltroFilmes{}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if !contemTitulo(depois, "Duna: Parte 2 (Reestreia)") {
		t.Fatal("a consulta seguinte não refletiu a alteração: há cache servindo dado desatualizado")
	}
	if contemTitulo(antes, "Duna: Parte 2 (Reestreia)") {
		t.Fatal("teste inválido: o título já estava alterado antes do UPDATE")
	}

	// Mesma verificação na grade, que é a consulta com junções.
	antesGrade := consultar(t, usecase.FiltroSessoes{}, pagina(t, 1, 20))
	if _, err := pool.Exec(ctx,
		`UPDATE sessoes SET status = 'CANCELADA' WHERE id = 'e1b2c3d4-0000-4000-8000-000000000001'`); err != nil {
		t.Fatal(err)
	}
	depoisGrade := consultar(t, usecase.FiltroSessoes{}, pagina(t, 1, 20))

	if depoisGrade.Total != antesGrade.Total-1 {
		t.Fatalf("cancelamento não refletido na consulta seguinte: antes=%d depois=%d",
			antesGrade.Total, depoisGrade.Total)
	}
}

func contemTitulo(p shared.Page[catalogo.Filme], titulo string) bool {
	for _, f := range p.Itens {
		if f.Titulo == titulo {
			return true
		}
	}
	return false
}
