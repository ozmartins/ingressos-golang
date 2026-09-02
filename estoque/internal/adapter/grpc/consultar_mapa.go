package grpc

import (
	"context"
	"time"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
)

type CasoDeUsoMapa interface {
	Executar(ctx context.Context, sessaoID string) ([]poltrona.Poltrona, error)
}

func (s *Servidor) ConsultarMapaPoltronas(ctx context.Context, req *pb.SolicitacaoMapa) (*pb.RespostaMapa, error) {
	inicio := time.Now()

	mapa, err := s.mapa.Executar(ctx, req.GetSessaoId())
	if err != nil {
		desfecho := classificar(err)
		s.metricas.registrar(ctx, "ConsultarMapaPoltronas", desfecho, inicio)
		s.obs.LogOperacao(ctx, "ConsultarMapaPoltronas", desfecho, inicio,
			"sessao_id", req.GetSessaoId(),
			"chamador", subjectDoChamador(ctx),
			"erro", err.Error())
		return nil, paraStatus(err, s.limite)
	}

	resposta := &pb.RespostaMapa{
		SessaoId:  req.GetSessaoId(),
		Poltronas: make([]*pb.Poltrona, 0, len(mapa)),
	}
	for _, p := range mapa {
		resposta.Poltronas = append(resposta.Poltronas, &pb.Poltrona{
			Rotulo:  p.Rotulo,
			Fileira: p.Fileira,
			Numero:  int32(p.Numero),
			Tipo:    paraTipoPB(p.Tipo),
			Status:  paraStatusPB(p.Status),
		})
	}

	s.metricas.registrar(ctx, "ConsultarMapaPoltronas", desfechoOK, inicio)
	s.obs.LogOperacao(ctx, "ConsultarMapaPoltronas", desfechoOK, inicio,
		"sessao_id", req.GetSessaoId(),
		"poltronas", len(mapa),
		"chamador", subjectDoChamador(ctx))
	return resposta, nil
}

func paraTipoPB(t poltrona.Tipo) pb.TipoPoltrona {
	switch t {
	case poltrona.Normal:
		return pb.TipoPoltrona_NORMAL
	case poltrona.PCD:
		return pb.TipoPoltrona_PCD
	case poltrona.Namoradeira:
		return pb.TipoPoltrona_NAMORADEIRA
	default:
		return pb.TipoPoltrona_TIPO_NAO_INFORMADO
	}
}

func paraStatusPB(s poltrona.Status) pb.StatusPoltrona {
	switch s {
	case poltrona.Livre:
		return pb.StatusPoltrona_LIVRE
	case poltrona.Reservada:
		return pb.StatusPoltrona_RESERVADA
	case poltrona.Ocupada:
		return pb.StatusPoltrona_OCUPADA
	default:
		return pb.StatusPoltrona_STATUS_NAO_INFORMADO
	}
}
