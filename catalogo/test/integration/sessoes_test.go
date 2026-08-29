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

const (
	filmeDuna      = "c394c8b3-76a1-4328-b803-02f5923b7a15"
	cinemaCentro   = "b1b2c3d4-0000-4000-8000-000000000001"
	cinemaBeiramar = "b1b2c3d4-0000-4000-8000-000000000002"
)

func consultar(t *testing.T, filtro usecase.FiltroSessoes, req shared.PageRequest) shared.Page[catalogo.SessaoDetalhada] {
	t.Helper()
	uc := usecase.ConsultarSessoes{Repo: pgadapter.NovoSessaoRepository(pool)}
	p, err := uc.Executar(context.Background(), filtro, req)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// US2, cenário 6 — canceladas e finalizadas não aparecem.
func TestGradeExcluiCanceladasEFinalizadas(t *testing.T) {
	carregarFixtures(t)
	p := consultar(t, usecase.FiltroSessoes{}, pagina(t, 1, 20))

	// 7 sessões nas fixtures, menos 1 cancelada e 1 finalizada.
	if p.Total != 5 {
		t.Fatalf("esperava 5 sessões visíveis, obteve %d", p.Total)
	}
}

// US2, cenário 1 — dados consolidados vindos das quatro tabelas.
func TestGradeConsolidaFilmeCinemaESala(t *testing.T) {
	carregarFixtures(t)
	p := consultar(t, usecase.FiltroSessoes{FilmeID: filmeDuna}, pagina(t, 1, 20))

	if p.Total != 2 {
		t.Fatalf("esperava 2 sessões de Duna visíveis, obteve %d", p.Total)
	}
	for _, s := range p.Itens {
		if s.FilmeTitulo != "Duna: Parte 2" || s.CinemaNome == "" || s.SalaNumero == 0 || s.TipoTela == "" {
			t.Fatalf("junção incompleta: %+v", s)
		}
	}
}

// US2, cenário 2.
func TestGradeFiltraPorData(t *testing.T) {
	carregarFixtures(t)
	p := consultar(t, usecase.FiltroSessoes{Data: &usecase.DataDoDia{Ano: 2026, Mes: 9, Dia: 2}}, pagina(t, 1, 20))

	if p.Total != 2 {
		t.Fatalf("esperava 2 sessões em 2026-09-02, obteve %d", p.Total)
	}
	for _, s := range p.Itens {
		if s.DataHoraInicio.Day() != 2 || int(s.DataHoraInicio.Month()) != 9 {
			t.Errorf("sessão fora da data filtrada: %s", s.DataHoraInicio)
		}
	}
}

// US2, cenário 3 — o filtro por cinema alcança as sessões através das salas.
func TestGradeCombinaFiltros(t *testing.T) {
	carregarFixtures(t)

	p := consultar(t, usecase.FiltroSessoes{FilmeID: filmeDuna, CinemaID: cinemaCentro,
		Data: &usecase.DataDoDia{Ano: 2026, Mes: 9, Dia: 1}}, pagina(t, 1, 20))
	if p.Total != 2 {
		t.Fatalf("esperava 2 sessões, obteve %d", p.Total)
	}

	// Combinação sem resultado devolve página vazia, não erro (FR-017).
	vazia := consultar(t, usecase.FiltroSessoes{FilmeID: filmeDuna, CinemaID: cinemaBeiramar}, pagina(t, 1, 20))
	if vazia.Total != 0 || len(vazia.Itens) != 0 {
		t.Fatalf("esperava página vazia, obteve %+v", vazia)
	}
}

// US2, cenário 4 — estabilidade entre páginas consecutivas.
func TestGradePaginacaoEstavel(t *testing.T) {
	carregarFixtures(t)

	p1 := consultar(t, usecase.FiltroSessoes{}, pagina(t, 1, 2))
	p2 := consultar(t, usecase.FiltroSessoes{}, pagina(t, 2, 2))
	p3 := consultar(t, usecase.FiltroSessoes{}, pagina(t, 3, 2))

	vistos := map[string]bool{}
	for _, p := range []shared.Page[catalogo.SessaoDetalhada]{p1, p2, p3} {
		if p.Total != 5 {
			t.Fatalf("total inconsistente entre páginas: %d", p.Total)
		}
		for _, s := range p.Itens {
			if vistos[s.ID] {
				t.Fatalf("sessão %s repetida entre páginas", s.ID)
			}
			vistos[s.ID] = true
		}
	}
	if len(vistos) != 5 {
		t.Fatalf("as três páginas cobriram %d de 5 sessões", len(vistos))
	}
	// Ordenação crescente por início.
	for i := 1; i < len(p1.Itens); i++ {
		if p1.Itens[i-1].DataHoraInicio.After(p1.Itens[i].DataHoraInicio) {
			t.Error("ordem por data_hora_inicio quebrada")
		}
	}
}

// O tipo decimal sobrevive à ida e volta ao banco.
func TestPrecoBasePreservaExatidao(t *testing.T) {
	carregarFixtures(t)
	p := consultar(t, usecase.FiltroSessoes{}, pagina(t, 1, 20))

	esperados := map[string]bool{"42.00": false, "32.50": false, "55.00": false}
	for _, s := range p.Itens {
		if _, conhecido := esperados[s.PrecoBase.String()]; !conhecido {
			t.Errorf("preço inesperado: %s", s.PrecoBase)
		}
		esperados[s.PrecoBase.String()] = true
	}
	for preco, visto := range esperados {
		if !visto {
			t.Errorf("preço %s não apareceu; conversão pode ter perdido exatidão", preco)
		}
	}
}

// Caso de borda: sessão apontando para filme inexistente é omitida, sem derrubar
// a consulta inteira.
func TestSessaoOrfaEhOmitidaSemDerrubarAGrade(t *testing.T) {
	carregarFixtures(t)
	ctx := context.Background()

	// Remove a chave estrangeira para poder criar a inconsistência que o mundo
	// real produz quando o processo administrativo apaga um filme.
	if _, err := pool.Exec(ctx, `ALTER TABLE sessoes DROP CONSTRAINT sessoes_filme_id_fkey`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM sessoes WHERE id = 'orfa'`)
		_, _ = pool.Exec(ctx, `ALTER TABLE sessoes ADD CONSTRAINT sessoes_filme_id_fkey FOREIGN KEY (filme_id) REFERENCES filmes(id)`)
	}()

	_, err := pool.Exec(ctx, `INSERT INTO sessoes (id, filme_id, sala_id, data_hora_inicio, idioma, preco_base, status)
	  VALUES ('orfa','filme-que-nao-existe','d1b2c3d4-0000-4000-8000-000000000001','2026-09-05T10:00:00Z','DUBLADO',30.00,'AGENDADA')`)
	if err != nil {
		t.Fatal(err)
	}

	p := consultar(t, usecase.FiltroSessoes{}, pagina(t, 1, 20))
	if p.Total != 5 {
		t.Fatalf("a sessão órfã deveria ser omitida, mantendo 5 visíveis; obteve %d", p.Total)
	}
	for _, s := range p.Itens {
		if s.ID == "orfa" {
			t.Fatal("sessão órfã apareceu na grade")
		}
	}
}

// --- busca por id, usada pela reserva ---------------------------------------

func TestBuscarSessaoPorID(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)

	s, err := repo.BuscarPorID(context.Background(), "e1b2c3d4-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != catalogo.SessaoAgendada || s.PrecoBase.String() != "42.00" {
		t.Fatalf("sessão lida incorretamente: %+v", s)
	}

	_, err = repo.BuscarPorID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
}
