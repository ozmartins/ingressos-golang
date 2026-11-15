// Package catalogo é o adaptador de saída que consulta o serviço de catálogo
// por gRPC (CatalogoService.ObterSessao) para precificar e validar reservas.
package catalogo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/catalogo/v1"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/sessao"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/obs"
)

type Opcoes struct {
	// Endereco do servidor gRPC do catálogo (host:porta).
	Endereco        string
	Timeout         time.Duration
	FalhasParaAbrir uint32
	IntervaloAberto time.Duration
}

type Cliente struct {
	conn    *grpc.ClientConn
	api     pb.CatalogoServiceClient
	timeout time.Duration
	breaker *gobreaker.CircuitBreaker[sessao.Sessao]
}

// NovoCliente prepara a conexão (preguiçosa: não falha se o catálogo ainda não
// subiu). O otelgrpc abre um span de cliente e injeta o traceparent na chamada:
// é assim que o trace da reserva atravessa os dois serviços.
func NovoCliente(opts Opcoes) (*Cliente, error) {
	conn, err := grpc.NewClient(opts.Endereco,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("catálogo gRPC %q: %w", opts.Endereco, err)
	}
	c := NovoClienteComAPI(opts, pb.NewCatalogoServiceClient(conn))
	c.conn = conn
	return c, nil
}

// NovoClienteComAPI permite injetar o cliente gerado (testes usam bufconn).
func NovoClienteComAPI(opts Opcoes, api pb.CatalogoServiceClient) *Cliente {
	return &Cliente{
		api:     api,
		timeout: opts.Timeout,
		breaker: gobreaker.NewCircuitBreaker[sessao.Sessao](gobreaker.Settings{
			Name:        "catalogo.sessao",
			MaxRequests: 1,
			Timeout:     opts.IntervaloAberto,
			OnStateChange: func(nome string, de, para gobreaker.State) {
				obs.EstadoDoBreaker(nome, para.String())
				slog.Warn("disjuntor mudou de estado", "breaker", nome, "de", de.String(), "para", para.String())
			},
			ReadyToTrip: func(c gobreaker.Counts) bool {
				return c.ConsecutiveFailures >= opts.FalhasParaAbrir
			},
			IsSuccessful: func(err error) bool {
				return err == nil || errors.Is(err, shared.ErrSessaoNaoEncontrada)
			},
		}),
	}
}

// Fechar encerra a conexão gRPC.
func (c *Cliente) Fechar() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Cliente) BuscarSessao(ctx context.Context, sessaoID string) (sessao.Sessao, error) {
	s, err := c.breaker.Execute(func() (sessao.Sessao, error) { return c.buscar(ctx, sessaoID) })
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return sessao.Sessao{}, fmt.Errorf("%w: %v", shared.ErrCatalogoIndisponivel, err)
	}
	return s, err
}

func (c *Cliente) buscar(ctx context.Context, sessaoID string) (sessao.Sessao, error) {
	ctx, cancelar := context.WithTimeout(ctx, c.timeout)
	defer cancelar()

	resp, err := c.api.ObterSessao(ctx, &pb.ObterSessaoRequest{Id: sessaoID})
	if err != nil {
		return sessao.Sessao{}, traduzirErro(err, sessaoID)
	}

	if resp.GetStatus() == "" || resp.GetPrecoBase() == "" ||
		resp.GetDataHoraInicio() == nil || resp.GetDataHoraInicio().CheckValid() != nil {
		return sessao.Sessao{}, fmt.Errorf("%w: sessão %s incompleta", shared.ErrRespostaInvalidaDoCatalogo, sessaoID)
	}
	return sessao.Sessao{
		ID:        sessaoID,
		Status:    resp.GetStatus(),
		Inicio:    resp.GetDataHoraInicio().AsTime(),
		PrecoBase: resp.GetPrecoBase(),
	}, nil
}

func traduzirErro(err error, sessaoID string) error {
	switch status.Code(err) {
	case codes.NotFound:
		return fmt.Errorf("%w: %s", shared.ErrSessaoNaoEncontrada, sessaoID)
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled, codes.ResourceExhausted, codes.Aborted:
		return fmt.Errorf("%w: %v", shared.ErrCatalogoIndisponivel, err)
	default:
		// Internal, InvalidArgument, Unknown etc.: o catálogo respondeu, mas
		// não algo que o estoque saiba usar.
		return fmt.Errorf("%w: %v", shared.ErrRespostaInvalidaDoCatalogo, err)
	}
}
