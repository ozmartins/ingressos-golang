package usecase

import (
	"context"
	"fmt"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// ConsultarMapa devolve o estado corrente das poltronas de uma sessão. Leitura
// pura: não altera estado algum (FR-030).
type ConsultarMapa struct {
	Poltronas RepositorioPoltronas
}

// Executar devolve o mapa da sessão. Sessão sem nenhuma poltrona é desconhecida
// e precisa ser distinguível de uma resposta vazia legítima (FR-031).
func (uc ConsultarMapa) Executar(ctx context.Context, sessaoID string) ([]poltrona.Poltrona, error) {
	if sessaoID == "" {
		return nil, fmt.Errorf("%w: sessão não informada", shared.ErrSolicitacaoInvalida)
	}

	mapa, err := uc.Poltronas.MapaDaSessao(ctx, sessaoID)
	if err != nil {
		return nil, err
	}
	if len(mapa) == 0 {
		return nil, fmt.Errorf("%w: %s", shared.ErrSessaoDesconhecida, sessaoID)
	}
	return mapa, nil
}
