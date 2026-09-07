package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

const (
	RoutingKeySessaoAlterada  = "sessao.alterada"
	RoutingKeySessaoCancelada = "sessao.cancelada"
)

// A sessão mudou em algo que não é a sala — horário, idioma ou preço. A sala não
// pode mudar, e é por isso que este fato nunca invalida a matriz de poltronas de
// quem a provisionou.
//
// `sala_id` viaja de todo modo: quem consome não deveria precisar guardar de qual
// sala era para concluir que não mudou.
type EventoSessaoAlterada struct {
	Evento         string `json:"evento"`
	Versao         int    `json:"versao"`
	OcorridoEm     string `json:"ocorrido_em"`
	SessaoID       string `json:"sessao_id"`
	SalaID         string `json:"sala_id"`
	DataHoraInicio string `json:"data_hora_inicio"`
	Idioma         string `json:"idioma"`
	PrecoBase      string `json:"preco_base"`
}

// A sessão saiu da grade. Quem tem estado preso a ela precisa soltá-lo: no
// estoque, as reservas pendentes das poltronas dela.
type EventoSessaoCancelada struct {
	Evento     string `json:"evento"`
	Versao     int    `json:"versao"`
	OcorridoEm string `json:"ocorrido_em"`
	SessaoID   string `json:"sessao_id"`
}

func (uc AtualizarSessao) anunciarAlteracao(ctx context.Context, sessao catalogo.Sessao) (FatoPendente, error) {
	payload, err := json.Marshal(EventoSessaoAlterada{
		Evento:         "SESSAO_ALTERADA",
		Versao:         1,
		OcorridoEm:     agoraOu(uc.Agora)().UTC().Format(time.RFC3339),
		SessaoID:       sessao.ID,
		SalaID:         sessao.SalaID,
		DataHoraInicio: sessao.DataHoraInicio.UTC().Format(time.RFC3339),
		Idioma:         string(sessao.Idioma),
		PrecoBase:      sessao.PrecoBase.String(),
	})
	if err != nil {
		return FatoPendente{}, fmt.Errorf("montando o anúncio da alteração: %w", err)
	}

	// Identificador próprio por ocorrência, e não o da sessão: a mesma sessão
	// pode ser alterada muitas vezes, e a caixa de saída tem `message_id` único
	// — a segunda alteração colidiria com a primeira e seria descartada em
	// silêncio.
	return FatoPendente{
		MessageID:    uc.GerarID(),
		RoutingKey:   RoutingKeySessaoAlterada,
		Payload:      payload,
		TraceContext: traceDe(uc.TraceContextDe, ctx),
	}, nil
}

func (uc RemoverSessao) anunciarCancelamento(ctx context.Context, sessaoID string) (FatoPendente, error) {
	payload, err := json.Marshal(EventoSessaoCancelada{
		Evento:     "SESSAO_CANCELADA",
		Versao:     1,
		OcorridoEm: agoraOu(uc.Agora)().UTC().Format(time.RFC3339),
		SessaoID:   sessaoID,
	})
	if err != nil {
		return FatoPendente{}, fmt.Errorf("montando o anúncio do cancelamento: %w", err)
	}

	// O cancelamento é terminal e acontece uma vez, então o identificador da
	// sessão com o sufixo do fato é estável — e a idempotência de quem consome
	// vem de graça.
	return FatoPendente{
		MessageID:    sessaoID + ":cancelada",
		RoutingKey:   RoutingKeySessaoCancelada,
		Payload:      payload,
		TraceContext: traceDe(uc.TraceContextDe, ctx),
	}, nil
}

func agoraOu(f func() time.Time) func() time.Time {
	if f != nil {
		return f
	}
	return time.Now
}

func traceDe(f func(context.Context) map[string]string, ctx context.Context) map[string]string {
	if f == nil {
		return nil
	}
	return f(ctx)
}
