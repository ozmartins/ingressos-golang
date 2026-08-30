package grpc

import (
	"context"
	"errors"
	"time"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

// CasoDeUsoBloqueio é a porta de entrada do bloqueio, do ponto de vista do
// adaptador. Declará-la aqui mantém o adaptador dependente de uma abstração
// própria, não da estrutura concreta do caso de uso.
type CasoDeUsoBloqueio interface {
	Executar(ctx context.Context, sessaoID, usuarioID string, rotulos []string) (usecase.ResultadoBloqueio, error)
}

// BloquearPoltronas atende a RPC de bloqueio.
func (s *Servidor) BloquearPoltronas(ctx context.Context, req *pb.SolicitacaoBloqueio) (*pb.RespostaBloqueio, error) {
	inicio := time.Now()

	resultado, err := s.bloqueio.Executar(ctx, req.GetSessaoId(), req.GetUsuarioId(), req.GetPoltronasIds())
	if err != nil {
		desfecho := classificar(err)
		s.metricas.registrar(ctx, "BloquearPoltronas", desfecho, inicio)
		s.obs.LogOperacao(ctx, "BloquearPoltronas", desfecho, inicio,
			"sessao_id", req.GetSessaoId(),
			"poltronas", len(req.GetPoltronasIds()),
			"chamador", subjectDoChamador(ctx),
			// O erro completo fica no log, nunca na resposta (princípio IV).
			"erro", err.Error())
		return nil, paraStatus(err, s.limite)
	}

	if !resultado.Concedido {
		s.metricas.registrar(ctx, "BloquearPoltronas", desfechoIndisponivel, inicio)
		s.obs.LogOperacao(ctx, "BloquearPoltronas", desfechoIndisponivel, inicio,
			"sessao_id", req.GetSessaoId(),
			"poltronas", len(req.GetPoltronasIds()),
			"chamador", subjectDoChamador(ctx))
		return &pb.RespostaBloqueio{
			Sucesso:  false,
			Mensagem: resultado.Mensagem,
			Motivo:   pb.MotivoFalha_POLTRONAS_INDISPONIVEIS,
		}, nil
	}

	s.metricas.registrar(ctx, "BloquearPoltronas", desfechoConcedido, inicio)
	s.obs.LogOperacao(ctx, "BloquearPoltronas", desfechoConcedido, inicio,
		"sessao_id", req.GetSessaoId(),
		"reserva_id", resultado.Reserva.ID,
		"poltronas", len(resultado.Reserva.Rotulos),
		"chamador", subjectDoChamador(ctx))

	return &pb.RespostaBloqueio{
		Sucesso:   true,
		ReservaId: resultado.Reserva.ID,
		Mensagem:  resultado.Mensagem,
		ExpiraEm:  resultado.Reserva.ExpiraEm.Unix(),
		Motivo:    pb.MotivoFalha_MOTIVO_NAO_INFORMADO,
	}, nil
}

// classificar mapeia o erro para o rótulo de desfecho das métricas.
func classificar(err error) string {
	switch {
	case errors.Is(err, shared.ErrSolicitacaoInvalida),
		errors.Is(err, shared.ErrLimiteExcedido),
		errors.Is(err, shared.ErrPoltronaInexistente),
		errors.Is(err, shared.ErrSessaoNaoProvisionada),
		errors.Is(err, shared.ErrSessaoDesconhecida):
		return desfechoInvalido
	case errors.Is(err, shared.ErrPoltronasIndisponiveis):
		return desfechoIndisponivel
	default:
		return desfechoFalha
	}
}
