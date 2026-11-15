// Package grpc expõe o catálogo a outros serviços por gRPC. É um adaptador de
// entrada, irmão do adaptador HTTP: traduz o protocolo e delega ao mesmo caso
// de uso, sem regra de negócio própria.
package grpc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/oseias/ingressos-golang/catalogo/gen/pb/catalogo/v1"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// BuscadorDeSessao é a parte do caso de uso BuscarSessao que o adaptador usa.
type BuscadorDeSessao interface {
	Executar(ctx context.Context, sessaoID string) (catalogo.Sessao, error)
}

type Servidor struct {
	pb.UnimplementedCatalogoServiceServer
	BuscarSessao BuscadorDeSessao
}

func (s *Servidor) ObterSessao(ctx context.Context, req *pb.ObterSessaoRequest) (*pb.Sessao, error) {
	if _, err := uuid.Parse(req.GetId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "id deve ser um UUID")
	}

	sessao, err := s.BuscarSessao.Executar(ctx, req.GetId())
	switch {
	case err == nil:
	case errors.Is(err, shared.ErrNaoEncontrado):
		return nil, status.Error(codes.NotFound, "sessão não encontrada")
	case errors.Is(err, context.Canceled):
		return nil, status.Error(codes.Canceled, "chamada cancelada")
	case errors.Is(err, context.DeadlineExceeded):
		return nil, status.Error(codes.DeadlineExceeded, "prazo da chamada esgotado")
	default:
		// O detalhe vai para o log, não para o cliente.
		slog.ErrorContext(ctx, "falha ao buscar sessão via gRPC", "erro", err, "sessao_id", req.GetId())
		return nil, status.Error(codes.Internal, "erro interno")
	}

	return &pb.Sessao{
		Id:             sessao.ID,
		Status:         string(sessao.Status),
		DataHoraInicio: timestamppb.New(sessao.DataHoraInicio.UTC()),
		PrecoBase:      sessao.PrecoBase.String(),
	}, nil
}
