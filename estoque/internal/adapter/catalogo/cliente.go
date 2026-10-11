// Package catalogo é o cliente HTTP do serviço de catálogo: o estoque o usa
// para saber o preço e o status da sessão antes de bloquear poltronas.
package catalogo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/sessao"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type Opcoes struct {
	URLBase         string
	Timeout         time.Duration
	FalhasParaAbrir uint32
	IntervaloAberto time.Duration
}

type Cliente struct {
	base    string
	http    *http.Client
	breaker *gobreaker.CircuitBreaker[sessao.Sessao]
}

func NovoCliente(opts Opcoes) *Cliente {
	return NovoClienteComTransporte(opts, otelhttp.NewTransport(http.DefaultTransport))
}

func NovoClienteComTransporte(opts Opcoes, transporte http.RoundTripper) *Cliente {
	return &Cliente{
		base: strings.TrimRight(opts.URLBase, "/"),
		http: &http.Client{Transport: transporte, Timeout: opts.Timeout},
		breaker: gobreaker.NewCircuitBreaker[sessao.Sessao](gobreaker.Settings{
			Name:        "catalogo.sessao",
			MaxRequests: 1,
			Timeout:     opts.IntervaloAberto,
			ReadyToTrip: func(c gobreaker.Counts) bool {
				return c.ConsecutiveFailures >= opts.FalhasParaAbrir
			},
			// Sessão inexistente é resposta válida do catálogo, não falha dele.
			IsSuccessful: func(err error) bool {
				return err == nil || errors.Is(err, shared.ErrSessaoNaoEncontrada)
			},
		}),
	}
}

type sessaoJSON struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	DataHoraInicio string `json:"data_hora_inicio"`
	PrecoBase      string `json:"preco_base"`
}

func (c *Cliente) BuscarSessao(ctx context.Context, sessaoID string) (sessao.Sessao, error) {
	s, err := c.breaker.Execute(func() (sessao.Sessao, error) { return c.buscar(ctx, sessaoID) })
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return sessao.Sessao{}, fmt.Errorf("%w: %v", shared.ErrCatalogoIndisponivel, err)
	}
	return s, err
}

func (c *Cliente) buscar(ctx context.Context, sessaoID string) (sessao.Sessao, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.base+"/api/v1/sessoes/"+url.PathEscape(sessaoID), nil)
	if err != nil {
		return sessao.Sessao{}, fmt.Errorf("%w: %v", shared.ErrCatalogoIndisponivel, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return sessao.Sessao{}, fmt.Errorf("%w: %v", shared.ErrCatalogoIndisponivel, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return sessao.Sessao{}, fmt.Errorf("%w: %s", shared.ErrSessaoNaoEncontrada, sessaoID)
	case resp.StatusCode >= 500:
		return sessao.Sessao{}, fmt.Errorf("%w: catálogo respondeu %d", shared.ErrCatalogoIndisponivel, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return sessao.Sessao{}, fmt.Errorf("%w: catálogo respondeu %d", shared.ErrRespostaInvalidaDoCatalogo, resp.StatusCode)
	}

	var corpo sessaoJSON
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&corpo); err != nil {
		return sessao.Sessao{}, fmt.Errorf("%w: %v", shared.ErrRespostaInvalidaDoCatalogo, err)
	}
	inicio, err := time.Parse(time.RFC3339, corpo.DataHoraInicio)
	if err != nil || corpo.Status == "" || corpo.PrecoBase == "" {
		return sessao.Sessao{}, fmt.Errorf("%w: sessão %s incompleta", shared.ErrRespostaInvalidaDoCatalogo, sessaoID)
	}
	return sessao.Sessao{ID: sessaoID, Status: corpo.Status, Inicio: inicio, PrecoBase: corpo.PrecoBase}, nil
}
