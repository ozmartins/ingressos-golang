package grpc

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/oseias/ingressos-golang/catalogo/gen/pb/catalogo/v1"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type verificadorPorToken map[string]identidade.Identidade

func (v verificadorPorToken) Verificar(_ context.Context, t string) (identidade.Identidade, error) {
	if id, ok := v[t]; ok {
		return id, nil
	}
	return identidade.Identidade{}, errors.New("token inválido")
}

func clienteAutenticado(t *testing.T) pb.CatalogoServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	v := verificadorPorToken{
		"bom":       {UsuarioID: "estoque", Permissoes: []string{identidade.PermissaoLerSessao}},
		"sem-papel": {UsuarioID: "outro", Permissoes: []string{"qualquer-coisa"}},
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(InterceptadorAutenticacao(v, identidade.PermissaoLerSessao)))
	pb.RegisterCatalogoServiceServer(srv, &Servidor{BuscarSessao: buscadorFalso{sessao: catalogo.Sessao{ID: "x"}}})
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

func TestInterceptadorDistingue401de403(t *testing.T) {
	const id = "0b5f2c1e-8f5e-4c1e-9a55-3a1f7a3c2d10"
	casos := []struct {
		nome, auth string
		esperado   codes.Code
	}{
		{"sem token", "", codes.Unauthenticated},
		{"token inválido", "Bearer lixo", codes.Unauthenticated},
		{"esquema errado", "Basic bom", codes.Unauthenticated},
		{"token válido sem permissão", "Bearer sem-papel", codes.PermissionDenied},
		{"token com permissão", "Bearer bom", codes.OK},
	}
	c := clienteAutenticado(t)
	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			ctx := context.Background()
			if tc.auth != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", tc.auth)
			}
			_, err := c.ObterSessao(ctx, &pb.ObterSessaoRequest{Id: id})
			if got := status.Code(err); got != tc.esperado {
				t.Errorf("código = %v (%v), esperado %v", got, err, tc.esperado)
			}
		})
	}
}
