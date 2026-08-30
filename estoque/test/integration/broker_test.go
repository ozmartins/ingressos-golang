//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"

	adaptadoramqp "github.com/oseias/ingressos-golang/estoque/internal/adapter/amqp"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func observabilidade(t *testing.T) *observability.Observabilidade {
	t.Helper()
	obs, err := observability.Iniciar(context.Background(), "error", "")
	if err != nil {
		t.Fatalf("observabilidade: %v", err)
	}
	return obs
}

func conectarBroker(t *testing.T) *adaptadoramqp.Conexao {
	t.Helper()
	conexao, err := adaptadoramqp.Conectar(ambiente.RabbitURL)
	if err != nil {
		t.Fatalf("conectar ao broker: %v", err)
	}
	t.Cleanup(conexao.Fechar)
	return conexao
}

// espiao assina reserva.criada em uma fila temporária, para ver o que foi
// realmente publicado.
func espiao(t *testing.T) <-chan amqp091.Delivery {
	t.Helper()

	conn, err := amqp091.Dial(ambiente.RabbitURL)
	if err != nil {
		t.Fatalf("espião: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	canal, err := conn.Channel()
	if err != nil {
		t.Fatalf("espião: %v", err)
	}
	if err := canal.ExchangeDeclare(adaptadoramqp.Exchange, "topic", true, false, false, false, nil); err != nil {
		t.Fatalf("espião: %v", err)
	}
	fila, err := canal.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatalf("espião: %v", err)
	}
	if err := canal.QueueBind(fila.Name, usecase.RoutingKeyReservaCriada, adaptadoramqp.Exchange, false, nil); err != nil {
		t.Fatalf("espião: %v", err)
	}
	entregas, err := canal.Consume(fila.Name, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("espião: %v", err)
	}
	return entregas
}

// TestFatoSobreviveABrokerIndisponivel cobre SC-005: o bloqueio é concedido com
// o broker fora do ar e o fato é entregue quando ele volta — sem duplicar a
// reserva e sem invalidar o bloqueio.
func TestFatoSobreviveABrokerIndisponivel(t *testing.T) {
	c := montarCenario(t, false)
	ctx := context.Background()
	entregas := espiao(t)

	// Publicador apontando para uma conexão que será fechada: é o equivalente
	// funcional do broker fora do ar no instante da concessão.
	conexaoCaida, err := adaptadoramqp.Conectar(ambiente.RabbitURL)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	conexaoCaida.Fechar()

	publicadorCaido := &adaptadoramqp.Publicador{Conexao: conexaoCaida, Banco: c.Banco, Obs: observabilidade(t)}

	sessao, reservaID := reservaPendente(t, c)
	_ = sessao

	// O bloqueio já está concedido; a publicação falha.
	if n, _ := publicadorCaido.Drenar(ctx, 10); n != 0 {
		t.Fatalf("publicou %d fatos com o broker fora do ar", n)
	}

	var pendentes int
	if err := c.Pool.QueryRow(ctx,
		`SELECT count(*) FROM outbox_eventos WHERE message_id = $1 AND publicado_em IS NULL`,
		reservaID).Scan(&pendentes); err != nil {
		t.Fatal(err)
	}
	if pendentes != 1 {
		t.Fatalf("fato pendente = %d, esperado 1", pendentes)
	}
	// O bloqueio permanece válido apesar da falha de publicação.
	if got := c.statusReserva(t, reservaID); got != "PENDENTE" {
		t.Errorf("reserva = %s — falha de publicação invalidou o bloqueio", got)
	}

	// O broker volta.
	// A caixa de saída é compartilhada com os outros testes da suíte, então a
	// asserção é sobre ESTE fato, não sobre a contagem do lote.
	publicador := &adaptadoramqp.Publicador{Conexao: conectarBroker(t), Banco: c.Banco, Obs: observabilidade(t)}
	if _, err := publicador.Drenar(ctx, 200); err != nil {
		t.Fatalf("drenar após retorno: %v", err)
	}
	var publicado bool
	if err := c.Pool.QueryRow(ctx,
		`SELECT publicado_em IS NOT NULL FROM outbox_eventos WHERE message_id = $1`,
		reservaID).Scan(&publicado); err != nil {
		t.Fatal(err)
	}
	if !publicado {
		t.Fatal("o fato continuou pendente depois do retorno do broker")
	}

	// A fila do espião recebe todo reserva.criada da suíte; procuramos o fato
	// desta reserva, sem supor que ele seja o primeiro.
	prazo := time.After(15 * time.Second)
	for {
		select {
		case msg := <-entregas:
			var evento usecase.EventoReservaCriada
			if err := json.Unmarshal(msg.Body, &evento); err != nil {
				t.Fatalf("payload inválido: %v", err)
			}
			if evento.ReservaID != reservaID {
				continue // fato de outro teste
			}
			if msg.MessageId != reservaID {
				t.Errorf("message_id = %q, esperado o id da reserva (chave de deduplicação)", msg.MessageId)
			}
			// SC-009: o contexto capturado na concessão viaja com o fato.
			if msg.Headers["traceparent"] == nil {
				t.Error("fato publicado sem traceparent — rastro quebra no broker")
			}
			goto entregue
		case <-prazo:
			t.Fatal("fato não chegou após o retorno do broker")
		}
	}
entregue:

	// Publicar de novo não pode reenviar o que já foi confirmado.
	var pendenteDepois int
	if err := c.Pool.QueryRow(ctx,
		`SELECT count(*) FROM outbox_eventos WHERE message_id = $1 AND publicado_em IS NULL`,
		reservaID).Scan(&pendenteDepois); err != nil {
		t.Fatal(err)
	}
	if pendenteDepois != 0 {
		t.Errorf("fato voltou a ficar pendente depois de publicado")
	}
}

// TestConsumoAplicaEfeitoEConfirma exercita o laço real de consumo: publica um
// desfecho de pagamento e verifica que o efeito chega ao banco.
func TestConsumoAplicaEfeitoEConfirma(t *testing.T) {
	c := montarCenario(t, false)
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()

	conexao := conectarBroker(t)
	if err := adaptadoramqp.ConsumirPagamentoSucesso(ctx, conexao, 8, observabilidade(t), c.Confirmar); err != nil {
		t.Fatalf("consumidor: %v", err)
	}

	sessao, reservaID := reservaPendente(t, c)
	publicarDesfecho(t, "pagamento.sucesso", reservaID, "PAGAMENTO_SUCESSO")

	aguardar(t, 15*time.Second, func() bool {
		return c.statusReserva(t, reservaID) == "CONFIRMADA"
	}, "reserva não foi confirmada pelo consumo")

	if got := c.statusPoltrona(t, sessao, "A1"); got != "OCUPADA" {
		t.Errorf("A1 = %s, esperado OCUPADA", got)
	}
}

// TestMensagemInvalidaVaiParaDLQ cobre FR-023: erro definitivo sai do fluxo
// normal para inspeção, sem travar o processamento das demais.
func TestMensagemInvalidaVaiParaDLQ(t *testing.T) {
	c := montarCenario(t, false)
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()

	conexao := conectarBroker(t)
	if err := adaptadoramqp.ConsumirPagamentoFalhou(ctx, conexao, 8, observabilidade(t), c.Cancelar); err != nil {
		t.Fatalf("consumidor: %v", err)
	}

	dlq := adaptadoramqp.NomeDLQ(adaptadoramqp.FilaPagamentoFalhou)
	antes := profundidade(t, dlq)

	// Corpo que não é JSON e corpo sem reserva_id: ambos definitivos.
	publicarBruto(t, "pagamento.falhou", []byte(`{{{ nao sou json`), "")
	publicarBruto(t, "pagamento.falhou", []byte(`{"evento":"PAGAMENTO_FALHOU"}`), "")

	aguardar(t, 15*time.Second, func() bool {
		return profundidade(t, dlq) >= antes+2
	}, "mensagens inválidas não chegaram à fila-morta")

	// E o consumo das mensagens boas continua funcionando.
	sessao, reservaID := reservaPendente(t, c)
	publicarDesfecho(t, "pagamento.falhou", reservaID, "PAGAMENTO_FALHOU")

	aguardar(t, 15*time.Second, func() bool {
		return c.statusReserva(t, reservaID) == "CANCELADA"
	}, "mensagem válida não foi processada depois das inválidas")

	if got := c.statusPoltrona(t, sessao, "A1"); got != "LIVRE" {
		t.Errorf("A1 = %s, esperado LIVRE", got)
	}
}

// TestSessaoCriadaProvisionaPeloConsumo liga o fato do catálogo ao estoque.
func TestSessaoCriadaProvisionaPeloConsumo(t *testing.T) {
	c := montarCenario(t, false)
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()

	conexao := conectarBroker(t)
	if err := adaptadoramqp.ConsumirSessaoCriada(ctx, conexao, 8, observabilidade(t), c.Provisionar); err != nil {
		t.Fatalf("consumidor: %v", err)
	}

	sessaoID := "sessao-por-evento-" + time.Now().UTC().Format("150405.000000")
	corpo, _ := json.Marshal(eventoDeSessao(sessaoID,
		usecase.LayoutPoltrona{Fileira: "A", Numero: 1, Tipo: "NORMAL"},
		usecase.LayoutPoltrona{Fileira: "A", Numero: 2, Tipo: "NORMAL"},
	))
	publicarBruto(t, "sessao.criada", corpo, sessaoID)

	aguardar(t, 20*time.Second, func() bool {
		return c.contarPorStatus(t, sessaoID)["LIVRE"] == 2
	}, "matriz não foi provisionada pelo consumo de sessao.criada")

	resultado, err := c.Bloquear.Executar(context.Background(), sessaoID, usuario, []string{"A1"})
	if err != nil || !resultado.Concedido {
		t.Fatalf("sessão provisionada por evento devia aceitar bloqueio: %v", err)
	}
}

// --- utilitários ---

func publicarDesfecho(t *testing.T, routingKey, reservaID, evento string) {
	t.Helper()
	corpo, _ := json.Marshal(usecase.DesfechoPagamento{
		Evento: evento, Versao: 1, ReservaID: reservaID,
		OcorridoEm: time.Now().UTC().Format(time.RFC3339),
	})
	publicarBruto(t, routingKey, corpo, reservaID)
}

func publicarBruto(t *testing.T, routingKey string, corpo []byte, messageID string) {
	t.Helper()

	conn, err := amqp091.Dial(ambiente.RabbitURL)
	if err != nil {
		t.Fatalf("publicar: %v", err)
	}
	defer conn.Close()

	canal, err := conn.Channel()
	if err != nil {
		t.Fatalf("publicar: %v", err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()

	err = canal.PublishWithContext(ctx, adaptadoramqp.Exchange, routingKey, false, false,
		amqp091.Publishing{
			ContentType: "application/json", DeliveryMode: amqp091.Persistent,
			MessageId: messageID, Body: corpo,
			Headers: amqp091.Table{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"},
		})
	if err != nil {
		t.Fatalf("publicar: %v", err)
	}
}

func profundidade(t *testing.T, fila string) int {
	t.Helper()

	conn, err := amqp091.Dial(ambiente.RabbitURL)
	if err != nil {
		t.Fatalf("inspecionar fila: %v", err)
	}
	defer conn.Close()

	canal, err := conn.Channel()
	if err != nil {
		t.Fatalf("inspecionar fila: %v", err)
	}
	info, err := canal.QueueInspect(fila)
	if err != nil {
		return 0
	}
	return info.Messages
}

func aguardar(t *testing.T, limite time.Duration, condicao func() bool, mensagem string) {
	t.Helper()
	prazo := time.Now().Add(limite)
	for time.Now().Before(prazo) {
		if condicao() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal(mensagem)
}

// publicarBrutoComHeader publica um desfecho de pagamento com um traceparent
// específico, para exercitar a propagação de contexto pelo broker.
func publicarBrutoComHeader(t *testing.T, routingKey, reservaID, traceparent string) {
	t.Helper()

	conn, err := amqp091.Dial(ambiente.RabbitURL)
	if err != nil {
		t.Fatalf("publicar: %v", err)
	}
	defer conn.Close()

	canal, err := conn.Channel()
	if err != nil {
		t.Fatalf("publicar: %v", err)
	}
	corpo, _ := json.Marshal(usecase.DesfechoPagamento{
		Evento: "PAGAMENTO_SUCESSO", Versao: 1, ReservaID: reservaID,
		OcorridoEm: time.Now().UTC().Format(time.RFC3339),
	})

	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	err = canal.PublishWithContext(ctx, adaptadoramqp.Exchange, routingKey, false, false,
		amqp091.Publishing{
			ContentType: "application/json", DeliveryMode: amqp091.Persistent,
			MessageId: reservaID, Body: corpo,
			Headers: amqp091.Table{"traceparent": traceparent},
		})
	if err != nil {
		t.Fatalf("publicar: %v", err)
	}
}
