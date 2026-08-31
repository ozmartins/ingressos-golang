// Command publicar gera fatos reserva.criada no formato que o Servico-Pagamento
// exige, para o roteiro de quickstart.md.
//
// Existe por uma razão específica e temporária: o Servico-Estoque publica
// reserva.criada SEM valor_total e SEM forma_pagamento (ver
// estoque/internal/usecase/bloquear_poltronas.go, EventoReservaCriada), então
// nenhum evento real dele é processável por este serviço hoje. A divergência foi
// decidida com o mantenedor e está registrada em research.md D1 e na caixa de
// dependência de integração de contracts/eventos.md §1.
//
// Quando o estoque passar a propagar os dois campos, este comando deixa de ser
// necessário para validar o caminho feliz.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type eventoReservaCriada struct {
	Evento         string      `json:"evento"`
	Versao         int         `json:"versao"`
	OcorridoEm     string      `json:"ocorrido_em"`
	ReservaID      string      `json:"reserva_id"`
	SessaoID       string      `json:"sessao_id"`
	UsuarioID      string      `json:"usuario_id"`
	PoltronasIDs   []string    `json:"poltronas_ids"`
	ValorTotal     json.Number `json:"valor_total"`
	FormaPagamento string      `json:"forma_pagamento"`
	ExpiraEm       string      `json:"expira_em"`
}

func main() {
	var (
		url      = flag.String("amqp", env("AMQP_URL", "amqp://guest:guest@localhost:5672/"), "URL do broker")
		exchange = flag.String("exchange", env("AMQP_EXCHANGE", "cinema.eventos"), "exchange")
		reserva  = flag.String("reserva", "", "UUID da reserva (padrão: gerado)")
		usuario  = flag.String("usuario", "", "UUID da pessoa (padrão: gerado)")
		valor    = flag.String("valor", "84.00", "valor total")
		forma    = flag.String("forma", "PIX", "PIX ou CARTAO_CREDITO")
		expira   = flag.Duration("expira-em", 10*time.Minute, "prazo relativo (negativo = já expirada)")
		rajada   = flag.Int("rajada", 0, "publica N intenções distintas")
		espiar   = flag.String("espiar", "", "em vez de publicar, drena os fatos de uma routing key (ex.: pagamento.sucesso)")
	)
	flag.Parse()

	conexao, err := amqp.Dial(*url)
	if err != nil {
		morrer(err)
	}
	defer conexao.Close()

	canal, err := conexao.Channel()
	if err != nil {
		morrer(err)
	}
	if err := canal.Confirm(false); err != nil {
		morrer(err)
	}

	if *espiar != "" {
		if err := espiarFatos(canal, *exchange, *espiar); err != nil {
			morrer(err)
		}
		return
	}

	n := *rajada
	if n <= 0 {
		n = 1
	}
	for i := 0; i < n; i++ {
		r := *reserva
		if r == "" || n > 1 {
			r = uuid.NewString()
		}
		u := *usuario
		if u == "" {
			u = uuid.NewString()
		}
		if err := publicar(canal, *exchange, eventoReservaCriada{
			Evento: "RESERVA_CRIADA", Versao: 1,
			OcorridoEm:     time.Now().UTC().Format(time.RFC3339),
			ReservaID:      r,
			SessaoID:       uuid.NewString(),
			UsuarioID:      u,
			PoltronasIDs:   []string{"A1", "A2"},
			ValorTotal:     json.Number(*valor),
			FormaPagamento: *forma,
			ExpiraEm:       time.Now().UTC().Add(*expira).Format(time.RFC3339),
		}); err != nil {
			morrer(err)
		}
		if n == 1 {
			fmt.Printf("reserva.criada publicado: reserva=%s usuario=%s valor=%s forma=%s\n", r, u, *valor, *forma)
		}
	}
	if n > 1 {
		fmt.Printf("%d intenções publicadas\n", n)
	}
}

func publicar(canal *amqp.Channel, exchange string, e eventoReservaCriada) error {
	corpo, err := json.Marshal(e)
	if err != nil {
		return err
	}
	conf, err := canal.PublishWithDeferredConfirmWithContext(context.Background(),
		exchange, "reserva.criada", false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    e.ReservaID,
			Body:         corpo,
		})
	if err != nil {
		return err
	}
	ok, err := conf.WaitContext(context.Background())
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("broker recusou a publicação")
	}
	return nil
}

// espiarFatos liga uma fila temporária à routing key pedida e imprime o que
// chegar por 5 segundos. Serve ao roteiro de quickstart.md para ver os anúncios
// sem competir com os consumidores reais.
func espiarFatos(canal *amqp.Channel, exchange, routingKey string) error {
	q, err := canal.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return err
	}
	if err := canal.QueueBind(q.Name, routingKey, exchange, false, nil); err != nil {
		return err
	}
	entregas, err := canal.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		return err
	}
	fmt.Printf("espiando %s por 5s...\n", routingKey)
	limite := time.After(5 * time.Second)
	for {
		select {
		case d := <-entregas:
			fmt.Println(string(d.Body))
		case <-limite:
			return nil
		}
	}
}

func env(k, padrao string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return padrao
}

func morrer(err error) {
	fmt.Fprintln(os.Stderr, "publicar:", err)
	os.Exit(1)
}
