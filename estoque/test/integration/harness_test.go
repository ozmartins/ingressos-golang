//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/estoque/internal/adapter/postgres"
	adaptadorredis "github.com/oseias/ingressos-golang/estoque/internal/adapter/redis"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

const usuario = "c394c8b3-76a1-4328-b803-02f5923b7a15"

type registrador struct{ t *testing.T }

func (r registrador) Info(msg string, args ...any)  { r.t.Logf("INFO  %s %v", msg, args) }
func (r registrador) Warn(msg string, args ...any)  { r.t.Logf("WARN  %s %v", msg, args) }
func (r registrador) Error(msg string, args ...any) { r.t.Logf("ERROR %s %v", msg, args) }

// Cenario reúne o serviço montado sobre as dependências reais.
type Cenario struct {
	Banco     *postgres.Banco
	Pool      *pgxpool.Pool
	Reservas  *postgres.Reservas
	Poltronas *postgres.Poltronas
	Relogio   *shared.RelogioFixo

	Bloquear    usecase.BloquearPoltronas
	Consultar   usecase.ConsultarMapa
	Confirmar   usecase.ConfirmarReserva
	Cancelar    usecase.CancelarReserva
	Expirar     usecase.ExpirarReservas
	Provisionar usecase.ProvisionarSessao
}

// montarCenario abre o banco e monta os casos de uso com relógio controlável.
// Cada teste usa uma sessão própria, então não há necessidade de limpar tabelas
// entre eles — e isso mantém a suíte paralelizável.
func montarCenario(t *testing.T, comPrazo bool) *Cenario {
	t.Helper()
	ctx := context.Background()

	banco, err := postgres.Abrir(ctx, ambiente.DatabaseURL)
	if err != nil {
		t.Fatalf("abrir banco: %v", err)
	}
	t.Cleanup(banco.Fechar)

	var prazo usecase.IndiceDePrazo
	if comPrazo {
		obs, err := observability.Iniciar(ctx, "error", "")
		if err != nil {
			t.Fatalf("observabilidade: %v", err)
		}
		indice, err := adaptadorredis.Abrir(ctx, ambiente.RedisURL, obs)
		if err != nil {
			t.Fatalf("abrir redis: %v", err)
		}
		t.Cleanup(indice.Fechar)
		prazo = indice
	}

	reservas := postgres.NovoRepositorioReservas(banco)
	poltronas := postgres.NovoRepositorioPoltronas(banco)
	relogio := shared.NovoRelogioFixo(time.Now().UTC())
	log := registrador{t: t}

	return &Cenario{
		Banco: banco, Pool: banco.Pool(), Reservas: reservas, Poltronas: poltronas, Relogio: relogio,
		Bloquear: usecase.BloquearPoltronas{
			Reservas: reservas, Prazo: prazo, Relogio: relogio, Log: log,
			TTL: 10 * time.Minute, Limite: 10,
			TraceContextDe: func(context.Context) map[string]string {
				return map[string]string{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}
			},
		},
		Consultar:   usecase.ConsultarMapa{Poltronas: poltronas},
		Confirmar:   usecase.ConfirmarReserva{Reservas: reservas, Prazo: prazo, Relogio: relogio, Log: log},
		Cancelar:    usecase.CancelarReserva{Reservas: reservas, Prazo: prazo, Relogio: relogio, Log: log},
		Expirar:     usecase.ExpirarReservas{Reservas: reservas, Prazo: prazo, Relogio: relogio, Log: log, LotePorVarredura: 500},
		Provisionar: usecase.ProvisionarSessao{Poltronas: poltronas, Log: log},
	}
}

// novaSessao provisiona uma sessão com fileiras × assentos poltronas e devolve
// o identificador. Usa o caminho real de provisionamento (FR-033).
func (c *Cenario) novaSessao(t *testing.T, fileiras []string, assentos int) string {
	t.Helper()

	sessaoID := uuid.NewString()
	evento := usecase.EventoSessaoCriada{
		Evento: "SESSAO_CRIADA", Versao: 1, SessaoID: sessaoID,
		OcorridoEm: time.Now().UTC().Format(time.RFC3339),
	}
	for _, fileira := range fileiras {
		for n := 1; n <= assentos; n++ {
			evento.Poltronas = append(evento.Poltronas,
				usecase.LayoutPoltrona{Fileira: fileira, Numero: n, Tipo: "NORMAL"})
		}
	}

	res, err := c.Provisionar.Executar(context.Background(), "estoque.sessao-criada", sessaoID, evento)
	if err != nil {
		t.Fatalf("provisionar sessão: %v", err)
	}
	if res != usecase.TransicaoAplicada {
		t.Fatalf("provisionamento devolveu %s", res)
	}
	return sessaoID
}

// statusPoltrona lê o estado corrente direto do banco.
func (c *Cenario) statusPoltrona(t *testing.T, sessaoID, rotulo string) poltrona.Status {
	t.Helper()
	var status string
	err := c.Pool.QueryRow(context.Background(),
		`SELECT status FROM poltronas WHERE sessao_id = $1 AND rotulo = $2`, sessaoID, rotulo).Scan(&status)
	if err != nil {
		t.Fatalf("ler poltrona %s: %v", rotulo, err)
	}
	return poltrona.Status(status)
}

// statusReserva lê o estado corrente da reserva.
func (c *Cenario) statusReserva(t *testing.T, reservaID string) string {
	t.Helper()
	var status string
	err := c.Pool.QueryRow(context.Background(),
		`SELECT status FROM reservas WHERE id = $1`, reservaID).Scan(&status)
	if err != nil {
		t.Fatalf("ler reserva %s: %v", reservaID, err)
	}
	return status
}

// contarPorStatus conta as poltronas de uma sessão em cada estado.
func (c *Cenario) contarPorStatus(t *testing.T, sessaoID string) map[string]int {
	t.Helper()
	linhas, err := c.Pool.Query(context.Background(),
		`SELECT status, count(*) FROM poltronas WHERE sessao_id = $1 GROUP BY status`, sessaoID)
	if err != nil {
		t.Fatalf("contar poltronas: %v", err)
	}
	defer linhas.Close()

	contagem := map[string]int{}
	for linhas.Next() {
		var status string
		var n int
		if err := linhas.Scan(&status, &n); err != nil {
			t.Fatalf("contar poltronas: %v", err)
		}
		contagem[status] = n
	}
	return contagem
}
