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

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	adaptadoramqp "github.com/oseias/ingressos-golang/catalogo/internal/adapter/amqp"
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
		CAFile:          cfg.EstoqueTLSCAFile,
		CertFile:        cfg.EstoqueTLSCertFile,
		KeyFile:         cfg.EstoqueTLSKeyFile,
	})

	if err != nil {
		return err
	}

	defer func() { _ = clienteEstoque.Fechar() }()

	broker, err := adaptadoramqp.Conectar(cfg.RabbitMQURL)

	if err != nil {
		return err
	}

	defer broker.Fechar()

	filmes := postgres.NovoFilmeRepository(pool)
	cinemas := postgres.NovoCinemaRepository(pool)
	salas := postgres.NovoSalaRepository(pool)
	sessoes := postgres.NovoSessaoRepository(pool)
	caixa := postgres.NovaCaixaDeSaida(pool)

	// A caixa é drenada fora do caminho da requisição: criar uma sessão não
	// espera pelo broker, e o fato sai quando ele estiver de pé.
	publicador := &adaptadoramqp.Publicador{
		Conexao:   broker,
		Caixa:     caixa,
		Log:       logger,
		Intervalo: cfg.OutboxIntervalo,
		Lote:      cfg.OutboxLote,
	}
	publicador.Iniciar(ctx)

	router := adapterhttp.NovoRouter(adapterhttp.Dependencias{
		Handlers: adapterhttp.Handlers{
			ListarFilmes:     usecase.ListarFilmes{Repo: filmes},
			BuscarFilme:      usecase.BuscarFilme{Repo: filmes},
			CriarFilme:       usecase.CriarFilme{Repo: filmes, GerarID: uuid.NewString},
			AtualizarFilme:   usecase.AtualizarFilme{Repo: filmes},
			RemoverFilme:     usecase.RemoverFilme{Repo: filmes},
			ListarCinemas:    usecase.ListarCinemas{Repo: cinemas},
			BuscarCinema:     usecase.BuscarCinema{Repo: cinemas},
			CriarCinema:      usecase.CriarCinema{Repo: cinemas, GerarID: uuid.NewString},
			AtualizarCinema:  usecase.AtualizarCinema{Repo: cinemas},
			RemoverCinema:    usecase.RemoverCinema{Repo: cinemas},
			ListarSalas:      usecase.ListarSalas{Cinemas: cinemas, Salas: salas},
			BuscarSala:       usecase.BuscarSala{Salas: salas},
			CriarSala:        usecase.CriarSala{Cinemas: cinemas, Salas: salas, GerarID: uuid.NewString},
			AtualizarSala:    usecase.AtualizarSala{Cinemas: cinemas, Salas: salas},
			RemoverSala:      usecase.RemoverSala{Salas: salas},
			ConsultarSessoes: usecase.ConsultarSessoes{Repo: sessoes},
			BuscarSessao:     usecase.BuscarSessao{Repo: sessoes},
			CriarSessao: usecase.CriarSessao{
				Sessoes: sessoes, Filmes: filmes, Salas: salas, GerarID: uuid.NewString,
				Agora: time.Now, TraceContextDe: contextoDeRastreamento,
			},
			AtualizarSessao: usecase.AtualizarSessao{Sessoes: sessoes, Filmes: filmes, Salas: salas},
			RemoverSessao:   usecase.RemoverSessao{Repo: sessoes},
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

// O contexto W3C da requisição, capturado para viajar nos cabeçalhos do fato. O
// publicador roda fora da requisição, e sem isso o span de quem consome nasceria
// órfão.
func contextoDeRastreamento(ctx context.Context) map[string]string {
	portador := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, portador)
	if len(portador) == 0 {
		return nil
	}
	return portador
}
