package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// EventoReservaCriada é o payload publicado em reserva.criada. A forma é o
// contrato (contracts/eventos.md) e só muda com versão nova.
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

// RoutingKeyReservaCriada é a chave de roteamento do fato publicado.
const RoutingKeyReservaCriada = "reserva.criada"

// BloquearPoltronas é o caso de uso central do serviço.
type BloquearPoltronas struct {
	Reservas RepositorioReservas
	Prazo    IndiceDePrazo
	Relogio  Relogio
	Log      Registrador

	// TTL é o prazo da reserva (padrão 10 min, configurável).
	TTL time.Duration
	// Limite é o máximo de poltronas por bloqueio (padrão 10, configurável).
	Limite int
	// TraceContextDe extrai o contexto de rastreamento corrente para que o
	// publicador o reinjete depois. Opcional: sem ele o fato vai sem contexto.
	TraceContextDe func(context.Context) map[string]string
}

// ResultadoBloqueio é o que o adaptador precisa para montar a resposta.
type ResultadoBloqueio struct {
	Concedido bool
	Reserva   reserva.Reserva
	Mensagem  string
}

// Executar valida a solicitação, delega o bloqueio atômico e devolve o desfecho.
//
// A ordem importa: validar antes de abrir transação significa que solicitação
// inválida não trava linha nenhuma nem gasta conexão (FR-003, FR-004).
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
		// Indisponibilidade de poltrona é desfecho de negócio, não erro: vira
		// sucesso=false na resposta (contracts/erros.md).
		if errors.Is(err, shared.ErrPoltronasIndisponiveis) {
			return ResultadoBloqueio{Concedido: false, Mensagem: "uma ou mais poltronas já estão reservadas ou ocupadas"}, nil
		}
		return ResultadoBloqueio{}, err
	}

	// O índice de prazo é conveniência, não correção: falha aqui é registrada e
	// o bloqueio segue válido — a varredura cobre a expiração (D2/D4).
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
