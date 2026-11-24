package grpc

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/oseias/ingressos-golang/catalogo/internal/platform/obs"
)

// captura guarda os registros de log junto com o request_id do contexto.
type captura struct {
	mu        sync.Mutex
	registros []registro
}

type registro struct {
	msg       string
	requestID string
	attrs     map[string]string
}

func (c *captura) Enabled(context.Context, slog.Level) bool { return true }
func (c *captura) WithAttrs([]slog.Attr) slog.Handler       { return c }
func (c *captura) WithGroup(string) slog.Handler            { return c }
func (c *captura) Handle(ctx context.Context, r slog.Record) error {
	reg := registro{msg: r.Message, requestID: obs.RequestID(ctx), attrs: map[string]string{}}
	r.Attrs(func(a slog.Attr) bool { reg.attrs[a.Key] = a.Value.String(); return true })
	c.mu.Lock()
	c.registros = append(c.registros, reg)
	c.mu.Unlock()
	return nil
}

func capturarLogs(t *testing.T) *captura {
	t.Helper()
	c := &captura{}
	anterior := slog.Default()
	slog.SetDefault(slog.New(c))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	return c
}

func chamar(t *testing.T, metodo, requestID string, erro error) {
	t.Helper()
	ctx := context.Background()
	if requestID != "" {
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(obs.ChaveRequestID, requestID))
	}
	_, _ = InterceptadorRequestID()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: metodo},
		func(context.Context, any) (any, error) { return nil, erro })
}

func TestInterceptadorRegistraChamadaComRequestIDDoChamador(t *testing.T) {
	logs := capturarLogs(t)
	chamar(t, "/catalogo.v1.CatalogoService/ObterSessao", "req-abc.123", nil)

	if len(logs.registros) != 1 {
		t.Fatalf("esperava 1 registro, obteve %d", len(logs.registros))
	}
	r := logs.registros[0]
	if r.requestID != "req-abc.123" {
		t.Errorf("request_id do contexto = %q, esperava o enviado pelo chamador", r.requestID)
	}
	if r.attrs["metodo"] != "/catalogo.v1.CatalogoService/ObterSessao" || r.attrs["codigo"] != "OK" {
		t.Errorf("atributos inesperados: %v", r.attrs)
	}
}

func TestInterceptadorRegistraCodigoDeErro(t *testing.T) {
	logs := capturarLogs(t)
	chamar(t, "/catalogo.v1.CatalogoService/ObterSessao", "req-1", status.Error(codes.NotFound, "x"))
	if got := logs.registros[0].attrs["codigo"]; got != "NotFound" {
		t.Errorf("codigo = %q, esperava NotFound", got)
	}
}

func TestInterceptadorGeraRequestIDSemMetadado(t *testing.T) {
	logs := capturarLogs(t)
	chamar(t, "/catalogo.v1.CatalogoService/ObterSessao", "", nil)
	if logs.registros[0].requestID == "" {
		t.Error("sem metadado deveria gerar um request_id")
	}
}

func TestInterceptadorNaoRegistraSondaDeSaude(t *testing.T) {
	logs := capturarLogs(t)
	chamar(t, "/grpc.health.v1.Health/Check", "req-1", nil)
	if len(logs.registros) != 0 {
		t.Errorf("sonda de saúde não deveria gerar log, obteve %d", len(logs.registros))
	}
}
