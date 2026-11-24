package usecase

import (
	"context"
)

type VarrerCobrancas struct {
	Repo     Repositorio
	Cobranca ProcessarPagamento
	Relogio  Relogio

	Lote int

	// ComRequestID devolve ctx carregando o request id gravado na transação
	// (nil = não restaura). A varredura roda fora de qualquer requisição.
	ComRequestID func(ctx context.Context, id string) context.Context
}

func (uc VarrerCobrancas) contextoDe(ctx context.Context, requestID string) context.Context {
	if uc.ComRequestID == nil || requestID == "" {
		return ctx
	}
	return uc.ComRequestID(ctx, requestID)
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
		_, _ = uc.Cobranca.anunciar(uc.contextoDe(ctx, t.RequestID), t)
	}
	return nil
}

func (uc VarrerCobrancas) desistirDasVencidas(ctx context.Context, lote int) error {
	canceladas, err := uc.Repo.CancelarEsperasVencidas(ctx, uc.Relogio.Agora(), lote)
	if err != nil {
		return err
	}
	for _, t := range canceladas {
		_, _ = uc.Cobranca.anunciar(uc.contextoDe(ctx, t.RequestID), t)
	}
	return nil
}

func (uc VarrerCobrancas) cobrarAsEscolhidas(ctx context.Context, lote int) error {
	pendentes, err := uc.Repo.AguardandoCobranca(ctx, lote)
	if err != nil {
		return err
	}
	for _, t := range pendentes {
		_, _ = uc.Cobranca.Cobrar(uc.contextoDe(ctx, t.RequestID), t)
	}
	return nil
}
