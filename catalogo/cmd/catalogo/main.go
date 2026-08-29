// Command catalogo é o ponto de entrada do Servico-Catalogo.
//
// Este é o único lugar onde o núcleo encontra a infraestrutura: a composição
// acontece aqui e em nenhum outro ponto (constituição, princípio I).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/estoque"
	adapterhttp "github.com/oseias/ingressos-golang/catalogo/internal/adapter/http"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/config"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/health"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

func main() {
	if err := executar(); err != nil {
		// Falhar aqui é falhar antes de atender qualquer requisição — que é
		// exatamente onde uma configuração incompleta deve doer.
		fmt.Fprintf(os.Stderr, "não foi possível iniciar o serviço: %v\n", err)
		os.Exit(1)
	}
}

func executar() error {
	cfg, err := config.Carregar()
	if err != nil {
		return err
	}

	logger := observability.ConfigurarLogger(cfg.LogLevel)

	ctx, pararSinais := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer pararSinais()

	metricas, encerrarTelemetria, err := observability.Iniciar(ctx, cfg.OTLPEndpoint)
	if err != nil {
		return fmt.Errorf("iniciando telemetria: %w", err)
	}
	defer func() {
		ctxEncerramento, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := encerrarTelemetria(ctxEncerramento); err != nil {
			logger.Error("falha ao encerrar telemetria", slog.Any("erro", err))
		}
	}()

	pool, err := postgres.NovoPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	verificador, err := identidade.NovoVerificador(ctx, cfg.KeycloakIssuerURL, cfg.KeycloakAudience)
	if err != nil {
		return err
	}

	clienteEstoque, err := estoque.NovoCliente(estoque.Opcoes{
		Endereco:        cfg.EstoqueGRPCAddr,
		Timeout:         cfg.EstoqueTimeout,
		FalhasParaAbrir: cfg.BreakerFalhasConsecutivas,
		IntervaloAberto: cfg.BreakerIntervaloAberto,
		Metricas:        metricas,
	})
	if err != nil {
		return err
	}
	defer func() { _ = clienteEstoque.Fechar() }()

	filmes := postgres.NovoFilmeRepository(pool)
	cinemas := postgres.NovoCinemaRepository(pool)
	salas := postgres.NovoSalaRepository(pool)
	sessoes := postgres.NovoSessaoRepository(pool)

	router := adapterhttp.NovoRouter(adapterhttp.Dependencias{
		Handlers: adapterhttp.Handlers{
			ListarFilmes:     usecase.ListarFilmes{Repo: filmes},
			ListarCinemas:    usecase.ListarCinemas{Repo: cinemas},
			ListarSalas:      usecase.ListarSalas{Cinemas: cinemas, Salas: salas},
			ConsultarSessoes: usecase.ConsultarSessoes{Repo: sessoes},
			ReservarPoltronas: usecase.ReservarPoltronas{
				Sessoes: sessoes,
				Estoque: clienteEstoque,
				Agora:   time.Now,
			},
			Limites: adapterhttp.LimitesPaginacao{
				Padrao: cfg.PaginacaoTamanhoPadrao,
				Maximo: cfg.PaginacaoTamanhoMaximo,
			},
		},
		Saude:       health.Handler(pool),
		Verificador: verificador,
		Metricas:    metricas,
	})

	servidor := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	erros := make(chan error, 1)
	go func() {
		logger.Info("servidor iniciado", slog.String("porta", cfg.HTTPPort))
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- err
		}
	}()

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():
		logger.Info("encerrando por sinal do sistema")
	}

	ctxDesligamento, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return servidor.Shutdown(ctxDesligamento)
}
