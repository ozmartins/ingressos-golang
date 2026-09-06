package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

const RoutingKeySessaoCriada = "sessao.criada"

// O corpo do fato `sessao.criada`, como está em
// specs/001-catalogo-sessoes-reserva/contracts/eventos.md. A planta da sala vai
// expandida assento a assento: quem consome materializa a matriz da sessão a
// partir dela, e não conhece o conceito de fileira.
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
	Sessoes SessaoRepository
	Filmes  FilmeRepository
	Salas   SalaRepository
	GerarID func() string
	Agora   func() time.Time
	// Captura o contexto de rastreamento da requisição para viajar com o fato: o
	// publicador roda fora dela, e sem isso o span do consumidor nasceria órfão.
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

// A planta anunciada é a que a sala tinha agora: redesenhá-la depois não reemite
// o fato nem muda o que já foi anunciado.
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

	// O `message_id` é o da sessão: é por ele que quem consome descarta a
	// repetição, e a entrega é ao menos uma vez.
	return FatoPendente{
		MessageID:    sessao.ID,
		RoutingKey:   RoutingKeySessaoCriada,
		Payload:      payload,
		TraceContext: traceCtx,
	}, nil
}

// O banco garante apenas que o filme e a sala existem. O que ele não sabe é que
// uma sala projeta um filme de cada vez: a janela da sessão vai do início até o
// fim do filme, e não pode alcançar outra sessão viva na mesma sala.
//
// Devolve a sala porque quem cria a sessão também precisa da planta dela.
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

	// Uma sessão cancelada ou já finalizada não ocupa a sala.
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
