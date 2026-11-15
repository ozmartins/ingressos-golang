package grpc

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/oseias/ingressos-golang/catalogo/gen/pb/catalogo/v1"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type buscadorFalso struct {
	sessao catalogo.Sessao
	err    error
}

func (b buscadorFalso) Executar(context.Context, string) (catalogo.Sessao, error) {
	return b.sessao, b.err
}

func cliente(t *testing.T, b BuscadorDeSessao) pb.CatalogoServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	pb.RegisterCatalogoServiceServer(srv, &Servidor{BuscarSessao: b})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewCatalogoServiceClient(conn)
}

const idValido = "0b3f6c1e-8a53-4c53-9a1e-3d1f0f5a7a11"

func TestObterSessaoDevolveOsCamposDaSessao(t *testing.T) {
	inicio := time.Date(2030, 5, 1, 21, 30, 0, 0, time.UTC)
	preco := catalogo.DinheiroDeCentavos(3250)
	c := cliente(t, buscadorFalso{sessao: catalogo.Sessao{
		ID: idValido, Status: catalogo.SessaoAgendada, DataHoraInicio: inicio, PrecoBase: preco,
	}})

	got, err := c.ObterSessao(context.Background(), &pb.ObterSessaoRequest{Id: idValido})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if got.GetId() != idValido || got.GetStatus() != "AGENDADA" || got.GetPrecoBase() != "32.50" ||
		!got.GetDataHoraInicio().AsTime().Equal(inicio) {
		t.Fatalf("resposta inesperada: %v", got)
	}
}

func TestObterSessaoTraduzErros(t *testing.T) {
	casos := []struct {
		nome string
		id   string
		err  error
		want codes.Code
	}{
		{"id que não é UUID", "abc", nil, codes.InvalidArgument},
		{"sessão inexistente", idValido, shared.NaoEncontrado("sessao", idValido), codes.NotFound},
		{"prazo esgotado", idValido, context.DeadlineExceeded, codes.DeadlineExceeded},
		{"falha do banco", idValido, errors.New("conexão recusada"), codes.Internal},
	}
	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			c := cliente(t, buscadorFalso{err: tc.err})
			_, err := c.ObterSessao(context.Background(), &pb.ObterSessaoRequest{Id: tc.id})
			if got := status.Code(err); got != tc.want {
				t.Fatalf("código = %v, quero %v (%v)", got, tc.want, err)
			}
		})
	}
}
