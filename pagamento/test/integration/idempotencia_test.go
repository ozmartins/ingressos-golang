//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

// T027 / SC-002: vinte entregas simultâneas da mesma reserva produzem uma
// transação e uma única cobrança. É a garantia que envolve dinheiro real.
func TestVinteEntregasSimultaneasCobramUmaVez(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Aprovada, Codigo: "gw"})
	_, parar := a.consumidorDe(t, adq, 10)
	defer parar()

	reserva := uuid.NewString()
	msg := intencao(reserva, "84.00", "PIX", 10*time.Minute)
	for i := 0; i < 20; i++ {
		a.publicarIntencao(t, msg)
	}

	a.esperarStatus(t, reserva, transacao.Pago, 60*time.Second)
	// Deixa as reentregas restantes serem drenadas antes de contar.
	time.Sleep(5 * time.Second)

	if n := adq.total(); n != 1 {
		t.Fatalf("SC-002 violado: esperava exatamente 1 cobrança, veio %d", n)
	}

	var linhas int
	if err := a.Pool.QueryRow(t.Context(),
		"SELECT count(*) FROM transacoes_pagamento WHERE reserva_id=$1", reserva).Scan(&linhas); err != nil {
		t.Fatal(err)
	}
	if linhas != 1 {
		t.Fatalf("esperava 1 transação, veio %d", linhas)
	}

	// Anúncios podem ser mais de um (entrega ao menos uma vez, contracts/eventos.md
	// §4), mas todos precisam se referir à mesma transação.
	fatos := a.fatosEspiados(t)
	if len(fatos) == 0 {
		t.Fatal("o resultado precisa ser anunciado ao menos uma vez")
	}
	primeiro := fatos[0]["transacao_id"]
	for _, f := range fatos {
		if f["transacao_id"] != primeiro || f["__routing_key"] != "pagamento.sucesso" {
			t.Fatalf("anúncios divergentes entre entregas: %v", fatos)
		}
	}
}
