package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type LayoutPoltrona struct {
	Fileira string `json:"fileira"`
	Numero  int    `json:"numero"`
	Tipo    string `json:"tipo"`
}

type EventoSessaoCriada struct {
	Evento     string           `json:"evento"`
	Versao     int              `json:"versao"`
	OcorridoEm string           `json:"ocorrido_em"`
	SessaoID   string           `json:"sessao_id"`
	SalaID     string           `json:"sala_id"`
	Poltronas  []LayoutPoltrona `json:"poltronas"`
}

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

type ProvisionarSessao struct {
	Poltronas RepositorioPoltronas
	Log       Registrador
}

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
		uc.Log.Info("sessão já provisionada; reanúncio ignorado", "sessao_id", evento.SessaoID)
	}
	return resultado, nil
}

func montarMatriz(evento EventoSessaoCriada) ([]poltrona.Poltrona, error) {
	vistos := make(map[string]struct{}, len(evento.Poltronas))
	matriz := make([]poltrona.Poltrona, 0, len(evento.Poltronas))

	for _, l := range evento.Poltronas {
		p, err := poltrona.Nova(evento.SessaoID, l.Fileira, l.Numero, poltrona.Tipo(l.Tipo))
		if err != nil {
			return nil, fmt.Errorf("layout inválido: %w", err)
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
