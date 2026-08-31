//go:build integration

// Package integration exercita o serviço contra PostgreSQL e RabbitMQ reais.
// É a única camada que precisa de Docker, e é onde as garantias de
// concorrência, reentrega e quarentena podem ser provadas de verdade
// (research.md D12).
package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcrabbit "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/wait"

	adaptamqp "github.com/oseias/ingressos-golang/notificacao/internal/adapter/amqp"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/codigo"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/notificador/simulado"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/sistema"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

const (
	exchange    = "cinema.eventos"
	exchangeDLX = "cinema.eventos.dlx"
	fila        = "notificacao.pagamento-sucesso"
	filaDLQ     = "notificacao.pagamento-sucesso.dlq"
	segredoQR   = "segredo-de-teste-da-integracao"
	// LimiteEntregas do ambiente de teste: a FR-022 fala em TENTATIVAS.
	limiteEntregas = 3
)

type ambiente struct {
	Pool      *pgxpool.Pool
	Ingressos postgres.Ingressos
	Avisos    postgres.Avisos
	Conexao   *amqp091.Connection
	Canal     *amqp091.Channel
	Assinador *codigo.Assinador
	Log       *slog.Logger
}

// subirAmbiente cria Postgres e RabbitMQ reais, aplica a migração e declara a
// topologia — exatamente a mesma declaração que o serviço usa na largada.
func subirAmbiente(t *testing.T) *ambiente {
	t.Helper()
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("notificacao"),
		tcpostgres.WithUsername("notificacao"),
		tcpostgres.WithPassword("notificacao"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(2*time.Minute)),
	)
	if err != nil {
		t.Fatalf("subir postgres: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	aplicarMigracao(t, pool)

	rmq, err := tcrabbit.Run(ctx, "rabbitmq:3.13-management-alpine")
	if err != nil {
		t.Fatalf("subir rabbitmq: %v", err)
	}
	t.Cleanup(func() { _ = rmq.Terminate(context.Background()) })

	url, err := rmq.AmqpURL(ctx)
	if err != nil {
		t.Fatal(err)
	}
	conexao, err := amqp091.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conexao.Close() })

	canal, err := conexao.Channel()
	if err != nil {
		t.Fatal(err)
	}

	topo := adaptamqp.Topologia{
		Exchange: exchange, ExchangeDLX: exchangeDLX,
		Fila: fila, FilaDLQ: filaDLQ,
		Binding: "pagamento.sucesso", LimiteEntregas: limiteEntregas,
	}
	if err := topo.Declarar(canal); err != nil {
		t.Fatalf("declarar topologia: %v", err)
	}

	assinador, err := codigo.NovoAssinador(segredoQR)
	if err != nil {
		t.Fatal(err)
	}

	return &ambiente{
		Pool: pool, Ingressos: postgres.Ingressos{Pool: pool}, Avisos: postgres.Avisos{Pool: pool},
		Conexao: conexao, Canal: canal, Assinador: assinador,
		Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
}

func aplicarMigracao(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000001_criar_ingressos.up.sql"))
	if err != nil {
		t.Fatalf("ler migração: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(b)); err != nil {
		t.Fatalf("aplicar migração: %v", err)
	}
}

// caso monta o caso de uso de emissão sobre o ambiente real.
func (a *ambiente) caso(falharAviso bool) usecase.EmitirIngresso {
	return usecase.EmitirIngresso{
		Ingressos:   a.Ingressos,
		Avisos:      a.Avisos,
		Notificador: simulado.Notificador{Falhar: falharAviso},
		Assinador:   a.Assinador,
		Relogio:     sistema.Relogio{},
		IDs:         sistema.GeradorID{},
		Log:         a.Log,
	}
}

func (a *ambiente) consumidor(t *testing.T, falharAviso bool) *adaptamqp.Consumidor {
	t.Helper()
	canal, err := a.Conexao.Channel()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = canal.Close() })
	return &adaptamqp.Consumidor{
		Canal: canal, Fila: fila, Prefetch: 10,
		Caso: a.caso(falharAviso), Log: a.Log,
		EsperaAntesDeDevolver: 50 * time.Millisecond,
	}
}

// publicar envia um pagamento.sucesso na mesma forma que o Servico-Pagamento
// publica hoje (research.md D1).
func (a *ambiente) publicar(t *testing.T, reservaID, usuarioID string) {
	t.Helper()
	agora := time.Now().UTC().Format(time.RFC3339)
	corpo, err := json.Marshal(map[string]any{
		"evento": "PAGAMENTO_SUCESSO", "versao": 1, "ocorrido_em": agora,
		"transacao_id": uuid.NewString(), "reserva_id": reservaID,
		"usuario_id": usuarioID, "valor_total": 84.00, "pago_em": agora,
	})
	if err != nil {
		t.Fatal(err)
	}
	a.publicarCru(t, corpo)
}

func (a *ambiente) publicarCru(t *testing.T, corpo []byte) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := a.Canal.PublishWithContext(ctx, exchange, "pagamento.sucesso", false, false,
		amqp091.Publishing{ContentType: "application/json", Body: corpo}); err != nil {
		t.Fatalf("publicar: %v", err)
	}
}

// contarFila devolve quantas mensagens estão na fila, sem consumi-las.
func (a *ambiente) contarFila(t *testing.T, nome string) int {
	t.Helper()
	canal, err := a.Conexao.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer canal.Close()
	q, err := canal.QueueDeclarePassive(nome, true, false, false, false, nil)
	if err != nil {
		t.Fatalf("inspecionar fila %s: %v", nome, err)
	}
	return q.Messages
}

// esperar repete a condição até o prazo, e falha com a mensagem se não vier.
func esperar(t *testing.T, prazo time.Duration, mensagem string, cond func() bool) {
	t.Helper()
	limite := time.Now().Add(prazo)
	for time.Now().Before(limite) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("prazo de %s esgotado esperando: %s", prazo, mensagem)
}

func (a *ambiente) contarIngressos(t *testing.T) int {
	t.Helper()
	var n int
	if err := a.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ingressos_emitidos`).Scan(&n); err != nil {
		t.Fatalf("contar ingressos: %v", err)
	}
	return n
}
