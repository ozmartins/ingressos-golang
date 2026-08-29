package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// Relogio permite fixar o tempo nos testes sem esperar o mundo real.
type Relogio func() time.Time

type ReservarPoltronas struct {
	Sessoes SessaoRepository
	Estoque EstoqueGateway
	Agora   Relogio
}

// Executar traduz a intenção de compra em um bloqueio no Servico-Estoque.
//
// A ordem importa e é deliberada: validar a entrada, buscar a sessão, checar se
// ela ainda aceita reserva e só então sair para a rede. Cada recusa que acontece
// antes da chamada é uma ida ao estoque que não foi feita — e, no caso de
// credencial inválida, é também uma solicitação anônima que nunca alcança um
// serviço interno.
func (uc ReservarPoltronas) Executar(ctx context.Context, s reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error) {
	if err := s.Validar(); err != nil {
		return reserva.ResultadoReserva{}, err
	}

	sessao, err := uc.Sessoes.BuscarPorID(ctx, s.SessaoID)
	if err != nil {
		return reserva.ResultadoReserva{}, err
	}

	agora := time.Now
	if uc.Agora != nil {
		agora = uc.Agora
	}
	if !sessao.AceitaReserva(agora()) {
		return reserva.ResultadoReserva{}, fmt.Errorf("%w: sessão %s está %s", shared.ErrSessaoNaoReservavel, sessao.ID, sessao.Status)
	}

	resultado, err := uc.Estoque.BloquearPoltronas(ctx, s)
	uc.auditar(ctx, s, resultado, err)
	if err != nil {
		return reserva.ResultadoReserva{}, err
	}
	if err := resultado.ValidarIntegridade(); err != nil {
		return reserva.ResultadoReserva{}, err
	}
	return resultado, nil
}

// auditar registra quem pediu o quê e o que aconteceu (FR-033).
//
// Registra a identidade já extraída, nunca a credencial: o token não entra em
// log em nenhuma forma, nem truncado.
func (uc ReservarPoltronas) auditar(ctx context.Context, s reserva.SolicitacaoReserva, r reserva.ResultadoReserva, err error) {
	attrs := []any{
		slog.String("evento", "solicitacao_reserva"),
		slog.String("usuario_id", s.UsuarioID),
		slog.String("sessao_id", s.SessaoID),
		slog.Any("poltronas", s.PoltronasIDs),
	}
	switch {
	case err == nil:
		attrs = append(attrs, slog.String("desfecho", "confirmada"), slog.String("reserva_id", r.ReservaID))
	case errors.Is(err, shared.ErrPoltronasIndisponiveis):
		attrs = append(attrs, slog.String("desfecho", "indisponivel"))
	case errors.Is(err, shared.ErrEstoqueIndisponivel):
		attrs = append(attrs, slog.String("desfecho", "estoque_indisponivel"))
	default:
		attrs = append(attrs, slog.String("desfecho", "erro"))
	}
	slog.InfoContext(ctx, "auditoria de reserva", attrs...)
}
