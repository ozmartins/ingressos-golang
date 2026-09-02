package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type EventoReservaCriada struct {
	Evento       string   `json:"evento"`
	Versao       int      `json:"versao"`
	OcorridoEm   string   `json:"ocorrido_em"`
	ReservaID    string   `json:"reserva_id"`
	SessaoID     string   `json:"sessao_id"`
	UsuarioID    string   `json:"usuario_id"`
	PoltronasIDs []string `json:"poltronas_ids"`
	ExpiraEm     string   `json:"expira_em"`
}

const RoutingKeyReservaCriada = "reserva.criada"

type BloquearPoltronas struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
	Log      Registrador

	TTL            time.Duration
	Limite         int
	TraceContextDe func(context.Context) map[string]string
}

type ResultadoBloqueio struct {
	Concedido bool
	Reserva   reserva.Reserva
	Mensagem  string
}

func (uc BloquearPoltronas) Executar(ctx context.Context, sessaoID, usuarioID string, rotulos []string) (ResultadoBloqueio, error) {
	sol, err := reserva.NovaSolicitacao(sessaoID, usuarioID, rotulos, uc.Limite)
	if err != nil {
		return ResultadoBloqueio{}, err
	}

	agora := uc.Relogio.Agora()
	r := reserva.Nova(sol.SessaoID, sol.UsuarioID, sol.Rotulos, agora, uc.TTL)

	fato, err := uc.montarFato(ctx, r)
	if err != nil {
		return ResultadoBloqueio{}, err
	}

	if err := uc.Reservas.Conceder(ctx, sol, r, fato); err != nil {
		if errors.Is(err, shared.ErrPoltronasIndisponiveis) {
			return ResultadoBloqueio{Concedido: false, Mensagem: "uma ou mais poltronas já estão reservadas ou ocupadas"}, nil
		}
		return ResultadoBloqueio{}, err
	}

	if uc.Prazo != nil {
		if err := uc.Prazo.Marcar(ctx, r.ID, r.ExpiraEm); err != nil {
			uc.Log.Warn("índice de prazo indisponível; expiração fica com a varredura",
				"reserva_id", r.ID, "erro", err.Error())
		}
	}

	return ResultadoBloqueio{Concedido: true, Reserva: r, Mensagem: "poltronas bloqueadas"}, nil
}

func (uc BloquearPoltronas) montarFato(ctx context.Context, r reserva.Reserva) (FatoPendente, error) {
	payload, err := json.Marshal(EventoReservaCriada{
		Evento:       "RESERVA_CRIADA",
		Versao:       1,
		OcorridoEm:   r.CriadoEm.Format(time.RFC3339),
		ReservaID:    r.ID,
		SessaoID:     r.SessaoID,
		UsuarioID:    r.UsuarioID,
		PoltronasIDs: r.Rotulos,
		ExpiraEm:     r.ExpiraEm.Format(time.RFC3339),
	})
	if err != nil {
		return FatoPendente{}, err
	}

	var traceCtx map[string]string
	if uc.TraceContextDe != nil {
		traceCtx = uc.TraceContextDe(ctx)
	}

	return FatoPendente{
		MessageID:    r.ID,
		RoutingKey:   RoutingKeyReservaCriada,
		Payload:      payload,
		TraceContext: traceCtx,
	}, nil
}
