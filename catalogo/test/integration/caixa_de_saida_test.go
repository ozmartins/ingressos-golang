//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

// O fato e a sessão vão na mesma transação. Se a inserção da sessão falha, o
// anúncio dela não pode sobrar na caixa: é a indivisibilidade que o princípio VI
// exige.
func TestSessaoRecusadaNaoDeixaFatoNaCaixa(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	ctx := context.Background()
	const id = "e0000000-0000-4000-8000-0000000000f1"

	sessao, err := catalogo.NovaSessao(id, dadosSessao(time.Date(2026, 10, 5, 19, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	// Uma sala que não existe: a chave estrangeira derruba a transação inteira.
	sessao.SalaID = "d0000000-0000-4000-8000-00000000ffff"

	if err := repo.Criar(ctx, sessao, fatoDeTeste(id)); err == nil {
		t.Fatal("esperava falha ao gravar sessão com sala inexistente")
	}

	var pendentes int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox_eventos WHERE message_id = $1`, id).Scan(&pendentes); err != nil {
		t.Fatal(err)
	}
	if pendentes != 0 {
		t.Fatalf("a transação desfeita deixou %d fato(s) na caixa", pendentes)
	}
}

func TestCaixaDeSaidaDrenaEMarcaOFatoPublicado(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	caixa := pgadapter.NovaCaixaDeSaida(pool)
	ctx := context.Background()
	const id = "e0000000-0000-4000-8000-0000000000f2"

	sessao, err := catalogo.NovaSessao(id, dadosSessao(time.Date(2026, 10, 6, 19, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Criar(ctx, sessao, fatoDeTeste(id)); err != nil {
		t.Fatalf("Criar: %v", err)
	}

	var recebidos []pgadapter.FatoNaCaixa
	n, err := caixa.Drenar(ctx, 10, func(f pgadapter.FatoNaCaixa) error {
		recebidos = append(recebidos, f)
		return nil
	})
	if err != nil {
		t.Fatalf("Drenar: %v", err)
	}
	if n != 1 || len(recebidos) != 1 {
		t.Fatalf("drenou %d fato(s), esperava 1", n)
	}

	f := recebidos[0]
	if f.MessageID != id || f.RoutingKey != usecase.RoutingKeySessaoCriada {
		t.Errorf("fato inesperado: %+v", f)
	}
	// O contexto de rastreamento atravessa o JSONB e volta inteiro.
	if f.TraceContext["traceparent"] == "" {
		t.Errorf("o contexto de rastreamento não sobreviveu à ida e volta: %+v", f.TraceContext)
	}
	var corpo map[string]any
	if err := json.Unmarshal(f.Payload, &corpo); err != nil {
		t.Errorf("o payload não voltou como JSON: %v", err)
	}

	// Drenado uma vez, não sai de novo.
	if n, err := caixa.Drenar(ctx, 10, func(pgadapter.FatoNaCaixa) error {
		t.Error("um fato já publicado não deveria ser reenviado")
		return nil
	}); err != nil || n != 0 {
		t.Fatalf("segunda drenagem devolveu %d (%v), esperava 0", n, err)
	}
}

// Publicação que falha não marca o fato: ele volta no próximo tique, com a
// tentativa contada. É daí que vem o "ao menos uma vez".
func TestCaixaDeSaidaMantemOFatoQuandoAPublicacaoFalha(t *testing.T) {
	carregarFixtures(t)
	repo := pgadapter.NovoSessaoRepository(pool)
	caixa := pgadapter.NovaCaixaDeSaida(pool)
	ctx := context.Background()
	const id = "e0000000-0000-4000-8000-0000000000f3"

	sessao, err := catalogo.NovaSessao(id, dadosSessao(time.Date(2026, 10, 7, 19, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Criar(ctx, sessao, fatoDeTeste(id)); err != nil {
		t.Fatalf("Criar: %v", err)
	}

	brokerFora := errors.New("broker fora do ar")
	if n, err := caixa.Drenar(ctx, 10, func(pgadapter.FatoNaCaixa) error { return brokerFora }); err != nil || n != 0 {
		t.Fatalf("drenagem com broker fora devolveu %d (%v), esperava 0", n, err)
	}

	var tentativas int
	if err := pool.QueryRow(ctx,
		`SELECT tentativas FROM outbox_eventos WHERE message_id = $1 AND publicado_em IS NULL`,
		id).Scan(&tentativas); err != nil {
		t.Fatalf("o fato deveria continuar pendente: %v", err)
	}
	if tentativas != 1 {
		t.Fatalf("tentativas = %d, esperava 1", tentativas)
	}

	// Com o broker de volta, o mesmo fato sai.
	if n, err := caixa.Drenar(ctx, 10, func(pgadapter.FatoNaCaixa) error { return nil }); err != nil || n != 1 {
		t.Fatalf("drenagem seguinte devolveu %d (%v), esperava 1", n, err)
	}
}
