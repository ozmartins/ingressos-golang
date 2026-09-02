package estoque

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	estoquepb "github.com/oseias/ingressos-golang/catalogo/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type estoqueSimulado struct {
	estoquepb.UnimplementedServicoEstoqueServer

	mu                sync.Mutex
	chamadas          int
	ultimaSolicitacao *estoquepb.SolicitacaoBloqueio

	resposta *estoquepb.RespostaBloqueio
	erro     error
	atraso   time.Duration
}

func (e *estoqueSimulado) chamadasFeitas() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.chamadas
}

func (e *estoqueSimulado) solicitacaoRecebida() *estoquepb.SolicitacaoBloqueio {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.ultimaSolicitacao
}

func (e *estoqueSimulado) BloquearPoltronas(ctx context.Context, s *estoquepb.SolicitacaoBloqueio) (*estoquepb.RespostaBloqueio, error) {
	e.mu.Lock()
	e.chamadas++
	e.ultimaSolicitacao = s
	e.mu.Unlock()
	e.mu.Lock()
	atraso := e.atraso
	e.mu.Unlock()
	if atraso > 0 {
		select {
		case <-time.After(atraso):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if e.erro != nil {
		return nil, e.erro
	}
	return e.resposta, nil
}

func subirSimulado(t *testing.T, sim *estoquepb.RespostaBloqueio) (*estoqueSimulado, *grpc.ClientConn) {
	t.Helper()
	s := &estoqueSimulado{resposta: sim}
	return s, conectar(t, s)
}

func conectar(t *testing.T, sim *estoqueSimulado) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	estoquepb.RegisterServicoEstoqueServer(srv, sim)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func solicitacao() reserva.SolicitacaoReserva {
	return reserva.SolicitacaoReserva{SessaoID: "s1", PoltronasIDs: []string{"A1", "A2"}, UsuarioID: "u1"}
}

func clienteCom(conn *grpc.ClientConn, timeout time.Duration, falhas uint32, intervalo time.Duration) *Cliente {
	return NovoClienteComConexao(conn, Opcoes{
		Timeout: timeout, FalhasParaAbrir: falhas, IntervaloAberto: intervalo,
	})
}

func TestBloquearSucessoEncaminhaTodosOsCampos(t *testing.T) {
	expira := time.Now().Add(10 * time.Minute).Unix()
	sim, conn := subirSimulado(t, &estoquepb.RespostaBloqueio{
		Sucesso: true, ReservaId: "r1", ExpiraEm: expira,
	})
	c := clienteCom(conn, 2*time.Second, 5, time.Second)

	r, err := c.BloquearPoltronas(context.Background(), solicitacao())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r.ReservaID != "r1" || r.ExpiraEm.Unix() != expira {
		t.Fatalf("resultado inesperado: %+v", r)
	}
	got := sim.solicitacaoRecebida()
	if got.GetSessaoId() != "s1" || got.GetUsuarioId() != "u1" || len(got.GetPoltronasIds()) != 2 {
		t.Fatalf("solicitação encaminhada incompleta: %+v", got)
	}
}

func TestBloquearSemSucessoViraPoltronasIndisponiveis(t *testing.T) {
	_, conn := subirSimulado(t, &estoquepb.RespostaBloqueio{Sucesso: false, Mensagem: "ocupada"})
	c := clienteCom(conn, 2*time.Second, 5, time.Second)

	if _, err := c.BloquearPoltronas(context.Background(), solicitacao()); !errors.Is(err, shared.ErrPoltronasIndisponiveis) {
		t.Fatalf("esperava ErrPoltronasIndisponiveis, obteve %v", err)
	}
}

func TestBloquearSucessoIncompletoViraErroDoParceiro(t *testing.T) {
	casos := map[string]*estoquepb.RespostaBloqueio{
		"sem reserva_id": {Sucesso: true, ExpiraEm: time.Now().Unix()},
		"sem expira_em":  {Sucesso: true, ReservaId: "r1"},
	}
	for nome, resp := range casos {
		t.Run(nome, func(t *testing.T) {
			_, conn := subirSimulado(t, resp)
			c := clienteCom(conn, 2*time.Second, 5, time.Second)
			if _, err := c.BloquearPoltronas(context.Background(), solicitacao()); !errors.Is(err, shared.ErrRespostaInvalidaDoParceiro) {
				t.Fatalf("esperava ErrRespostaInvalidaDoParceiro, obteve %v", err)
			}
		})
	}
}

func TestBloquearRespeitaOTimeout(t *testing.T) {
	sim := &estoqueSimulado{atraso: 5 * time.Second, resposta: &estoquepb.RespostaBloqueio{Sucesso: true}}
	c := clienteCom(conectar(t, sim), 300*time.Millisecond, 100, time.Second)

	inicio := time.Now()
	_, err := c.BloquearPoltronas(context.Background(), solicitacao())
	decorrido := time.Since(inicio)

	if !errors.Is(err, shared.ErrEstoqueIndisponivel) {
		t.Fatalf("esperava ErrEstoqueIndisponivel, obteve %v", err)
	}
	if decorrido > time.Second {
		t.Fatalf("esperou %s, muito além do timeout configurado", decorrido)
	}
}

func TestBloquearNaoRetenta(t *testing.T) {
	sim := &estoqueSimulado{atraso: 2 * time.Second, resposta: &estoquepb.RespostaBloqueio{Sucesso: true}}
	c := clienteCom(conectar(t, sim), 200*time.Millisecond, 100, time.Second)

	_, _ = c.BloquearPoltronas(context.Background(), solicitacao())
	time.Sleep(300 * time.Millisecond)

	if sim.chamadasFeitas() != 1 {
		t.Fatalf("o estoque recebeu %d chamadas para uma única solicitação", sim.chamadasFeitas())
	}
}

func TestRecusaRapidaAbreEDepoisRetomaSozinha(t *testing.T) {
	sim := &estoqueSimulado{atraso: time.Second, resposta: &estoquepb.RespostaBloqueio{Sucesso: true, ReservaId: "r1", ExpiraEm: time.Now().Unix()}}
	const intervaloAberto = 400 * time.Millisecond
	c := clienteCom(conectar(t, sim), 100*time.Millisecond, 3, intervaloAberto)

	for i := 0; i < 3; i++ {
		if _, err := c.BloquearPoltronas(context.Background(), solicitacao()); !errors.Is(err, shared.ErrEstoqueIndisponivel) {
			t.Fatalf("chamada %d: esperava indisponibilidade, obteve %v", i+1, err)
		}
	}
	chamadasAteAbrir := sim.chamadasFeitas()

	inicio := time.Now()
	_, err := c.BloquearPoltronas(context.Background(), solicitacao())
	recusaRapida := time.Since(inicio)

	if !errors.Is(err, shared.ErrEstoqueIndisponivel) {
		t.Fatalf("com a recusa rápida ativa, esperava indisponibilidade, obteve %v", err)
	}
	if recusaRapida > 50*time.Millisecond {
		t.Fatalf("recusa rápida levou %s; deveria ser imediata", recusaRapida)
	}
	if sim.chamadasFeitas() != chamadasAteAbrir {
		t.Fatal("com a recusa rápida ativa, o estoque não deveria ser contatado")
	}

	sim.mu.Lock()
	sim.atraso = 0
	sim.mu.Unlock()
	time.Sleep(intervaloAberto + 100*time.Millisecond)

	if _, err := c.BloquearPoltronas(context.Background(), solicitacao()); err != nil {
		t.Fatalf("após o intervalo, a chamada deveria voltar a passar: %v", err)
	}
}

func TestPoltronaOcupadaNaoAbreRecusaRapida(t *testing.T) {
	sim := &estoqueSimulado{resposta: &estoquepb.RespostaBloqueio{Sucesso: false}}
	c := clienteCom(conectar(t, sim), time.Second, 2, time.Minute)

	for i := 0; i < 5; i++ {
		if _, err := c.BloquearPoltronas(context.Background(), solicitacao()); !errors.Is(err, shared.ErrPoltronasIndisponiveis) {
			t.Fatalf("chamada %d: obteve %v", i+1, err)
		}
	}
	if sim.chamadasFeitas() != 5 {
		t.Fatalf("o estoque recebeu %d chamadas; a recusa rápida abriu indevidamente", sim.chamadasFeitas())
	}
}
