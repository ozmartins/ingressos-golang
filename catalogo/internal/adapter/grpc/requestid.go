package grpc

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/oseias/ingressos-golang/catalogo/internal/platform/obs"
)

// InterceptadorRequestID lê o metadado x-request-id enviado pelo chamador (o
// estoque) e o põe no contexto, para que os logs do catálogo carreguem o mesmo
// request_id da requisição de origem. Sem metadado (ou malformado), gera um.
// Registra uma linha por chamada (método, código, duração) já com o request_id,
// exceto as sondas de saúde do gRPC, que são frequentes e sem interesse.
func InterceptadorRequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		var recebido string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get(obs.ChaveRequestID); len(v) > 0 {
				recebido = v[0]
			}
		}
		ctx = obs.ComRequestID(ctx, obs.RequestIDAceito(recebido))
		inicio := time.Now()
		resp, err := h(ctx, req)
		if !strings.HasPrefix(info.FullMethod, "/grpc.health.") {
			slog.InfoContext(ctx, "chamada gRPC concluída", "metodo", info.FullMethod,
				"codigo", status.Code(err).String(), "duration_ms", float64(time.Since(inicio).Microseconds())/1000)
		}
		return resp, err
	}
}
