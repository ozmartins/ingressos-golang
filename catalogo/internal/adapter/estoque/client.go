// Package estoque é o adaptador de saída para o Servico-Estoque.
package estoque

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	estoquepb "github.com/oseias/ingressos-golang/catalogo/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"

	"go.opentelemetry.io/otel/metric"
)

// Cliente conversa com o Servico-Estoque.
//
// Uma conexão duradoura, um timeout por chamada e nenhuma política de
// retentativa: bloquear poltronas não é idempotente do ponto de vista do
// cliente, e repetir um pedido que já pode ter sido recebido criaria um bloqueio
// órfão do outro lado (constituição, princípio V).
type Cliente struct {
	rpc      estoquepb.ServicoEstoqueClient
	conn     *grpc.ClientConn
	timeout  time.Duration
	breaker  *RecusaRapida
	metricas *observability.Metricas
}

type Opcoes struct {
	Endereco        string
	Timeout         time.Duration
	FalhasParaAbrir uint32
	IntervaloAberto time.Duration
	Metricas        *observability.Metricas
}

func NovoCliente(opts Opcoes) (*Cliente, error) {
	conn, err := grpc.NewClient(opts.Endereco, append(
		OpcoesDeConexao(), grpc.WithTransportCredentials(insecure.NewCredentials()))...)
	if err != nil {
		return nil, fmt.Errorf("criando cliente do estoque: %w", err)
	}
	return NovoClienteComConexao(conn, opts), nil
}

// OpcoesDeConexao são as opções que toda conexão com o estoque precisa ter,
// inclusive as montadas nos testes.
//
// Existe porque a instrumentação é o que propaga o contexto de rastreamento
// (FR-036): deixá-la só dentro de NovoCliente faria o teste que injeta a própria
// conexão exercitar um caminho sem propagação — e passar sem provar nada.
func OpcoesDeConexao() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
}

// NovoClienteComConexao permite injetar uma conexão pronta — é o que torna os
// testes com bufconn possíveis sem abrir porta de rede.
func NovoClienteComConexao(conn *grpc.ClientConn, opts Opcoes) *Cliente {
	return &Cliente{
		rpc:      estoquepb.NewServicoEstoqueClient(conn),
		conn:     conn,
		timeout:  opts.Timeout,
		breaker:  NovaRecusaRapida(opts.FalhasParaAbrir, opts.IntervaloAberto, opts.Metricas),
		metricas: opts.Metricas,
	}
}

func (c *Cliente) Fechar() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// BloquearPoltronas encaminha a solicitação e traduz o desfecho.
func (c *Cliente) BloquearPoltronas(ctx context.Context, s reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error) {
	inicio := time.Now()

	resposta, desfecho, err := c.breaker.Executar(func() (*estoquepb.RespostaBloqueio, error) {
		ctxChamada, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.rpc.BloquearPoltronas(ctxChamada, &estoquepb.SolicitacaoBloqueio{
			SessaoId:     s.SessaoID,
			PoltronasIds: s.PoltronasIDs,
			UsuarioId:    s.UsuarioID,
		})
	})

	resultado, erroDominio := traduzir(resposta, desfecho, err)
	c.registrarMetricas(ctx, time.Since(inicio), desfechoFinal(desfecho, err, erroDominio))
	return resultado, erroDominio
}

func (c *Cliente) registrarMetricas(ctx context.Context, d time.Duration, desfecho string) {
	if c.metricas == nil {
		return
	}
	attrs := metric.WithAttributes(observability.RotuloDesfecho(desfecho))
	c.metricas.EstoqueDuracao.Record(ctx, d.Seconds(), attrs)
	c.metricas.EstoqueTotal.Add(ctx, 1, attrs)
}
