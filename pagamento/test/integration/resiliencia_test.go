//go:build integration

package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

var errSempreFora = errors.New("dependência fora do ar")

type adquirenteQueVolta struct {
	mu              sync.Mutex
	falhasRestantes int
	chamadas        int
}

func (a *adquirenteQueVolta) Cobrar(context.Context, usecase.Cobranca) (usecase.ResultadoCobranca, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chamadas++
	if a.falhasRestantes > 0 {
		a.falhasRestantes--
		return usecase.ResultadoCobranca{}, errSempreFora
	}
	return usecase.ResultadoCobranca{Desfecho: usecase.Aprovada, Codigo: "gw-depois-da-volta"}, nil
}

func TestFalhaTransitoriaDevolveAFilaEDepoisCompleta(t *testing.T) {
	a := subirAmbiente(t)
	adq := &adquirenteQueVolta{falhasRestantes: 2}
	_, parar := a.consumidorDe(t, adq, 4)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "84.00", "PIX", 30*time.Minute))

	tr := a.esperarStatus(t, reserva, transacao.Pago, 60*time.Second)
	if tr.CodigoTransacaoGateway != "gw-depois-da-volta" {
		t.Fatalf("esperava a cobrança da tentativa bem-sucedida, veio %q", tr.CodigoTransacaoGateway)
	}
	if adq.chamadas < 3 {
		t.Fatalf("esperava ao menos 3 tentativas (2 falhas + 1 sucesso), veio %d", adq.chamadas)
	}

	fatos := a.fatosEspiados(t)
	if len(fatos) != 1 {
		t.Fatalf("esperava exatamente um anúncio, veio %d: %v", len(fatos), fatos)
	}
}

func TestNadaEAnunciadoEnquantoAInfraEstaFora(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{})
	adq.erro = errSempreFora
	_, parar := a.consumidorDe(t, adq, 2)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "84.00", "PIX", 30*time.Minute))

	time.Sleep(6 * time.Second)
	if fatos := a.fatosEspiados(t); len(fatos) != 0 {
		t.Fatalf("nenhum anúncio pode sair sem desfecho, veio %v", fatos)
	}
}
