//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

func TestCobrancaPontaAPonta(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Aprovada, Codigo: "gw-integr"})
	_, parar := a.consumidorDe(t, adq, 4)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "84.00", "PIX", 10*time.Minute))

	tr := a.esperarStatus(t, reserva, transacao.Pago, 30*time.Second)
	if tr.CodigoTransacaoGateway != "gw-integr" || tr.PagoEm == nil {
		t.Fatalf("transação mal gravada: %+v", tr)
	}
	if !tr.ResultadoAnunciado {
		t.Fatal("resultado devia estar marcado como anunciado (FR-014)")
	}
	if tr.AtualizadoEm.Equal(tr.CriadoEm) {
		t.Fatal("atualizado_em devia avançar a cada escrita (FR-023)")
	}

	fatos := a.fatosEspiados(t)
	if len(fatos) != 1 {
		t.Fatalf("esperava 1 anúncio, veio %d: %v", len(fatos), fatos)
	}
	f := fatos[0]
	if f["__routing_key"] != "pagamento.sucesso" || f["evento"] != "PAGAMENTO_SUCESSO" {
		t.Fatalf("fato fora do contrato: %v", f)
	}
	for _, campo := range []string{"transacao_id", "reserva_id", "usuario_id", "valor_total", "pago_em", "versao", "ocorrido_em"} {
		if _, ok := f[campo]; !ok {
			t.Fatalf("campo %q ausente (contracts/eventos.md §2)", campo)
		}
	}
	if f["reserva_id"] != reserva {
		t.Fatalf("reserva_id errado: %v", f["reserva_id"])
	}
}

func TestReservaExpiradaNaoCobraPontaAPonta(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Aprovada})
	_, parar := a.consumidorDe(t, adq, 4)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "84.00", "PIX", -time.Minute))

	tr := a.esperarStatus(t, reserva, transacao.Cancelado, 30*time.Second)
	if tr.MotivoFalha != transacao.MotivoReservaExpirada {
		t.Fatalf("motivo errado: %q", tr.MotivoFalha)
	}
	if adq.total() != 0 {
		t.Fatalf("reserva expirada não pode ser cobrada, houve %d cobranças", adq.total())
	}

	fatos := a.fatosEspiados(t)
	if len(fatos) != 1 || fatos[0]["motivo"] != "RESERVA_EXPIRADA" {
		t.Fatalf("esperava pagamento.falhou com RESERVA_EXPIRADA, veio %v", fatos)
	}
}
