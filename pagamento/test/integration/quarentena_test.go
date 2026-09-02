//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/adapter/adquirente/simulado"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

func TestAnuncioInvalidoVaiParaFilaMortaSemCriarTransacao(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Aprovada})
	_, parar := a.consumidorDe(t, adq, 4)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, map[string]any{
		"evento": "RESERVA_CRIADA", "versao": 1,
		"ocorrido_em":   time.Now().UTC().Format(time.RFC3339),
		"reserva_id":    reserva,
		"sessao_id":     uuid.NewString(),
		"usuario_id":    uuid.NewString(),
		"poltronas_ids": []string{"A1"},
		"expira_em":     time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339),
	})

	esperarFila(t, a, filaDLQ, 1, 30*time.Second)

	if adq.total() != 0 {
		t.Fatal("anúncio inválido não pode gerar cobrança")
	}
	if _, err := a.Repo.BuscarPorReserva(context.Background(), reserva); err == nil {
		t.Fatal("anúncio inválido não pode criar transação")
	}
	if fatos := a.fatosEspiados(t); len(fatos) != 0 {
		t.Fatalf("anúncio inválido não pode ser anunciado, veio %v", fatos)
	}
}

func TestDesfechoIndeterminadoParaEstadoEQuarentenaJuntos(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Indeterminada})
	_, parar := a.consumidorDe(t, adq, 4)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "99.99", "CARTAO_CREDITO", 10*time.Minute))

	tr := a.esperarStatus(t, reserva, transacao.PendenteVerificacao, 30*time.Second)
	if tr.ResultadoAnunciado {
		t.Fatal("PENDENTE_VERIFICACAO nunca é marcada como anunciada")
	}

	esperarFila(t, a, filaDLQ, 1, 30*time.Second)

	if fatos := a.fatosEspiados(t); len(fatos) != 0 {
		t.Fatalf("SC-009 violado: o estado indeterminado não pode anunciar nada, veio %v", fatos)
	}
	if n := adq.total(); n != 1 {
		t.Fatalf("esperava exatamente uma tentativa, veio %d", n)
	}
}

func TestLimiteDeEntregasEncaminhaParaFilaMorta(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{})
	adq.erro = errSempreFora
	_, parar := a.consumidorDe(t, adq, 2)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "84.00", "PIX", 30*time.Minute))

	esperarFila(t, a, filaDLQ, 1, 60*time.Second)

	if n := adq.total(); n > 3 {
		t.Fatalf("FR-021 violado: esperava no máximo 3 tentativas, veio %d", n)
	}
	if n := adq.total(); n < 2 {
		t.Fatalf("esperava mais de uma tentativa antes da quarentena, veio %d", n)
	}
	if fatos := a.fatosEspiados(t); len(fatos) != 0 {
		t.Fatalf("nada pode ser anunciado sem desfecho, veio %v", fatos)
	}
	tr, err := a.Repo.BuscarPorReserva(context.Background(), reserva)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Status != transacao.Processando {
		t.Fatalf("sem desfecho, a transação segue PROCESSANDO; veio %s", tr.Status)
	}
}

func TestMensagemNaQuarentenaPreservaOCorpo(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Aprovada})
	_, parar := a.consumidorDe(t, adq, 2)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "84.00", "BOLETO", 10*time.Minute))
	esperarFila(t, a, filaDLQ, 1, 30*time.Second)

	canal, err := a.Conexao.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer canal.Close()
	msg, ok, err := canal.Get(filaDLQ, true)
	if err != nil || !ok {
		t.Fatalf("nada na quarentena: ok=%v err=%v", ok, err)
	}
	var m map[string]any
	if err := json.Unmarshal(msg.Body, &m); err != nil {
		t.Fatal(err)
	}
	if m["reserva_id"] != reserva {
		t.Fatalf("corpo alterado na quarentena: %v", m)
	}
	if _, tem := msg.Headers["x-death"]; !tem {
		t.Log("aviso: x-death ausente — a origem do descarte não fica registrada no cabeçalho")
	}
	_ = amqp091.Persistent
}

func TestPrazoDoAdquirenteRealLevaAQuarentena(t *testing.T) {
	a := subirAmbiente(t)
	_, parar := a.consumidorDe(t, simulado.Adquirente{Demora: 30 * time.Second}, 4)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "99.99", "CARTAO_CREDITO", 30*time.Minute))

	tr := a.esperarStatus(t, reserva, transacao.PendenteVerificacao, 60*time.Second)
	if !tr.CobrancaEmitida {
		t.Fatal("prazo estourado não pode liberar o direito de cobrar (FR-008)")
	}
	if tr.ResultadoAnunciado {
		t.Fatal("PENDENTE_VERIFICACAO nunca é anunciada")
	}
	esperarFila(t, a, filaDLQ, 1, 30*time.Second)
	if fatos := a.fatosEspiados(t); len(fatos) != 0 {
		t.Fatalf("nada pode ser anunciado, veio %v", fatos)
	}
}

func esperarFila(t *testing.T, a *ambiente, fila string, minimo int, prazo time.Duration) {
	t.Helper()
	limite := time.Now().Add(prazo)
	var ultimo int
	for time.Now().Before(limite) {
		ultimo = a.contarFila(t, fila)
		if ultimo >= minimo {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("fila %s não atingiu %d mensagens em %s (última contagem: %d)", fila, minimo, prazo, ultimo)
}
