package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
)

// VerificadorDeToken valida um JWT e devolve a identidade com as permissões.
type VerificadorDeToken interface {
	Verificar(ctx context.Context, token string) (identidade.Identidade, error)
}

// prefixoSaude identifica o serviço padrão de health checking do gRPC, que
// fica aberto para sondas de infraestrutura.
const prefixoSaude = "/grpc.health.v1.Health/"

// InterceptadorAutenticacao exige um Bearer token válido no metadata
// `authorization` de toda chamada (exceto health). Token ausente, inválido ou
// expirado => Unauthenticated (equivalente ao 401); token válido sem a
// permissão pedida => PermissionDenied (equivalente ao 403).
func InterceptadorAutenticacao(v VerificadorDeToken, permissao string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, prefixoSaude) {
			return h(ctx, req)
		}
		token, ok := tokenDoMetadata(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "credencial ausente ou malformada")
		}
		id, err := v.Verificar(ctx, token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "credencial inválida ou expirada")
		}
		if !id.Tem(permissao) {
			return nil, status.Error(codes.PermissionDenied, "credencial sem permissão para esta operação")
		}
		return h(ctx, req)
	}
}

func tokenDoMetadata(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	for _, v := range md.Get("authorization") {
		partes := strings.SplitN(v, " ", 2)
		if len(partes) == 2 && strings.EqualFold(partes[0], "Bearer") && strings.TrimSpace(partes[1]) != "" {
			return strings.TrimSpace(partes[1]), true
		}
	}
	return "", false
}
