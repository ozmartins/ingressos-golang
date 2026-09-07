package estoque

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	estoquepb "github.com/oseias/ingressos-golang/catalogo/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"

	"go.opentelemetry.io/otel/metric"
)

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

	// Material do canal mTLS. O estoque exige certificado de cliente, e é por
	// ele que sabe quem está chamando: o `usuario_id` vai no corpo justamente
	// porque a identidade do serviço vem daqui.
	CAFile   string
	CertFile string
	KeyFile  string
}

func NovoCliente(opts Opcoes) (*Cliente, error) {
	credenciais, err := credenciaisMTLS(opts)
	if err != nil {
		return nil, fmt.Errorf("criando cliente do estoque: %w", err)
	}

	conn, err := grpc.NewClient(opts.Endereco, append(
		OpcoesDeConexao(), grpc.WithTransportCredentials(credenciais))...)
	if err != nil {
		return nil, fmt.Errorf("criando cliente do estoque: %w", err)
	}
	return NovoClienteComConexao(conn, opts), nil
}

// A CA é a do estoque, e não uma autoridade pública: ela não está em nenhum
// pool do sistema, então o pool é montado à mão. Sem `InsecureSkipVerify` — o
// ponto do mTLS é que os dois lados se verifiquem, e desligar metade disso
// deixaria só a aparência.
func credenciaisMTLS(opts Opcoes) (credentials.TransportCredentials, error) {
	par, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("carregando o par de cliente: %w", err)
	}

	pemCA, err := os.ReadFile(opts.CAFile)
	if err != nil {
		return nil, fmt.Errorf("lendo a CA do estoque: %w", err)
	}
	raiz := x509.NewCertPool()
	if !raiz.AppendCertsFromPEM(pemCA) {
		return nil, fmt.Errorf("a CA em %s não contém certificado PEM válido", opts.CAFile)
	}

	// O `ServerName` não é informado: o gRPC o deriva do endereço de conexão, e
	// o certificado do servidor cobre tanto `estoque` (dentro do compose) como
	// `localhost` (fora dele).
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{par},
		RootCAs:      raiz,
		MinVersion:   tls.VersionTLS13,
	}), nil
}

func OpcoesDeConexao() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
}

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

func (c *Cliente) BloquearPoltronas(ctx context.Context, s reserva.SolicitacaoReserva) (reserva.ResultadoReserva, error) {
	inicio := time.Now()

	resposta, desfecho, err := c.breaker.Executar(func() (*estoquepb.RespostaBloqueio, error) {
		ctxChamada, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.rpc.BloquearPoltronas(ctxChamada, &estoquepb.SolicitacaoBloqueio{
			SessaoId:     s.SessaoID,
			PoltronasIds: s.PoltronasIDs,
			ValorTotal:   s.ValorTotal,
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
