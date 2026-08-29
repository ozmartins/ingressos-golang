// Command estoque-simulado é um Servico-Estoque de mentira, para desenvolvimento
// local e para o roteiro de quickstart.md.
//
// Não faz parte do serviço entregue: existe para que a jornada de reserva possa
// ser exercitada de ponta a ponta sem depender do serviço real.
package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	estoquepb "github.com/oseias/ingressos-golang/catalogo/gen/pb/estoque"
)

type servidor struct {
	estoquepb.UnimplementedServicoEstoqueServer

	mu         sync.Mutex
	concedidas map[string]bool
}

// Poltronas cujo identificador começa com B são tratadas como já ocupadas, para
// que o roteiro de validação consiga provocar o 409 de propósito.
func (s *servidor) BloquearPoltronas(_ context.Context, req *estoquepb.SolicitacaoBloqueio) (*estoquepb.RespostaBloqueio, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range req.GetPoltronasIds() {
		chave := req.GetSessaoId() + ":" + p
		if strings.HasPrefix(p, "B") || s.concedidas[chave] {
			return &estoquepb.RespostaBloqueio{
				Sucesso:  false,
				Mensagem: "Uma ou mais poltronas selecionadas não estão disponíveis.",
			}, nil
		}
	}
	for _, p := range req.GetPoltronasIds() {
		s.concedidas[req.GetSessaoId()+":"+p] = true
	}

	log.Printf("bloqueio concedido: sessao=%s usuario=%s poltronas=%v",
		req.GetSessaoId(), req.GetUsuarioId(), req.GetPoltronasIds())

	return &estoquepb.RespostaBloqueio{
		Sucesso:   true,
		ReservaId: "9982a1b3-44c1-4221-a123-" + time.Now().Format("150405.000000"),
		Mensagem:  "Poltronas reservadas com sucesso.",
		ExpiraEm:  time.Now().Add(10 * time.Minute).Unix(),
	}, nil
}

func main() {
	endereco := os.Getenv("LISTEN_ADDR")
	if endereco == "" {
		endereco = ":50051"
	}
	lis, err := net.Listen("tcp", endereco)
	if err != nil {
		log.Fatalf("escutando em %s: %v", endereco, err)
	}
	s := grpc.NewServer()
	estoquepb.RegisterServicoEstoqueServer(s, &servidor{concedidas: map[string]bool{}})
	log.Printf("estoque simulado ouvindo em %s", endereco)
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
