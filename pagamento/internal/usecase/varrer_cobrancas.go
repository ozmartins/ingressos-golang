package usecase

import (
	"context"
)

type VarrerCobrancas struct {
	Repo     Repositorio
	Cobranca ProcessarPagamento
	Relogio  Relogio

	Lote int
}

const lotePadraoVarredura = 50

func (uc VarrerCobrancas) Executar(ctx context.Context) error {
	lote := uc.Lote
	if lote <= 0 {
		lote = lotePadraoVarredura
	}

	if err := uc.desistirDasVencidas(ctx, lote); err != nil {
		return err
	}
	if err := uc.cobrarAsEscolhidas(ctx, lote); err != nil {
		return err
	}
	return uc.republicarAnunciosPendentes(ctx, lote)
}

func (uc VarrerCobrancas) republicarAnunciosPendentes(ctx context.Context, lote int) error {
	pendentes, err := uc.Repo.AnunciosPendentes(ctx, lote)
	if err != nil {
		return err
	}
	for _, t := range pendentes {
		_, _ = uc.Cobranca.anunciar(ctx, t)
	}
	return nil
}

func (uc VarrerCobrancas) desistirDasVencidas(ctx context.Context, lote int) error {
	canceladas, err := uc.Repo.CancelarEsperasVencidas(ctx, uc.Relogio.Agora(), lote)
	if err != nil {
		return err
	}
	for _, t := range canceladas {
		_, _ = uc.Cobranca.anunciar(ctx, t)
	}
	return nil
}

func (uc VarrerCobrancas) cobrarAsEscolhidas(ctx context.Context, lote int) error {
	pendentes, err := uc.Repo.AguardandoCobranca(ctx, lote)
	if err != nil {
		return err
	}
	for _, t := range pendentes {
		_, _ = uc.Cobranca.Cobrar(ctx, t)
	}
	return nil
}
