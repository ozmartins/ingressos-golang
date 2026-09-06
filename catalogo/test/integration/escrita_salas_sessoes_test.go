//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

const (
	cinemaDasFixtures  = "b1b2c3d4-0000-4000-8000-000000000001"
	filmeDasFixtures   = "c394c8b3-76a1-4328-b803-02f5923b7a15"
	salaVipDasFixtures = "d1b2c3d4-0000-4000-8000-000000000003"
)

func dadosSala(numero int) catalogo.DadosSala {
	return catalogo.DadosSala{
		CinemaID:        cinemaDasFixtures,
		Numero:          numero,
		TipoTela:        "VIP",
		CapacidadeTotal: 80,
	}
}

func TestEscritaDeSalaRoundTrip(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSalaRepository(pool)
	ctx := context.Background()
	const id = "d0000000-0000-4000-8000-00000000abcd"

	sala, err := catalogo.NovaSala(id, dadosSala(9))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Criar(ctx, sala); err != nil {
		t.Fatalf("Criar: %v", err)
	}

	lida, err := repo.BuscarPorID(ctx, id)
	if err != nil {
		t.Fatalf("BuscarPorID: %v", err)
	}
	if lida != sala {
		t.Fatalf("a sala lida difere da gravada:\n gravada: %+v\n lida:    %+v", sala, lida)
	}

	dados := dadosSala(9)
	dados.TipoTela, dados.CapacidadeTotal = "2D", 200
	alterada, err := catalogo.NovaSala(id, dados)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Atualizar(ctx, alterada); err != nil {
		t.Fatalf("Atualizar: %v", err)
	}
	if relida, err := repo.BuscarPorID(ctx, id); err != nil || relida != alterada {
		t.Fatalf("a atualização não persistiu: %+v (%v)", relida, err)
	}

	if err := repo.Desativar(ctx, id); err != nil {
		t.Fatalf("Desativar: %v", err)
	}

	// A remoção é lógica: a linha permanece e as sessões seguem apontando para ela.
	inativa, err := repo.BuscarPorID(ctx, id)
	if err != nil {
		t.Fatalf("a sala desativada deveria seguir legível: %v", err)
	}
	if inativa.Ativo {
		t.Fatal("a sala deveria estar inativa depois do Desativar")
	}
}

func TestListarSalasFiltraPorSituacao(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSalaRepository(pool)
	uc := usecase.ListarSalas{Cinemas: pgadapter.NovoCinemaRepository(pool), Salas: repo}
	ctx := context.Background()

	if err := repo.Desativar(ctx, "d1b2c3d4-0000-4000-8000-000000000001"); err != nil {
		t.Fatalf("Desativar: %v", err)
	}

	ativa := true
	ativas, err := uc.Executar(ctx, usecase.FiltroSalas{CinemaID: cinemaDasFixtures, Ativo: &ativa}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if ativas.Total != 1 {
		t.Fatalf("esperava 1 sala ativa no cinema, obteve %d", ativas.Total)
	}

	todas, err := uc.Executar(ctx, usecase.FiltroSalas{CinemaID: cinemaDasFixtures}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if todas.Total != 2 {
		t.Fatalf("sem filtro o repositório deveria ver as 2 linhas, obteve %d", todas.Total)
	}
}

func TestNumeroDeSalaEhUnicoEntreAsAtivas(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSalaRepository(pool)
	ctx := context.Background()

	emUso, err := repo.NumeroEmUso(ctx, cinemaDasFixtures, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if !emUso {
		t.Fatal("a sala 3 das fixtures está ativa: o número deveria constar em uso")
	}

	// A própria sala não disputa o próprio número.
	emUso, err = repo.NumeroEmUso(ctx, cinemaDasFixtures, 3, "d1b2c3d4-0000-4000-8000-000000000002")
	if err != nil {
		t.Fatal(err)
	}
	if emUso {
		t.Fatal("a sala deveria poder manter o próprio número")
	}

	// O índice é parcial: desativar libera o número.
	if err := repo.Desativar(ctx, "d1b2c3d4-0000-4000-8000-000000000002"); err != nil {
		t.Fatal(err)
	}
	if emUso, err = repo.NumeroEmUso(ctx, cinemaDasFixtures, 3, ""); err != nil || emUso {
		t.Fatalf("desativar a sala deveria liberar o número: emUso=%v err=%v", emUso, err)
	}

	substituta, err := catalogo.NovaSala("d0000000-0000-4000-8000-00000000beef", dadosSala(3))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Criar(ctx, substituta); err != nil {
		t.Fatalf("a sala substituta deveria ser aceita no número liberado: %v", err)
	}
}

func TestIndiceUnicoRecusaDuasSalasAtivasComOMesmoNumero(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSalaRepository(pool)
	ctx := context.Background()

	repetida, err := catalogo.NovaSala("d0000000-0000-4000-8000-00000000feed", dadosSala(3))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Criar(ctx, repetida); err == nil {
		t.Fatal("o banco deveria recusar duas salas ativas de número 3 no mesmo cinema")
	}
}

func TestEscritaDeSalaInexistenteDevolveNaoEncontrado(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSalaRepository(pool)
	ctx := context.Background()
	const ausente = "d1b2c3d4-0000-4000-8000-0000000000ff"

	if _, err := repo.BuscarPorID(ctx, ausente); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("BuscarPorID: esperava ErrNaoEncontrado, obteve %v", err)
	}

	sala, err := catalogo.NovaSala(ausente, dadosSala(11))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Atualizar(ctx, sala); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("Atualizar: esperava ErrNaoEncontrado, obteve %v", err)
	}
	if err := repo.Desativar(ctx, ausente); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("Desativar: esperava ErrNaoEncontrado, obteve %v", err)
	}
}

func dadosSessao(inicio time.Time) catalogo.DadosSessao {
	return catalogo.DadosSessao{
		FilmeID:        filmeDasFixtures,
		SalaID:         salaVipDasFixtures,
		DataHoraInicio: inicio,
		Idioma:         "LEGENDADO",
		PrecoBase:      "42.50",
	}
}

func TestEscritaDeSessaoRoundTrip(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	ctx := context.Background()
	const id = "e0000000-0000-4000-8000-00000000abcd"

	sessao, err := catalogo.NovaSessao(id, dadosSessao(time.Date(2026, 10, 1, 19, 30, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Criar(ctx, sessao); err != nil {
		t.Fatalf("Criar: %v", err)
	}

	lida, err := repo.BuscarPorID(ctx, id)
	if err != nil {
		t.Fatalf("BuscarPorID: %v", err)
	}
	if lida != sessao {
		t.Fatalf("a sessão lida difere da gravada:\n gravada: %+v\n lida:    %+v", sessao, lida)
	}
	if lida.PrecoBase.String() != "42.50" {
		t.Fatalf("o preço deveria atravessar o numeric sem perda, veio %s", lida.PrecoBase)
	}

	dados := dadosSessao(time.Date(2026, 10, 2, 21, 0, 0, 0, time.UTC))
	dados.Idioma, dados.PrecoBase = "DUBLADO", "55.00"
	alterada, err := catalogo.NovaSessao(id, dados)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Atualizar(ctx, alterada); err != nil {
		t.Fatalf("Atualizar: %v", err)
	}
	if relida, err := repo.BuscarPorID(ctx, id); err != nil || relida != alterada {
		t.Fatalf("a atualização não persistiu: %+v (%v)", relida, err)
	}

	if err := repo.Cancelar(ctx, id); err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	cancelada, err := repo.BuscarPorID(ctx, id)
	if err != nil {
		t.Fatalf("a sessão cancelada deveria seguir legível: %v", err)
	}
	if cancelada.Status != catalogo.SessaoCancelada {
		t.Fatalf("status = %q, esperava CANCELADA", cancelada.Status)
	}
}

func TestSessaoCanceladaSaiDaGrade(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	ctx := context.Background()
	const naGrade = "e1b2c3d4-0000-4000-8000-000000000001"

	antes, err := repo.Consultar(ctx, usecase.FiltroSessoes{}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Cancelar(ctx, naGrade); err != nil {
		t.Fatal(err)
	}
	depois, err := repo.Consultar(ctx, usecase.FiltroSessoes{}, pagina(t, 1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if depois.Total != antes.Total-1 {
		t.Fatalf("a sessão cancelada deveria sair da grade: %d → %d", antes.Total, depois.Total)
	}
}

// A sessão de 18h na sala VIP das fixtures projeta "Zebra Selvagem", de 90
// minutos: ocupa a sala das 18h às 19h30.
func TestSalaOcupadaUsaADuracaoDoFilme(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	ctx := context.Background()
	dia := func(hora, minuto int) time.Time {
		return time.Date(2026, 9, 2, hora, minuto, 0, 0, time.UTC)
	}

	casos := []struct {
		nome          string
		inicio, fim   time.Time
		esperaOcupada bool
	}{
		{"começa no meio da sessão existente", dia(18, 30), dia(20, 0), true},
		{"termina no meio da sessão existente", dia(17, 0), dia(18, 30), true},
		{"engloba a sessão existente", dia(17, 0), dia(21, 0), true},
		{"encosta no fim, sem sobrepor", dia(19, 30), dia(21, 0), false},
		{"encosta no início, sem sobrepor", dia(16, 30), dia(18, 0), false},
		{"em outro dia", dia(18, 0).AddDate(0, 0, 5), dia(20, 0).AddDate(0, 0, 5), false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			ocupada, err := repo.SalaOcupada(ctx, salaVipDasFixtures, c.inicio, c.fim, "")
			if err != nil {
				t.Fatal(err)
			}
			if ocupada != c.esperaOcupada {
				t.Fatalf("ocupada = %v, esperava %v", ocupada, c.esperaOcupada)
			}
		})
	}
}

func TestSalaOcupadaIgnoraSessoesQueNaoOcupam(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	ctx := context.Background()

	// A sessão CANCELADA das fixtures começa às 15h de 03/09 na sala IMAX.
	const salaImax = "d1b2c3d4-0000-4000-8000-000000000002"
	inicio := time.Date(2026, 9, 3, 15, 0, 0, 0, time.UTC)

	ocupada, err := repo.SalaOcupada(ctx, salaImax, inicio, inicio.Add(2*time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	if ocupada {
		t.Fatal("uma sessão cancelada não ocupa a sala")
	}
}

func TestSalaOcupadaExcluiAPropriaSessao(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	ctx := context.Background()
	const propria = "e1b2c3d4-0000-4000-8000-000000000003"
	inicio := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)

	ocupada, err := repo.SalaOcupada(ctx, salaVipDasFixtures, inicio, inicio.Add(90*time.Minute), propria)
	if err != nil {
		t.Fatal(err)
	}
	if ocupada {
		t.Fatal("na atualização, a própria sessão não deveria contar como ocupação")
	}
}

func TestEscritaDeSessaoInexistenteDevolveNaoEncontrado(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	ctx := context.Background()
	const ausente = "e1b2c3d4-0000-4000-8000-0000000000ff"

	if _, err := repo.BuscarPorID(ctx, ausente); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("BuscarPorID: esperava ErrNaoEncontrado, obteve %v", err)
	}

	sessao, err := catalogo.NovaSessao(ausente, dadosSessao(time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Atualizar(ctx, sessao); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("Atualizar: esperava ErrNaoEncontrado, obteve %v", err)
	}
	if err := repo.Cancelar(ctx, ausente); !errors.Is(err, shared.ErrNaoEncontrado) {
		t.Errorf("Cancelar: esperava ErrNaoEncontrado, obteve %v", err)
	}
}
