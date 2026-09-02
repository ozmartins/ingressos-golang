//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	estoquepb "github.com/oseias/ingressos-golang/catalogo/gen/pb/estoque"
	estoqueadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/estoque"
	adapterhttp "github.com/oseias/ingressos-golang/catalogo/internal/adapter/http"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
	pgadapter "github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

const sessaoAgendada = "e1b2c3d4-0000-4000-8000-000000000001"

type estoqueSimulado struct {
	estoquepb.UnimplementedServicoEstoqueServer

	mu            sync.Mutex
	concedidas    map[string]bool
	chamadas      int
	traceRecebido string

	atraso time.Duration
}

func (e *estoqueSimulado) BloquearPoltronas(ctx context.Context, s *estoquepb.SolicitacaoBloqueio) (*estoquepb.RespostaBloqueio, error) {
	e.mu.Lock()
	e.chamadas++
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("traceparent"); len(v) > 0 {
			e.traceRecebido = v[0]
		}
	}
	e.mu.Unlock()

	if e.atraso > 0 {
		select {
		case <-time.After(e.atraso):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.concedidas == nil {
		e.concedidas = map[string]bool{}
	}
	for _, p := range s.GetPoltronasIds() {
		if e.concedidas[p] {
			return &estoquepb.RespostaBloqueio{Sucesso: false, Mensagem: "poltrona ocupada"}, nil
		}
	}
	for _, p := range s.GetPoltronasIds() {
		e.concedidas[p] = true
	}
	return &estoquepb.RespostaBloqueio{
		Sucesso:   true,
		ReservaId: "reserva-" + s.GetPoltronasIds()[0],
		ExpiraEm:  time.Now().Add(10 * time.Minute).Unix(),
	}, nil
}

type verificadorFixo struct{}

func (verificadorFixo) Verificar(_ context.Context, token string) (identidade.Identidade, error) {
	if token != "token-bom" {
		return identidade.Identidade{}, identidade.ErrCredencialInvalida
	}
	return identidade.Identidade{UsuarioID: "usuario-1"}, nil
}

func montarServico(t *testing.T, sim *estoqueSimulado) (*httptest.Server, *estoqueSimulado) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	estoquepb.RegisterServicoEstoqueServer(srv, sim)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet", append(
		estoqueadapter.OpcoesDeConexao(),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	metricas, err := observability.NovasMetricas()
	if err != nil {
		t.Fatal(err)
	}
	cliente := estoqueadapter.NovoClienteComConexao(conn, estoqueadapter.Opcoes{
		Timeout: 2 * time.Second, FalhasParaAbrir: 5, IntervaloAberto: 30 * time.Second, Metricas: metricas,
	})

	sessoes := pgadapter.NovoSessaoRepository(pool)
	cinemas := pgadapter.NovoCinemaRepository(pool)

	router := adapterhttp.NovoRouter(adapterhttp.Dependencias{
		Handlers: adapterhttp.Handlers{
			ListarFilmes:     usecase.ListarFilmes{Repo: pgadapter.NovoFilmeRepository(pool)},
			ListarCinemas:    usecase.ListarCinemas{Repo: cinemas},
			ListarSalas:      usecase.ListarSalas{Cinemas: cinemas, Salas: pgadapter.NovoSalaRepository(pool)},
			ConsultarSessoes: usecase.ConsultarSessoes{Repo: sessoes},
			ReservarPoltronas: usecase.ReservarPoltronas{
				Sessoes: sessoes, Estoque: cliente,
				Agora: func() time.Time { return time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) },
			},
			Limites: adapterhttp.LimitesPaginacao{Padrao: 20, Maximo: 100},
		},
		Saude:       func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		Verificador: verificadorFixo{},
		Metricas:    metricas,
	})
	s := httptest.NewServer(router)
	t.Cleanup(s.Close)
	return s, sim
}

func pedirReserva(t *testing.T, s *httptest.Server, token, sessao string, poltronas []string, cabecalhos map[string]string) (int, []byte) {
	t.Helper()
	corpo, _ := json.Marshal(map[string]any{"poltronas_ids": poltronas})
	req, err := http.NewRequest(http.MethodPost, s.URL+"/api/v1/sessoes/"+sessao+"/reservar", bytes.NewReader(corpo))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range cabecalhos {
		req.Header.Set(k, v)
	}
	resp, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	dados, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, dados
}

func TestReservaPontaAPonta(t *testing.T) {
	carregarFixtures(t)
	s, sim := montarServico(t, &estoqueSimulado{})

	codigo, corpo := pedirReserva(t, s, "token-bom", sessaoAgendada, []string{"A1", "A2"}, nil)
	if codigo != http.StatusCreated {
		t.Fatalf("esperava 201, obteve %d (%s)", codigo, corpo)
	}

	codigo, _ = pedirReserva(t, s, "token-bom", sessaoAgendada, []string{"A1"}, nil)
	if codigo != http.StatusConflict {
		t.Fatalf("esperava 409 na segunda tentativa, obteve %d", codigo)
	}
	if sim.chamadas != 2 {
		t.Fatalf("esperava 2 chamadas ao estoque, obteve %d", sim.chamadas)
	}
}

func TestRastreamentoPropagaAteOEstoque(t *testing.T) {
	carregarFixtures(t)
	s, sim := montarServico(t, &estoqueSimulado{})

	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	traceparent := "00-" + traceID + "-00f067aa0ba902b7-01"

	codigo, corpo := pedirReserva(t, s, "token-bom", sessaoAgendada, []string{"C1"},
		map[string]string{"traceparent": traceparent})
	if codigo != http.StatusCreated {
		t.Fatalf("esperava 201, obteve %d (%s)", codigo, corpo)
	}

	sim.mu.Lock()
	recebido := sim.traceRecebido
	sim.mu.Unlock()

	if recebido == "" {
		t.Fatal("o estoque não recebeu contexto de rastreamento algum")
	}
	if len(recebido) < 36 || recebido[3:35] != traceID {
		t.Fatalf("o trace_id não foi propagado: enviado %s, recebido %q", traceID, recebido)
	}
}

func TestRecusasLocaisNaoContatamOEstoque(t *testing.T) {
	carregarFixtures(t)

	casos := []struct {
		nome      string
		token     string
		sessao    string
		poltronas []string
		codigo    int
	}{
		{"sem credencial", "", sessaoAgendada, []string{"A1"}, http.StatusUnauthorized},
		{"credencial inválida", "token-ruim", sessaoAgendada, []string{"A1"}, http.StatusUnauthorized},
		{"sessão inexistente", "token-bom", "00000000-0000-0000-0000-000000000000", []string{"A1"}, http.StatusNotFound},
		{"sessão cancelada", "token-bom", "e1b2c3d4-0000-4000-8000-000000000005", []string{"A1"}, http.StatusUnprocessableEntity},
		{"poltronas duplicadas", "token-bom", sessaoAgendada, []string{"A1", "A1"}, http.StatusBadRequest},
		{"lista vazia", "token-bom", sessaoAgendada, []string{}, http.StatusBadRequest},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			s, sim := montarServico(t, &estoqueSimulado{})
			codigo, corpo := pedirReserva(t, s, c.token, c.sessao, c.poltronas, nil)
			if codigo != c.codigo {
				t.Fatalf("esperava %d, obteve %d (%s)", c.codigo, codigo, corpo)
			}
			if sim.chamadas != 0 {
				t.Fatalf("o estoque foi contatado %d vez(es)", sim.chamadas)
			}
		})
	}
}

func TestConcorrenciaConfirmaNoMaximoUma(t *testing.T) {
	carregarFixtures(t)
	s, _ := montarServico(t, &estoqueSimulado{})

	const paralelas = 50
	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		confirmadas int
		conflitos   int
		outros      []int
	)
	inicio := make(chan struct{})

	for i := 0; i < paralelas; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-inicio
			codigo, _ := pedirReserva(t, s, "token-bom", sessaoAgendada, []string{"Z9"}, nil)
			mu.Lock()
			defer mu.Unlock()
			switch codigo {
			case http.StatusCreated:
				confirmadas++
			case http.StatusConflict:
				conflitos++
			default:
				outros = append(outros, codigo)
			}
		}()
	}
	close(inicio)
	wg.Wait()

	if confirmadas != 1 {
		t.Fatalf("esperava exatamente 1 confirmação em %d tentativas, obteve %d", paralelas, confirmadas)
	}
	if conflitos != paralelas-1 {
		t.Fatalf("esperava %d conflitos, obteve %d (outros: %v)", paralelas-1, conflitos, outros)
	}
}

func TestEstoqueLentoDevolve503DentroDoOrcamento(t *testing.T) {
	carregarFixtures(t)
	s, _ := montarServico(t, &estoqueSimulado{atraso: 10 * time.Second})

	comeco := time.Now()
	codigo, corpo := pedirReserva(t, s, "token-bom", sessaoAgendada, []string{"D1"}, nil)
	decorrido := time.Since(comeco)

	if codigo != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, obteve %d (%s)", codigo, corpo)
	}
	if decorrido > 2500*time.Millisecond {
		t.Fatalf("resposta levou %s; SC-004 exige desfecho conclusivo em menos de 2,5s", decorrido)
	}

	var p struct {
		Type   string `json:"type"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(corpo, &p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "https://cinema.example/errors/estoque-indisponivel" {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestNavegacaoFuncionaComEstoqueForaDoAr(t *testing.T) {
	carregarFixtures(t)
	s, _ := montarServico(t, &estoqueSimulado{atraso: 10 * time.Second})

	if codigo, _ := pedirReserva(t, s, "token-bom", sessaoAgendada, []string{"E1"}, nil); codigo != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503 na reserva, obteve %d", codigo)
	}

	for _, caminho := range []string{"/api/v1/filmes", "/api/v1/sessoes", "/api/v1/cinemas"} {
		resp, err := s.Client().Get(s.URL + caminho)
		if err != nil {
			t.Fatalf("%s: %v", caminho, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: esperava 200 com o estoque fora do ar, obteve %d", caminho, resp.StatusCode)
		}
	}
}
