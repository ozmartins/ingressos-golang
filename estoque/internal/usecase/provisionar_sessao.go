package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// LayoutPoltrona é uma poltrona descrita no fato sessao.criada.
type LayoutPoltrona struct {
	Fileira string `json:"fileira"`
	Numero  int    `json:"numero"`
	Tipo    string `json:"tipo"`
}

// EventoSessaoCriada é o corpo do fato publicado pelo catálogo.
type EventoSessaoCriada struct {
	Evento     string           `json:"evento"`
	Versao     int              `json:"versao"`
	OcorridoEm string           `json:"ocorrido_em"`
	SessaoID   string           `json:"sessao_id"`
	SalaID     string           `json:"sala_id"`
	Poltronas  []LayoutPoltrona `json:"poltronas"`
}

// LerSessaoCriada valida o fato recebido. Qualquer problema aqui é erro
// definitivo — a mensagem vai para a DLQ, não volta para a fila.
func LerSessaoCriada(corpo []byte) (EventoSessaoCriada, error) {
	var e EventoSessaoCriada
	if err := json.Unmarshal(corpo, &e); err != nil {
		return e, fmt.Errorf("corpo não é JSON válido: %w", err)
	}
	if e.SessaoID == "" {
		return e, fmt.Errorf("campo obrigatório ausente: sessao_id")
	}
	if len(e.Poltronas) == 0 {
		return e, fmt.Errorf("layout sem poltronas")
	}
	return e, nil
}

// ProvisionarSessao cria a matriz de poltronas ao receber sessao.criada.
type ProvisionarSessao struct {
	Poltronas RepositorioPoltronas
	Log       Registrador
}

// Executar valida o layout inteiro antes de gravar e provisiona de forma
// indivisível: ou todas as poltronas são criadas, ou nenhuma (FR-035).
func (uc ProvisionarSessao) Executar(ctx context.Context, fila, messageID string, evento EventoSessaoCriada) (ResultadoTransicao, error) {
	poltronas, err := montarMatriz(evento)
	if err != nil {
		return TransicaoIgnoradaInexistente, err
	}

	resultado, err := uc.Poltronas.ProvisionarMatriz(ctx, fila, messageID, evento.SessaoID, poltronas)
	if err != nil {
		return resultado, err
	}

	switch resultado {
	case TransicaoAplicada:
		uc.Log.Info("matriz de poltronas provisionada",
			"sessao_id", evento.SessaoID, "poltronas", len(poltronas))
	case TransicaoIgnoradaDuplicata:
		// Reanúncio do mesmo fato: nenhuma poltrona duplicada, nenhum estado
		// corrente reiniciado (FR-034).
		uc.Log.Info("sessão já provisionada; reanúncio ignorado", "sessao_id", evento.SessaoID)
	}
	return resultado, nil
}

// montarMatriz valida o layout completo e devolve as poltronas em ordem
// determinística. Fileira e número repetidos invalidam o fato inteiro.
func montarMatriz(evento EventoSessaoCriada) ([]poltrona.Poltrona, error) {
	vistos := make(map[string]struct{}, len(evento.Poltronas))
	matriz := make([]poltrona.Poltrona, 0, len(evento.Poltronas))

	for _, l := range evento.Poltronas {
		p, err := poltrona.Nova(evento.SessaoID, l.Fileira, l.Numero, poltrona.Tipo(l.Tipo))
		if err != nil {
			return nil, fmt.Errorf("layout inválido: %w", err) // já embrulha ErrSolicitacaoInvalida
		}
		if _, repetido := vistos[p.Rotulo]; repetido {
			return nil, fmt.Errorf("%w: layout com poltrona %s repetida na sessão %s", shared.ErrSolicitacaoInvalida, p.Rotulo, evento.SessaoID)
		}
		vistos[p.Rotulo] = struct{}{}
		matriz = append(matriz, p)
	}

	sort.Slice(matriz, func(i, j int) bool {
		if matriz[i].Fileira != matriz[j].Fileira {
			return matriz[i].Fileira < matriz[j].Fileira
		}
		return matriz[i].Numero < matriz[j].Numero
	})
	return matriz, nil
}
