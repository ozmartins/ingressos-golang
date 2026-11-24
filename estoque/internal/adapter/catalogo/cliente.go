// Package catalogo é o adaptador de saída que consulta o serviço de catálogo
// por gRPC (CatalogoService.ObterSessao) para precificar e validar reservas.
package catalogo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/catalogo/v1"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/sessao"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/obs"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/retry"
)

type Opcoes struct {
	// Endereco do servidor gRPC do catálogo (host:porta).
	Endereco string
	// TLS: TLSCA é a CA que assina o certificado do catálogo. TLSCert/TLSKey, se
	// informados, apresentam o certificado de cliente (mTLS).
	TLSCA, TLSCert, TLSKey string
	// Inseguro autoriza, só em desenvolvimento, o canal sem TLS quando TLSCA está
	// vazio. Sem TLSCA e sem Inseguro, NovoCliente falha.
	Inseguro        bool
	Timeout         time.Duration
	FalhasParaAbrir uint32
	IntervaloAberto time.Duration
	// Tentativas é o total de chamadas por consulta (0 ou 1 = sem retry). Só
	// indisponibilidade (timeout, conexão recusada) é repetida, com backoff
	// exponencial a partir de EsperaBase; erro de negócio nunca.
	Tentativas int
	EsperaBase time.Duration
	// Identidade do estoque junto ao catálogo (OAuth2 client credentials). Se
	// TokenURL estiver vazio, as chamadas seguem sem token (só para testes).
	TokenURL     string
	ClientID     string
	ClientSecret string
	TokenTimeout time.Duration
	// CacheTTL é por quanto tempo a última sessão consultada com sucesso pode
	// servir de fallback quando o catálogo está indisponível ou o disjuntor
	// está aberto (0 = sem fallback: a reserva falha).
	CacheTTL time.Duration
}

type Cliente struct {
	conn    *grpc.ClientConn
	api     pb.CatalogoServiceClient
	timeout time.Duration
	retry   retry.Politica

	// Fallback: última sessão lida com sucesso, usada só enquanto o catálogo
	// está indisponível. Nunca encobre erro de negócio.
	cacheTTL time.Duration
	agora    func() time.Time
	mu       sync.Mutex
	cache    map[string]sessaoEmCache
	breaker  *gobreaker.CircuitBreaker[sessao.Sessao]
}

// NovoCliente prepara a conexão (preguiçosa: não falha se o catálogo ainda não
// subiu). O otelgrpc abre um span de cliente e injeta o traceparent na chamada:
// é assim que o trace da reserva atravessa os dois serviços.
func NovoCliente(opts Opcoes) (*Cliente, error) {
	transporte, err := credenciaisDeTransporte(opts)
	if err != nil {
		return nil, err
	}
	dial := []grpc.DialOption{
		grpc.WithTransportCredentials(transporte),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(propagarRequestID),
	}
	if opts.TokenURL != "" {
		fonte := NovaFonteDeToken(opts.TokenURL, opts.ClientID, opts.ClientSecret, opts.TokenTimeout)
		fonte.ExigeTLS = opts.TLSCA != ""
		dial = append(dial, grpc.WithPerRPCCredentials(fonte))
	}
	conn, err := grpc.NewClient(opts.Endereco, dial...)
	if err != nil {
		return nil, fmt.Errorf("catálogo gRPC %q: %w", opts.Endereco, err)
	}
	c := NovoClienteComAPI(opts, pb.NewCatalogoServiceClient(conn))
	c.conn = conn
	return c, nil
}

// credenciaisDeTransporte devolve TLS (com mTLS opcional) quando há CA
// configurada. Sem CA, só segue em texto puro se Inseguro for pedido de forma
// explícita; do contrário recusa, para não rodar sem TLS por esquecimento.
func credenciaisDeTransporte(opts Opcoes) (credentials.TransportCredentials, error) {
	if opts.TLSCA == "" {
		if !opts.Inseguro {
			return nil, errors.New("gRPC para o catálogo exige TLS: defina CATALOGO_GRPC_TLS_CA (ou CATALOGO_GRPC_INSECURE=true, só em desenvolvimento)")
		}
		slog.Warn("gRPC para o catálogo SEM TLS (CATALOGO_GRPC_INSECURE=true): só para desenvolvimento")
		return insecure.NewCredentials(), nil
	}
	pem, err := os.ReadFile(opts.TLSCA)
	if err != nil {
		return nil, fmt.Errorf("ler CA do catálogo: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("CA do catálogo sem certificado PEM válido")
	}
	cfg := &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13}
	if (opts.TLSCert == "") != (opts.TLSKey == "") {
		return nil, errors.New("CATALOGO_GRPC_TLS_CERT e CATALOGO_GRPC_TLS_KEY: informe os dois ou nenhum")
	}
	if opts.TLSCert != "" {
		par, err := tls.LoadX509KeyPair(opts.TLSCert, opts.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("carregar certificado de cliente: %w", err)
		}
		cfg.Certificates = []tls.Certificate{par}
	}
	return credentials.NewTLS(cfg), nil
}

// NovoClienteComAPI permite injetar o cliente gerado (testes usam bufconn).
func NovoClienteComAPI(opts Opcoes, api pb.CatalogoServiceClient) *Cliente {
	return &Cliente{
		api:      api,
		timeout:  opts.Timeout,
		cacheTTL: opts.CacheTTL,
		agora:    time.Now,
		cache:    map[string]sessaoEmCache{},
		retry: retry.Politica{
			Tentativas: opts.Tentativas,
			Base:       opts.EsperaBase,
			Maximo:     opts.EsperaBase * 8,
			Retentavel: func(err error) bool { return errors.Is(err, shared.ErrCatalogoIndisponivel) },
		},
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
				return err == nil || errors.Is(err, shared.ErrSessaoNaoEncontrada) || errors.Is(err, context.Canceled)
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
	// O retry fica dentro do disjuntor: uma consulta que esgota as tentativas
	// conta como uma única falha para ele.
	s, err := c.breaker.Execute(func() (s sessao.Sessao, err error) {
		err = retry.Executar(ctx, "catalogo.ObterSessao", c.retry, func() (e error) {
			s, e = c.buscar(ctx, sessaoID)
			return e
		})
		return s, err
	})
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		err = fmt.Errorf("%w: %v", shared.ErrCatalogoIndisponivel, err)
	}
	switch {
	case err == nil:
		c.guardar(s)
	case errors.Is(err, shared.ErrSessaoNaoEncontrada):
		c.esquecer(sessaoID)
	case errors.Is(err, shared.ErrCatalogoIndisponivel):
		// Fallback: com o catálogo fora (ou o disjuntor aberto), segue com a
		// última sessão conhecida, se ainda recente. Preço e status podem estar
		// defasados por até CacheTTL; é o preço de não derrubar as reservas.
		if em, ok := c.recente(sessaoID); ok {
			obs.FallbackCatalogo()
			slog.Warn("catálogo indisponível, usando sessão em cache",
				"sessao_id", sessaoID, "idade", c.agora().Sub(em.lidaEm).Round(time.Second).String(), "erro", err)
			return em.sessao, nil
		}
	}
	return s, err
}

type sessaoEmCache struct {
	sessao sessao.Sessao
	lidaEm time.Time
}

const limiteCache = 1000

func (c *Cliente) guardar(s sessao.Sessao) {
	if c.cacheTTL <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	agora := c.agora()
	if len(c.cache) >= limiteCache { // poda as expiradas para não crescer sem limite
		for id, e := range c.cache {
			if agora.Sub(e.lidaEm) > c.cacheTTL {
				delete(c.cache, id)
			}
		}
	}
	c.cache[s.ID] = sessaoEmCache{sessao: s, lidaEm: agora}
}

func (c *Cliente) esquecer(id string) {
	c.mu.Lock()
	delete(c.cache, id)
	c.mu.Unlock()
}

func (c *Cliente) recente(id string) (sessaoEmCache, bool) {
	if c.cacheTTL <= 0 {
		return sessaoEmCache{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[id]
	if !ok || c.agora().Sub(e.lidaEm) > c.cacheTTL {
		return sessaoEmCache{}, false
	}
	return e, true
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
	case codes.Canceled:
		// Quem chamou desistiu: não é falha do catálogo, nem vale re-tentar.
		return fmt.Errorf("consulta ao catálogo cancelada: %w", context.Canceled)
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
		return fmt.Errorf("%w: %v", shared.ErrCatalogoIndisponivel, err)
	default:
		// Internal, InvalidArgument, Unknown etc.: o catálogo respondeu, mas
		// não algo que o estoque saiba usar.
		return fmt.Errorf("%w: %v", shared.ErrRespostaInvalidaDoCatalogo, err)
	}
}

// propagarRequestID leva o request id do contexto para o catálogo no metadado
// x-request-id, para que os logs dos dois serviços se correlacionem.
func propagarRequestID(ctx context.Context, metodo string, req, resp any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if id := obs.RequestID(ctx); id != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, obs.ChaveRequestID, id)
	}
	return invoker(ctx, metodo, req, resp, cc, opts...)
}
