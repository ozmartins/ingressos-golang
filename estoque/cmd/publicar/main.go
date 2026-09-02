package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func main() {
	if len(os.Args) < 2 {
		uso()
	}

	switch os.Args[1] {
	case "sessao-criada":
		publicarSessao(os.Args[2:])
	case "pagamento":
		publicarPagamento(os.Args[2:])
	default:
		uso()
	}
}

func uso() {
	fmt.Fprintln(os.Stderr, `uso:
  publicar sessao-criada -sessao=<uuid> -fileiras=A,B -assentos=10
  publicar pagamento -reserva=<uuid> -resultado=sucesso|falhou`)
	os.Exit(2)
}

func urlBroker() string {
	if url := os.Getenv("RABBITMQ_URL"); url != "" {
		return url
	}
	return "amqp://guest:guest@localhost:5672/"
}

func publicarSessao(args []string) {
	fs := flag.NewFlagSet("sessao-criada", flag.ExitOnError)
	sessao := fs.String("sessao", "", "identificador da sessão")
	fileiras := fs.String("fileiras", "A,B", "fileiras separadas por vírgula")
	assentos := fs.Int("assentos", 10, "assentos por fileira")
	_ = fs.Parse(args)

	if *sessao == "" {
		fmt.Fprintln(os.Stderr, "-sessao é obrigatório")
		os.Exit(2)
	}

	evento := usecase.EventoSessaoCriada{
		Evento: "SESSAO_CRIADA", Versao: 1,
		OcorridoEm: time.Now().UTC().Format(time.RFC3339),
		SessaoID:   *sessao,
	}
	for _, fileira := range strings.Split(*fileiras, ",") {
		fileira = strings.ToUpper(strings.TrimSpace(fileira))
		if fileira == "" {
			continue
		}
		for n := 1; n <= *assentos; n++ {
			evento.Poltronas = append(evento.Poltronas,
				usecase.LayoutPoltrona{Fileira: fileira, Numero: n, Tipo: "NORMAL"})
		}
	}

	corpo, _ := json.Marshal(evento)
	enviar("sessao.criada", *sessao, corpo)
	fmt.Printf("sessao.criada publicada: %s (%d poltronas)\n", *sessao, len(evento.Poltronas))
}

func publicarPagamento(args []string) {
	fs := flag.NewFlagSet("pagamento", flag.ExitOnError)
	reserva := fs.String("reserva", "", "identificador da reserva")
	resultado := fs.String("resultado", "sucesso", "sucesso|falhou")
	_ = fs.Parse(args)

	if *reserva == "" {
		fmt.Fprintln(os.Stderr, "-reserva é obrigatório")
		os.Exit(2)
	}

	routingKey := "pagamento.sucesso"
	nome := "PAGAMENTO_SUCESSO"
	if *resultado == "falhou" {
		routingKey = "pagamento.falhou"
		nome = "PAGAMENTO_FALHOU"
	}

	corpo, _ := json.Marshal(usecase.DesfechoPagamento{
		Evento: nome, Versao: 1,
		OcorridoEm: time.Now().UTC().Format(time.RFC3339),
		ReservaID:  *reserva,
	})
	enviar(routingKey, *reserva, corpo)
	fmt.Printf("%s publicado para a reserva %s\n", routingKey, *reserva)
}

func enviar(routingKey, messageID string, corpo []byte) {
	conn, err := amqp.Dial(urlBroker())
	if err != nil {
		fmt.Fprintf(os.Stderr, "conectar ao broker: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	canal, err := conn.Channel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "abrir canal: %v\n", err)
		os.Exit(1)
	}
	if err := canal.ExchangeDeclare("cinema.eventos", "topic", true, false, false, false, nil); err != nil {
		fmt.Fprintf(os.Stderr, "declarar exchange: %v\n", err)
		os.Exit(1)
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()

	err = canal.PublishWithContext(ctx, "cinema.eventos", routingKey, false, false,
		amqp.Publishing{
			ContentType: "application/json", DeliveryMode: amqp.Persistent,
			MessageId: messageID, Timestamp: time.Now().UTC(), Body: corpo,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "publicar: %v\n", err)
		os.Exit(1)
	}
}
