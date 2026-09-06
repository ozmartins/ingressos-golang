package usecase

import (
	"context"
	"log/slog"
)

// VarrerCobrancas roda fora do caminho da requisição e faz as duas coisas que
// dependem da passagem do tempo, não de uma mensagem: cobrar quem já escolheu a
// forma e desistir de quem nunca escolheu.
//
// A cobrança mora aqui, e não no endpoint, porque quem paga escolheu a forma e
// não precisa esperar o adquirente responder para saber que o pedido foi aceito.
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

// O desfecho está gravado, mas o anúncio não saiu — o processo caiu entre as
// duas coisas, ou o broker recusou. Republicar é o que torna a entrega ao menos
// uma vez de verdade: sem isso, quem espera o resultado nunca o receberia.
//
// Antes desta separação era o consumo do anúncio da reserva que retomava isso,
// ao reencontrar uma transação já resolvida. Como o consumo deixou de decidir
// cobrança, a retomada passou a ser trabalho da varredura.
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

// A reserva venceu sem ninguém escolher como pagar: as poltronas já foram
// liberadas do outro lado, e a transação precisa parar num estado final para
// que o desfecho seja anunciado a quem espera.
func (uc VarrerCobrancas) desistirDasVencidas(ctx context.Context, lote int) error {
	canceladas, err := uc.Repo.CancelarEsperasVencidas(ctx, uc.Relogio.Agora(), lote)
	if err != nil {
		return err
	}
	for _, t := range canceladas {
		uc.Log.Info("espera vencida sem escolha de forma; transação cancelada",
			"reserva_id", t.ReservaID, "transacao_id", t.ID)
		if _, err := uc.Cobranca.anunciar(ctx, t); err != nil {
			// O anúncio pendente é retomado no tique seguinte: o estado já é
			// final e durável, e é isso que a retomada exige.
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
