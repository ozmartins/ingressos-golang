package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/oseias/ingressos-golang/catalogo/internal/platform/obs"
)

// InterceptadorRequestID lê o metadado x-request-id enviado pelo chamador (o
// estoque) e o põe no contexto, para que os logs do catálogo carreguem o mesmo
// request_id da requisição de origem. Sem metadado (ou malformado), gera um.
func InterceptadorRequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		var recebido string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get(obs.ChaveRequestID); len(v) > 0 {
				recebido = v[0]
			}
		}
		return h(obs.ComRequestID(ctx, obs.RequestIDAceito(recebido)), req)
	}
}
