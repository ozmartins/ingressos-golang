package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

const RoutingKeySessaoCriada = "sessao.criada"

type EventoSessaoCriada struct {
	Evento     string           `json:"evento"`
	Versao     int              `json:"versao"`
	OcorridoEm string           `json:"ocorrido_em"`
	SessaoID   string           `json:"sessao_id"`
	SalaID     string           `json:"sala_id"`
	Poltronas  []PoltronaNoFato `json:"poltronas"`
}

type PoltronaNoFato struct {
	Fileira string `json:"fileira"`
	Numero  int    `json:"numero"`
	Tipo    string `json:"tipo"`
}

type CriarSessao struct {
	Sessoes        SessaoRepository
	Filmes         FilmeRepository
	Salas          SalaRepository
	GerarID        func() string
	Agora          func() time.Time
	TraceContextDe func(context.Context) map[string]string
}

func (uc CriarSessao) Executar(ctx context.Context, dados catalogo.DadosSessao) (catalogo.Sessao, error) {
	sessao, err := catalogo.NovaSessao(uc.GerarID(), dados)
	if err != nil {
		return catalogo.Sessao{}, err
	}
	sala, err := conferirGrade(ctx, uc.Filmes, uc.Salas, uc.Sessoes, sessao, "")
	if err != nil {
		return catalogo.Sessao{}, err
	}

	fato, err := uc.anunciar(ctx, sessao, sala)
	if err != nil {
		return catalogo.Sessao{}, err
	}
	if err := uc.Sessoes.Criar(ctx, sessao, fato); err != nil {
		return catalogo.Sessao{}, err
	}
	return sessao, nil
}

func (uc CriarSessao) anunciar(ctx context.Context, sessao catalogo.Sessao, sala catalogo.Sala) (FatoPendente, error) {
	poltronas := make([]PoltronaNoFato, 0, sala.CapacidadeTotal())
	for _, p := range sala.Layout.Poltronas() {
		poltronas = append(poltronas, PoltronaNoFato{
			Fileira: p.Fileira, Numero: p.Numero, Tipo: string(p.Tipo)})
	}

	agora := time.Now
	if uc.Agora != nil {
		agora = uc.Agora
	}

	payload, err := json.Marshal(EventoSessaoCriada{
		Evento:     "SESSAO_CRIADA",
		Versao:     1,
		OcorridoEm: agora().UTC().Format(time.RFC3339),
		SessaoID:   sessao.ID,
		SalaID:     sessao.SalaID,
		Poltronas:  poltronas,
	})
	if err != nil {
		return FatoPendente{}, fmt.Errorf("montando o anúncio da sessão: %w", err)
	}

	var traceCtx map[string]string
	if uc.TraceContextDe != nil {
		traceCtx = uc.TraceContextDe(ctx)
	}

	return FatoPendente{
		MessageID:    sessao.ID,
		RoutingKey:   RoutingKeySessaoCriada,
		Payload:      payload,
		TraceContext: traceCtx,
	}, nil
}

func conferirGrade(
	ctx context.Context,
	filmes FilmeRepository,
	salas SalaRepository,
	sessoes SessaoRepository,
	sessao catalogo.Sessao,
	excetoID string,
) (catalogo.Sala, error) {
	filme, err := filmes.BuscarPorID(ctx, sessao.FilmeID)
	if err != nil {
		return catalogo.Sala{}, err
	}
	sala, err := salas.BuscarPorID(ctx, sessao.SalaID)
	if err != nil {
		return catalogo.Sala{}, err
	}

	if sessao.Status != catalogo.SessaoAgendada && sessao.Status != catalogo.SessaoEmAndamento {
		return sala, nil
	}

	fim := sessao.FimPrevisto(filme.DuracaoMinutos)
	ocupada, err := sessoes.SalaOcupada(ctx, sessao.SalaID, sessao.DataHoraInicio, fim, excetoID)
	if err != nil {
		return catalogo.Sala{}, err
	}
	if ocupada {
		return catalogo.Sala{}, errConflito("a sala já tem uma sessão entre %s e %s",
			sessao.DataHoraInicio.Format("2006-01-02 15:04"), fim.Format("15:04"))
	}
	return sala, nil
}
