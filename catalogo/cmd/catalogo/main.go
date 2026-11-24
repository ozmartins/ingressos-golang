package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	saudegrpc "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	pb "github.com/oseias/ingressos-golang/catalogo/gen/pb/catalogo/v1"
	adaptadoramqp "github.com/oseias/ingressos-golang/catalogo/internal/adapter/amqp"
	adaptadorgrpc "github.com/oseias/ingressos-golang/catalogo/internal/adapter/grpc"
	adapterhttp "github.com/oseias/ingressos-golang/catalogo/internal/adapter/http"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/config"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/health"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/obs"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/ratelimit"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/retry"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/tlsconf"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

func main() {
	// `<serviço> healthcheck <url>`: sonda usada pelo HEALTHCHECK do compose.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "uso: healthcheck <url>")
			os.Exit(2)
		}
		if err := health.SondarURL(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
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

	ctx, pararSinais := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer pararSinais()

	telemetria, err := obs.Iniciar(ctx, "catalogo")
	if err != nil {
		return fmt.Errorf("observabilidade: %w", err)
	}
	defer func() {
		ctxDescarga, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = telemetria.Desligar(ctxDescarga)
	}()
	pararMetricas := obs.ServirMetricas(obs.EnderecoMetricas())
	defer func() {
		ctxDesligar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = pararMetricas(ctxDesligar)
	}()

	// O Postgres pode ainda estar subindo: espera com backoff (~1 min) antes de desistir.
	var banco *postgres.Banco
	err = retry.Executar(ctx, "conectar ao postgres", retry.Dependencia, func() (e error) {
		banco, e = postgres.Abrir(ctx, cfg.DatabaseURL)
		return e
	})
	if err != nil {
		return err
	}

	defer banco.Fechar()

	verificador, err := identidade.NovoVerificador(cfg.KeycloakIssuerURL, cfg.KeycloakAudience)

	if err != nil {
		return err
	}

	broker, err := adaptadoramqp.Conectar(cfg.RabbitMQURL)

	if err != nil {
		return err
	}

	defer broker.Fechar()

	filmes := postgres.NovoFilmeRepository(banco)
	cinemas := postgres.NovoCinemaRepository(banco)
	salas := postgres.NovoSalaRepository(banco)
	sessoes := postgres.NovoSessaoRepository(banco)
	caixa := postgres.NovaCaixaDeSaida(banco)

	publicador := &adaptadoramqp.Publicador{
		Conexao: broker,
		Caixa:   caixa,

		Intervalo: cfg.OutboxIntervalo,
		Lote:      cfg.OutboxLote,
	}

	publicador.Iniciar(ctx)

	router := adapterhttp.NovoRouter(adapterhttp.Dependencias{
		Taxa: ratelimit.Novo(float64(cfg.RateLimitRPS), cfg.RateLimitBurst),
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
				Agora: time.Now,
			},
			AtualizarSessao: usecase.AtualizarSessao{
				Sessoes: sessoes, Filmes: filmes, Salas: salas, GerarID: uuid.NewString,
				Agora: time.Now,
			},
			RemoverSessao: usecase.RemoverSessao{
				Repo: sessoes, Agora: time.Now,
			},
			Limites: adapterhttp.LimitesPaginacao{
				Padrao: cfg.PaginacaoTamanhoPadrao,
				Maximo: cfg.PaginacaoTamanhoMaximo,
			},
		},
		Saude:       health.Handler(banco),
		Verificador: verificador,
	})

	servidor := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// gRPC: canal serviço a serviço (o estoque consulta preço e status da
	// sessão). Reutiliza o mesmo caso de uso do REST.
	// Só o estoque (service account com o papel catalogo-leitura-sessao) pode consultar.
	opcoesGRPC := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(adaptadorgrpc.InterceptadorRequestID(), adaptadorgrpc.InterceptadorAutenticacao(verificador, identidade.PermissaoLerSessao)),
	}
	if cfg.GRPCTLSCert != "" {
		tlsCfg, err := tlsconf.Servidor(cfg.GRPCTLSCert, cfg.GRPCTLSKey, cfg.GRPCTLSClientCA)
		if err != nil {
			return fmt.Errorf("gRPC TLS: %w", err)
		}
		opcoesGRPC = append(opcoesGRPC, grpc.Creds(credentials.NewTLS(tlsCfg)))
		slog.Info("gRPC com TLS", "mtls", cfg.GRPCTLSClientCA != "")
	} else {
		slog.Warn("gRPC SEM TLS (GRPC_INSECURE=true): só para desenvolvimento")
	}
	servidorGRPC := grpc.NewServer(opcoesGRPC...)
	pb.RegisterCatalogoServiceServer(servidorGRPC, &adaptadorgrpc.Servidor{
		BuscarSessao: usecase.BuscarSessao{Repo: sessoes},
	})
	saudeGRPC := saudegrpc.NewServer()
	healthpb.RegisterHealthServer(servidorGRPC, saudeGRPC)

	var escuta net.ListenConfig
	ouvinteGRPC, err := escuta.Listen(ctx, "tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("gRPC: %w", err)
	}

	erros := make(chan error, 2)

	slog.Info("catalogo no ar", "http", servidor.Addr, "grpc", cfg.GRPCAddr, "metricas", obs.EnderecoMetricas())

	go func() {
		if err := servidorGRPC.Serve(ouvinteGRPC); err != nil {
			erros <- err
		}
	}()

	go func() {
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- err
		}
	}()

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():
	}

	ctxDesligamento, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	saudeGRPC.Shutdown()
	servidorGRPC.GracefulStop()

	return servidor.Shutdown(ctxDesligamento)
}
