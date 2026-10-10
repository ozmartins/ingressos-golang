package usecase

import (
	"context"
	"log/slog"
)

type VarrerCobrancas struct {
	Repo     Repositorio
	Cobranca ProcessarPagamento
	Relogio  Relogio
	Log      *slog.Logger

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
		if _, err := uc.Cobranca.anunciar(ctx, t); err != nil {
			uc.Log.Warn("falha ao republicar desfecho; será retomado",
				"transacao_id", t.ID, "erro", err)
		}
	}
	return nil
}

func (uc VarrerCobrancas) desistirDasVencidas(ctx context.Context, lote int) error {
	canceladas, err := uc.Repo.CancelarEsperasVencidas(ctx, uc.Relogio.Agora(), lote)
	if err != nil {
		return err
	}
	for _, t := range canceladas {
		uc.Log.Info("espera vencida sem escolha de forma; transação cancelada",
			"reserva_id", t.ReservaID, "transacao_id", t.ID)
		if _, err := uc.Cobranca.anunciar(ctx, t); err != nil {
			uc.Log.Warn("falha ao anunciar cancelamento; será retomado",
				"transacao_id", t.ID, "erro", err)
		}
	}
	return nil
}

func (uc VarrerCobrancas) cobrarAsEscolhidas(ctx context.Context, lote int) error {
	pendentes, err := uc.Repo.AguardandoCobranca(ctx, lote)
	if err != nil {
		return err
	}
	for _, t := range pendentes {
		desfecho, err := uc.Cobranca.Cobrar(ctx, t)
		if err != nil {
			uc.Log.Warn("cobrança não concluída; será retomada",
				"transacao_id", t.ID, "desfecho", desfecho, "erro", err)
			continue
		}
		uc.Log.Info("cobrança concluída", "transacao_id", t.ID, "reserva_id", t.ReservaID)
	}
	return nil
}
