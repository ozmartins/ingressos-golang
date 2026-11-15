package catalogo

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/catalogo/v1"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// servidorFalso implementa o CatalogoService com a resposta que o teste quiser.
type servidorFalso struct {
	pb.UnimplementedCatalogoServiceServer
	resp     func(ctx context.Context, id string) (*pb.Sessao, error)
	chamadas atomic.Int32
	ultimoID atomic.Value
}

func (s *servidorFalso) ObterSessao(ctx context.Context, r *pb.ObterSessaoRequest) (*pb.Sessao, error) {
	s.chamadas.Add(1)
	s.ultimoID.Store(r.GetId())
	return s.resp(ctx, r.GetId())
}

func sessaoValida() *pb.Sessao {
	return &pb.Sessao{
		Id: "s1", Status: "AGENDADA", PrecoBase: "42.00",
		DataHoraInicio: timestamppb.New(time.Date(2026, 9, 20, 19, 30, 0, 0, time.UTC)),
	}
}

func clienteCom(t *testing.T, resp func(ctx context.Context, id string) (*pb.Sessao, error)) (*Cliente, *servidorFalso) {
	t.Helper()
	falso := &servidorFalso{resp: resp}
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	pb.RegisterCatalogoServiceServer(srv, falso)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return NovoClienteComAPI(Opcoes{Timeout: 200 * time.Millisecond, FalhasParaAbrir: 2, IntervaloAberto: time.Hour},
		pb.NewCatalogoServiceClient(conn)), falso
}

func TestBuscarSessaoLeOsCamposDoCatalogo(t *testing.T) {
	c, falso := clienteCom(t, func(context.Context, string) (*pb.Sessao, error) { return sessaoValida(), nil })

	s, err := c.BuscarSessao(context.Background(), "s1")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if id := falso.ultimoID.Load(); id != "s1" {
		t.Errorf("id enviado = %v", id)
	}
	if s.ID != "s1" || s.Status != "AGENDADA" || s.PrecoBase != "42.00" ||
		s.Inicio.Format(time.RFC3339) != "2026-09-20T19:30:00Z" {
		t.Errorf("sessão = %+v", s)
	}
}

func TestBuscarSessaoTraduzErros(t *testing.T) {
	casos := map[string]struct {
		resp     func(context.Context, string) (*pb.Sessao, error)
		esperado error
	}{
		"not found": {func(context.Context, string) (*pb.Sessao, error) {
			return nil, status.Error(codes.NotFound, "x")
		}, shared.ErrSessaoNaoEncontrada},
		"unavailable": {func(context.Context, string) (*pb.Sessao, error) {
			return nil, status.Error(codes.Unavailable, "x")
		}, shared.ErrCatalogoIndisponivel},
		"internal": {func(context.Context, string) (*pb.Sessao, error) {
			return nil, status.Error(codes.Internal, "x")
		}, shared.ErrRespostaInvalidaDoCatalogo},
		"invalid argument": {func(context.Context, string) (*pb.Sessao, error) {
			return nil, status.Error(codes.InvalidArgument, "x")
		}, shared.ErrRespostaInvalidaDoCatalogo},
		"sessão incompleta": {func(context.Context, string) (*pb.Sessao, error) {
			return &pb.Sessao{Status: "AGENDADA"}, nil
		}, shared.ErrRespostaInvalidaDoCatalogo},
		"lento": {func(ctx context.Context, _ string) (*pb.Sessao, error) {
			select {
			case <-time.After(2 * time.Second):
				return sessaoValida(), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}, shared.ErrCatalogoIndisponivel},
	}
	for nome, tc := range casos {
		t.Run(nome, func(t *testing.T) {
			c, _ := clienteCom(t, tc.resp)
			_, err := c.BuscarSessao(context.Background(), "s1")
			if !errors.Is(err, tc.esperado) {
				t.Errorf("err = %v, esperado %v", err, tc.esperado)
			}
		})
	}
}

func TestBreakerAbreComFalhasConsecutivasENaoComSessaoInexistente(t *testing.T) {
	codigo := codes.NotFound
	c, falso := clienteCom(t, func(context.Context, string) (*pb.Sessao, error) {
		return nil, status.Error(codigo, "x")
	})

	for range 5 {
		_, _ = c.BuscarSessao(context.Background(), "s1")
	}
	if falso.chamadas.Load() != 5 {
		t.Fatalf("NotFound não deve abrir o breaker; chamadas = %d", falso.chamadas.Load())
	}

	codigo = codes.Unavailable
	for range 5 {
		_, err := c.BuscarSessao(context.Background(), "s1")
		if !errors.Is(err, shared.ErrCatalogoIndisponivel) {
			t.Fatalf("err = %v, esperado ErrCatalogoIndisponivel", err)
		}
	}
	if got := falso.chamadas.Load(); got != 7 {
		t.Errorf("chamadas = %d, esperado 7 (5 + 2 falhas até abrir; o resto recusado sem rede)", got)
	}
}
