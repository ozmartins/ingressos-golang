//go:build integration

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

	adaptamqp "github.com/oseias/ingressos-golang/pagamento/internal/adapter/amqp"
	"github.com/oseias/ingressos-golang/pagamento/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
	"go.opentelemetry.io/otel/propagation"
)

const (
	exchange    = "cinema.eventos"
	exchangeDLX = "cinema.eventos.dlx"
	filaReserva = "pagamento.reserva-criada"
	filaDLQ     = "pagamento.reserva-criada.dlq"
	filaEspiao  = "teste.espiao"
)

type ambiente struct {
	Pool    *pgxpool.Pool
	Repo    *postgres.Repositorio
	Conexao *amqp091.Connection
	Canal   *amqp091.Channel
	Publica *adaptamqp.Publicador
	Log     *slog.Logger
}

type relogioReal struct{}

func (relogioReal) Agora() time.Time { return time.Now().UTC() }

type idsReais struct{}

func (idsReais) Novo() string { return uuid.NewString() }

func subirAmbiente(t *testing.T) *ambiente {
	t.Helper()
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("pagamento"),
		tcpostgres.WithUsername("pagamento"),
		tcpostgres.WithPassword("pagamento"),
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
	pool, err := postgres.Abrir(ctx, dsn)
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
		Fila: filaReserva, FilaDLQ: filaDLQ,
		Binding: "reserva.criada", LimiteEntregas: 3,
	}
	if err := topo.Declarar(canal); err != nil {
		t.Fatalf("declarar topologia: %v", err)
	}

	if _, err := canal.QueueDeclare(filaEspiao, true, false, false, false, nil); err != nil {
		t.Fatal(err)
	}
	for _, rk := range []string{"pagamento.sucesso", "pagamento.falhou"} {
		if err := canal.QueueBind(filaEspiao, rk, exchange, false, nil); err != nil {
			t.Fatal(err)
		}
	}

	canalPub, err := conexao.Channel()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := adaptamqp.NovoPublicador(canalPub, exchange, propagation.TraceContext{})
	if err != nil {
		t.Fatal(err)
	}

	return &ambiente{
		Pool: pool, Repo: postgres.NovoRepositorio(pool),
		Conexao: conexao, Canal: canal, Publica: pub,
		Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
}

func aplicarMigracao(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000001_criar_transacoes.up.sql"))
	if err != nil {
		t.Fatalf("ler migração: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(b)); err != nil {
		t.Fatalf("aplicar migração: %v", err)
	}
}

func (a *ambiente) publicarIntencao(t *testing.T, i map[string]any) {
	t.Helper()
	corpo, err := json.Marshal(i)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Canal.PublishWithContext(context.Background(), exchange, "reserva.criada",
		false, false, amqp091.Publishing{
			ContentType: "application/json", DeliveryMode: amqp091.Persistent,
			MessageId: i["reserva_id"].(string), Body: corpo,
		}); err != nil {
		t.Fatal(err)
	}
}

func intencao(reservaID, valor, forma string, expiraEm time.Duration) map[string]any {
	return map[string]any{
		"evento": "RESERVA_CRIADA", "versao": 1,
		"ocorrido_em":     time.Now().UTC().Format(time.RFC3339),
		"reserva_id":      reservaID,
		"sessao_id":       uuid.NewString(),
		"usuario_id":      uuid.NewString(),
		"poltronas_ids":   []string{"A1", "A2"},
		"valor_total":     json.Number(valor),
		"forma_pagamento": forma,
		"expira_em":       time.Now().UTC().Add(expiraEm).Format(time.RFC3339),
	}
}

func (a *ambiente) consumidorDe(t *testing.T, adq usecase.Adquirente, prefetch int) (*adaptamqp.Consumidor, context.CancelFunc) {
	t.Helper()
	canal, err := a.Conexao.Channel()
	if err != nil {
		t.Fatal(err)
	}
	c := &adaptamqp.Consumidor{
		Canal: canal, Fila: filaReserva, Prefetch: prefetch,
		Caso: usecase.ProcessarPagamento{
			Repo: a.Repo, Adquirente: adq, Publicador: a.Publica,
			Relogio: relogioReal{}, IDs: idsReais{},
			PrazoAdquirente: 2 * time.Second,
		},
		Log: a.Log, Propagador: propagation.TraceContext{},
		EmAndamento: &adaptamqp.Medidor{},
	}
	ctx, cancelar := context.WithCancel(context.Background())
	go func() { _ = c.Consumir(ctx) }()
	return c, cancelar
}

func (a *ambiente) esperarStatus(t *testing.T, reservaID string, querido transacao.Status, prazo time.Duration) transacao.Transacao {
	t.Helper()
	limite := time.Now().Add(prazo)
	var ultima transacao.Transacao
	for time.Now().Before(limite) {
		tr, err := a.Repo.BuscarPorReserva(context.Background(), reservaID)
		if err == nil {
			ultima = tr
			if tr.Status == querido {
				return tr
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("transação de %s não chegou a %s em %s (última: %+v)", reservaID, querido, prazo, ultima)
	return ultima
}

func (a *ambiente) contarFila(t *testing.T, nome string) int {
	t.Helper()
	canal, err := a.Conexao.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer canal.Close()
	q, err := canal.QueueDeclarePassive(nome, true, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	return q.Messages
}

func (a *ambiente) fatosEspiados(t *testing.T) []map[string]any {
	t.Helper()
	canal, err := a.Conexao.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer canal.Close()

	var fatos []map[string]any
	for {
		msg, ok, err := canal.Get(filaEspiao, true)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			return fatos
		}
		var m map[string]any
		if err := json.Unmarshal(msg.Body, &m); err != nil {
			t.Fatal(err)
		}
		m["__routing_key"] = msg.RoutingKey
		fatos = append(fatos, m)
	}
}

type adquirenteControlado struct {
	resultado usecase.ResultadoCobranca
	erro      error
	demora    time.Duration
	cobrancas chan string
}

func novoAdquirente(res usecase.ResultadoCobranca) *adquirenteControlado {
	return &adquirenteControlado{resultado: res, cobrancas: make(chan string, 4096)}
}

func (a *adquirenteControlado) Cobrar(ctx context.Context, c usecase.Cobranca) (usecase.ResultadoCobranca, error) {
	a.cobrancas <- c.ReservaID
	if a.demora > 0 {
		select {
		case <-time.After(a.demora):
		case <-ctx.Done():
			return usecase.ResultadoCobranca{Desfecho: usecase.Indeterminada}, nil
		}
	}
	if a.erro != nil {
		return usecase.ResultadoCobranca{}, a.erro
	}
	return a.resultado, nil
}

func (a *adquirenteControlado) total() int { return len(a.cobrancas) }
