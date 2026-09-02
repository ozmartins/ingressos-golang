//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

func TestReentregaPublicaResultadoPendenteSemRecobrar(t *testing.T) {
	a := subirAmbiente(t)
	ctx := context.Background()

	reserva := uuid.NewString()
	usuario := uuid.NewString()
	tr := transacao.Nova(uuid.NewString(), reserva, usuario, "84.00", transacao.PIX, time.Now().UTC())
	criada, _, err := a.Repo.CriarSeAusente(ctx, tr)
	if err != nil || !criada {
		t.Fatalf("preparação falhou: criada=%v err=%v", criada, err)
	}
	if err := tr.Aprovar("gw-antes-da-queda", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := a.Repo.Finalizar(ctx, tr); err != nil {
		t.Fatal(err)
	}

	antes, _ := a.Repo.BuscarPorReserva(ctx, reserva)
	if antes.ResultadoAnunciado {
		t.Fatal("preparação inválida: o resultado não podia estar anunciado")
	}

	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Aprovada, Codigo: "nao-deve-acontecer"})
	_, parar := a.consumidorDe(t, adq, 4)
	defer parar()

	msg := intencao(reserva, "84.00", "PIX", 10*time.Minute)
	msg["usuario_id"] = usuario
	a.publicarIntencao(t, msg)

	limite := time.Now().Add(30 * time.Second)
	for time.Now().Before(limite) {
		atual, err := a.Repo.BuscarPorReserva(ctx, reserva)
		if err == nil && atual.ResultadoAnunciado {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	depois, err := a.Repo.BuscarPorReserva(ctx, reserva)
	if err != nil {
		t.Fatal(err)
	}
	if !depois.ResultadoAnunciado {
		t.Fatal("SC-003 violado: o resultado gravado nunca foi anunciado")
	}
	if depois.CodigoTransacaoGateway != "gw-antes-da-queda" {
		t.Fatalf("a transação foi sobrescrita: %q", depois.CodigoTransacaoGateway)
	}
	if n := adq.total(); n != 0 {
		t.Fatalf("reentrega NÃO pode cobrar de novo, houve %d cobranças", n)
	}

	fatos := a.fatosEspiados(t)
	if len(fatos) != 1 || fatos[0]["reserva_id"] != reserva {
		t.Fatalf("esperava um anúncio da transação gravada, veio %v", fatos)
	}
}
