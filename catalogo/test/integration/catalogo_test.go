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

func TestEscritaDeFilmeRoundTrip(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoFilmeRepository(pool)
	ctx := context.Background()
	const id = "b0000000-0000-4000-8000-00000000abcd"

	sinopse := "Um filme criado pelo teste de integração."
	novo, err := catalogo.NovoFilme(id, catalogo.DadosFilme{
		Titulo: "Filme de Integração", Sinopse: &sinopse, DuracaoMinutos: 95,
		ClassificacaoEtaria: "Livre", Genero: "Documentário",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Criar(ctx, novo); err != nil {
		t.Fatalf("criando filme: %v", err)
	}

	lido, err := repo.BuscarPorID(ctx, id)
	if err != nil {
		t.Fatalf("buscando filme recém-criado: %v", err)
	}
	if lido.Titulo != novo.Titulo || lido.Sinopse == nil || *lido.Sinopse != sinopse {
		t.Fatalf("o filme lido diverge do gravado: %+v", lido)
	}
	if lido.Status != catalogo.StatusEmCartaz {
		t.Fatalf("sem status explícito, o filme deveria nascer EM_CARTAZ, veio %q", lido.Status)
	}

	// A substituição é total: a sinopse omitida some da linha.
	atualizado, err := catalogo.NovoFilme(id, catalogo.DadosFilme{
		Titulo: "Filme de Integração II", DuracaoMinutos: 99,
		ClassificacaoEtaria: "Livre", Genero: "Documentário", Status: "BREVE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Atualizar(ctx, atualizado); err != nil {
		t.Fatalf("atualizando filme: %v", err)
	}
	lido, err = repo.BuscarPorID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if lido.Titulo != "Filme de Integração II" || lido.Status != catalogo.StatusBreve || lido.Sinopse != nil {
		t.Fatalf("atualização não substituiu o filme inteiro: %+v", lido)
	}

	if err := repo.MarcarForaDeCartaz(ctx, id); err != nil {
		t.Fatalf("removendo filme: %v", err)
	}
	lido, err = repo.BuscarPorID(ctx, id)
	if err != nil {
		t.Fatalf("a remoção é lógica: o filme deveria continuar no banco: %v", err)
	}
	if lido.Status != catalogo.StatusForaDeCartaz {
		t.Fatalf("status após remoção = %q, esperava FORA_DE_CARTAZ", lido.Status)
	}
}

func TestEscritaDeFilmeInexistenteDevolveNaoEncontrado(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoFilmeRepository(pool)
	ctx := context.Background()
	const ausente = "b0000000-0000-4000-8000-0000000000ff"

	if _, err := repo.BuscarPorID(ctx, ausente); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("BuscarPorID: esperava ErrNaoEncontrado, obteve %v", err)
	}

	filme, err := catalogo.NovoFilme(ausente, catalogo.DadosFilme{
		Titulo: "Fantasma", DuracaoMinutos: 10, ClassificacaoEtaria: "Livre", Genero: "Terror",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Atualizar(ctx, filme); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("Atualizar: esperava ErrNaoEncontrado, obteve %v", err)
	}
	if err := repo.MarcarForaDeCartaz(ctx, ausente); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("MarcarForaDeCartaz: esperava ErrNaoEncontrado, obteve %v", err)
	}
}
