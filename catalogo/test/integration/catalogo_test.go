//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

func pagina(t *testing.T, numero, tamanho int) shared.PageRequest {
	t.Helper()
	p, err := shared.NovoPageRequest(numero, tamanho, 20, 100)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestListarFilmesRecortePublico(t *testing.T) {
	carregarFixtures(t)
	uc := usecase.ListarFilmes{Repo: pgadapter.NovoFilmeRepository(pool)}

	p, err := uc.Executar(context.Background(), usecase.FiltroFilmes{}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 3 {
		t.Fatalf("esperava 3 filmes públicos (2 em cartaz + 1 em breve), obteve %d", p.Total)
	}
	for _, f := range p.Itens {
		if f.Status == catalogo.StatusForaDeCartaz {
			t.Errorf("filme fora de cartaz apareceu na vitrine: %s", f.Titulo)
		}
	}
}

func TestListarFilmesFiltradoPorStatus(t *testing.T) {
	carregarFixtures(t)
	uc := usecase.ListarFilmes{Repo: pgadapter.NovoFilmeRepository(pool)}

	fora := catalogo.StatusForaDeCartaz
	p, err := uc.Executar(context.Background(), usecase.FiltroFilmes{Status: &fora}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 1 || p.Itens[0].Titulo != "Filme Retirado" {
		t.Fatalf("filtro explícito não respeitado: %+v", p)
	}
}

func TestListarFilmesOrdenaPorTituloComDesempate(t *testing.T) {
	carregarFixtures(t)
	uc := usecase.ListarFilmes{Repo: pgadapter.NovoFilmeRepository(pool)}

	p, err := uc.Executar(context.Background(), usecase.FiltroFilmes{}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(p.Itens); i++ {
		if p.Itens[i-1].Titulo > p.Itens[i].Titulo {
			t.Fatalf("ordem quebrada: %q veio antes de %q", p.Itens[i-1].Titulo, p.Itens[i].Titulo)
		}
	}
}

func TestListarFilmesPreservaCamposOpcionaisNulos(t *testing.T) {
	carregarFixtures(t)
	uc := usecase.ListarFilmes{Repo: pgadapter.NovoFilmeRepository(pool)}

	p, _ := uc.Executar(context.Background(), usecase.FiltroFilmes{}, pagina(t, 1, 20))
	for _, f := range p.Itens {
		if f.Titulo == "Aurora Boreal" {
			if f.Sinopse != nil || f.ImagemURL != nil {
				t.Fatal("campos ausentes no banco deveriam chegar nil ao domínio")
			}
			return
		}
	}
	t.Fatal("filme sem material de apoio não foi listado")
}

func TestListarFilmesPaginacaoNaoRepeteRegistros(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoFilmeRepository(pool)

	p1, _ := repo.Listar(context.Background(), usecase.FiltroFilmes{}, catalogo.StatusPublicos, pagina(t, 1, 2))
	p2, _ := repo.Listar(context.Background(), usecase.FiltroFilmes{}, catalogo.StatusPublicos, pagina(t, 2, 2))

	if p1.Total != 3 || p2.Total != 3 {
		t.Fatalf("total deve refletir o filtro em toda página: %d/%d", p1.Total, p2.Total)
	}
	if !p1.TemProxima || p2.TemProxima {
		t.Fatalf("tem_proxima errado: p1=%v p2=%v", p1.TemProxima, p2.TemProxima)
	}
	vistos := map[string]bool{}
	for _, f := range append(p1.Itens, p2.Itens...) {
		if vistos[f.ID] {
			t.Fatalf("filme %s repetido entre páginas", f.ID)
		}
		vistos[f.ID] = true
	}
}

func TestListarFilmesAlemDoFim(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoFilmeRepository(pool)

	p, err := repo.Listar(context.Background(), usecase.FiltroFilmes{}, catalogo.StatusPublicos, pagina(t, 999, 20))
	if err != nil {
		t.Fatalf("página além do fim não deveria falhar: %v", err)
	}
	if len(p.Itens) != 0 {
		t.Fatalf("esperava página vazia, obteve %d itens", len(p.Itens))
	}
	if p.TemProxima {
		t.Error("página além do fim não pode indicar próxima")
	}
}

func TestListarCinemasESalas(t *testing.T) {
	carregarFixtures(t)
	cinemas := pgadapter.NovoCinemaRepository(pool)
	salas := pgadapter.NovoSalaRepository(pool)
	uc := usecase.ListarSalas{Cinemas: cinemas, Salas: salas}

	pc, err := cinemas.Listar(context.Background(), pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if pc.Total != 2 {
		t.Fatalf("esperava 2 cinemas, obteve %d", pc.Total)
	}

	ps, err := uc.Executar(context.Background(), "b1b2c3d4-0000-4000-8000-000000000001", pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if ps.Total != 2 {
		t.Fatalf("esperava 2 salas no cinema, obteve %d", ps.Total)
	}
	if ps.Itens[0].Numero > ps.Itens[1].Numero {
		t.Error("salas deveriam vir ordenadas por número")
	}
}

func TestListarSalasDeCinemaInexistente(t *testing.T) {
	carregarFixtures(t)
	uc := usecase.ListarSalas{
		Cinemas: pgadapter.NovoCinemaRepository(pool),
		Salas:   pgadapter.NovoSalaRepository(pool),
	}
	_, err := uc.Executar(context.Background(), "00000000-0000-0000-0000-000000000000", pagina(t, 1, 20))
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
}
