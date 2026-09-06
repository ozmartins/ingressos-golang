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

// O desfecho indeterminado não anuncia nada e para em PENDENTE_VERIFICACAO,
// para inspeção humana.
//
// A mensagem NÃO vai mais para a fila morta: a cobrança deixou de acontecer no
// consumo do anúncio, então não há entrega a descartar. O anúncio da reserva foi
// processado com sucesso — o que ficou indeterminado é a cobrança, e ela é
// sinalizada pelo estado da transação, não pelo destino da mensagem.
func TestDesfechoIndeterminadoParaEmVerificacaoSemAnunciar(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Indeterminada})
	_, parar := a.consumidorDe(t, adq, 4)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "99.99", 10*time.Minute))

	tr := a.esperarStatus(t, reserva, transacao.PendenteVerificacao, 30*time.Second)
	if tr.ResultadoAnunciado {
		t.Fatal("PENDENTE_VERIFICACAO nunca é marcada como anunciada")
	}
	if n := a.contarFila(t, filaDLQ); n != 0 {
		t.Fatalf("o anúncio da reserva foi processado; nada deveria ir para a fila morta, veio %d", n)
	}

	if fatos := a.fatosEspiados(t); len(fatos) != 0 {
		t.Fatalf("SC-009 violado: o estado indeterminado não pode anunciar nada, veio %v", fatos)
	}
	if n := adq.total(); n != 1 {
		t.Fatalf("esperava exatamente uma tentativa, veio %d", n)
	}
}

// Adquirente sempre fora: a cobrança é retentada indefinidamente pela varredura,
// e a transação fica em PROCESSANDO sem anunciar nada. Não há desfecho a
// inventar, e não há mensagem a descartar — o anúncio da reserva já foi
// processado.
//
// Antes desta mudança este caso terminava na fila morta, porque a cobrança
// acontecia no consumo e o limite de entregas a descartava. Com a cobrança fora
// do consumo, o que persiste é a linha no banco, e é ela que a varredura retoma.
func TestAdquirenteSempreForaMantemProcessandoSemAnunciar(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{})
	adq.erro = errSempreFora
	_, parar := a.consumidorDe(t, adq, 2)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "84.00", 30*time.Minute))

	tr := a.esperarStatus(t, reserva, transacao.Processando, 30*time.Second)
	if tr.Status != transacao.Processando {
		t.Fatalf("sem desfecho, a transação segue PROCESSANDO; veio %s", tr.Status)
	}

	// A retomada precisa acontecer de verdade: uma só tentativa significaria que
	// a varredura desistiu.
	esperarTentativas(t, adq, 2, 30*time.Second)

	if fatos := a.fatosEspiados(t); len(fatos) != 0 {
		t.Fatalf("nada pode ser anunciado sem desfecho, veio %v", fatos)
	}
	if n := a.contarFila(t, filaDLQ); n != 0 {
		t.Fatalf("o anúncio da reserva foi processado; nada deveria ir para a fila morta, veio %d", n)
	}
}

func esperarTentativas(t *testing.T, adq *adquirenteControlado, minimo int, prazo time.Duration) {
	t.Helper()
	limite := time.Now().Add(prazo)
	for time.Now().Before(limite) {
		if adq.total() >= minimo {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("esperava ao menos %d tentativas de cobrança em %s, veio %d", minimo, prazo, adq.total())
}

func TestMensagemNaQuarentenaPreservaOCorpo(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Aprovada})
	_, parar := a.consumidorDe(t, adq, 2)
	defer parar()

	reserva := uuid.NewString()
	// Valor inválido, e não forma inválida: a forma deixou de viajar no fato, e
	// quem a recusa agora é o endpoint de escolha. Ver os testes de
	// `internal/adapter/http`.
	a.publicarIntencao(t, intencao(reserva, "-1.00", 10*time.Minute))
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

// O prazo do adquirente estourado deixa a cobrança indeterminada: o direito de
// cobrar não é liberado (FR-008) e nada é anunciado.
func TestPrazoDoAdquirenteRealParaEmVerificacao(t *testing.T) {
	a := subirAmbiente(t)
	_, parar := a.consumidorDe(t, simulado.Adquirente{Demora: 30 * time.Second}, 4)
	defer parar()

	reserva := uuid.NewString()
	a.publicarIntencao(t, intencao(reserva, "99.99", 30*time.Minute))

	tr := a.esperarStatus(t, reserva, transacao.PendenteVerificacao, 60*time.Second)
	if !tr.CobrancaEmitida {
		t.Fatal("prazo estourado não pode liberar o direito de cobrar (FR-008)")
	}
	if tr.ResultadoAnunciado {
		t.Fatal("PENDENTE_VERIFICACAO nunca é anunciada")
	}
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
