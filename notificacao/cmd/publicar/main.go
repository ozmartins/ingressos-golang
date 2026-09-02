package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	var (
		url       = flag.String("url", env("AMQP_URL", "amqp://guest:guest@localhost:5672/"), "URL do broker")
		exchange  = flag.String("exchange", env("AMQP_EXCHANGE", "cinema.eventos"), "exchange de publicação")
		reserva   = flag.String("reserva", "", "identificador da reserva")
		usuario   = flag.String("usuario", "", "identificador da pessoa")
		transacao = flag.String("transacao", "", "identificador da transação (opcional)")
		cru       = flag.String("cru", "", "payload cru, para exercitar anúncio malformado")
	)
	flag.Parse()

	corpo, err := montar(*reserva, *usuario, *transacao, *cru)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := publicar(*url, *exchange, corpo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("publicado em %s com a chave pagamento.sucesso: %s\n", *exchange, corpo)
}

func montar(reserva, usuario, transacao, cru string) ([]byte, error) {
	if cru != "" {
		return []byte(cru), nil
	}
	if reserva == "" || usuario == "" {
		return nil, fmt.Errorf("informe -reserva e -usuario, ou -cru com o payload")
	}
	if transacao == "" {
		transacao = "e402a129-8812-4211-b123-000129381293"
	}
	agora := time.Now().UTC().Format(time.RFC3339)
	return json.Marshal(map[string]any{
		"evento": "PAGAMENTO_SUCESSO", "versao": 1, "ocorrido_em": agora,
		"transacao_id": transacao, "reserva_id": reserva, "usuario_id": usuario,
		"valor_total": 84.00, "pago_em": agora,
	})
}

func publicar(url, exchange string, corpo []byte) error {
	conexao, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("conectar: %w", err)
	}
	defer conexao.Close()

	canal, err := conexao.Channel()
	if err != nil {
		return fmt.Errorf("abrir canal: %w", err)
	}
	defer canal.Close()

	if err := canal.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declarar exchange: %w", err)
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	return canal.PublishWithContext(ctx, exchange, "pagamento.sucesso", false, false,
		amqp.Publishing{ContentType: "application/json", Body: corpo, Timestamp: time.Now()})
}

func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}
