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

	"go.opentelemetry.io/otel"

	adaptadoramqp "github.com/oseias/ingressos-golang/estoque/internal/adapter/amqp"
	adaptadorcatalogo "github.com/oseias/ingressos-golang/estoque/internal/adapter/catalogo"
	adaptadorhttp "github.com/oseias/ingressos-golang/estoque/internal/adapter/http"
	"github.com/oseias/ingressos-golang/estoque/internal/adapter/postgres"
	adaptadorredis "github.com/oseias/ingressos-golang/estoque/internal/adapter/redis"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/config"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/health"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/obs"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/ratelimit"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
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
		fmt.Fprintf(os.Stderr, "estoque não pôde subir: %v\n", err)
		os.Exit(1)
	}
}

func executar() error {
	cfg, err := config.Carregar()
	if err != nil {
		return err
	}

	ctx, encerrar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer encerrar()

	telemetria, err := obs.Iniciar(ctx, "estoque")
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

	banco, err := postgres.Abrir(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer banco.Fechar()

	broker, err := adaptadoramqp.Conectar(cfg.RabbitMQURL)
	if err != nil {
		return err
	}
	defer broker.Fechar()

	var prazo *adaptadorredis.Indice
	if cfg.RedisURL != "" {
		prazo, err = adaptadorredis.Abrir(ctx, cfg.RedisURL)
		if err != nil {
			slog.Warn("redis indisponível; a expiração dependerá só da varredura", "erro", err)
			prazo = nil
		} else {
			defer prazo.Fechar()
		}
	} else {
		slog.Warn("REDIS_URL vazio; a expiração dependerá só da varredura")
	}

	reservas := postgres.NovoRepositorioReservas(banco)
	poltronas := postgres.NovoRepositorioPoltronas(banco)
	relogio := shared.RelogioReal{}

	var indiceDePrazo usecase.IndiceDePrazo
	if prazo != nil {
		indiceDePrazo = prazo
	}

	bloquear := usecase.BloquearPoltronas{
		Reservas: reservas,
		Prazo:    indiceDePrazo,
		Relogio:  relogio,

		TTL:    cfg.ReservaTTL,
		Limite: cfg.PoltronasMaxPorBloqueio,
	}
	catalogo, err := adaptadorcatalogo.NovoCliente(adaptadorcatalogo.Opcoes{
		Endereco:        cfg.CatalogoGRPCAddr,
		Timeout:         cfg.CatalogoTimeout,
		FalhasParaAbrir: uint32(cfg.BreakerFalhas),
		IntervaloAberto: cfg.BreakerIntervalo,
		Tentativas:      cfg.CatalogoTentativas,
		EsperaBase:      cfg.CatalogoEsperaBase,
		TokenURL:        cfg.KeycloakTokenURL,
		ClientID:        cfg.CatalogoClientID,
		ClientSecret:    cfg.CatalogoClientSegredo,
		TokenTimeout:    cfg.CatalogoTimeout,
		CacheTTL:        cfg.CatalogoCacheTTL,
	})
	if err != nil {
		return err
	}
	defer func() { _ = catalogo.Fechar() }()

	reservar := usecase.ReservarPoltronas{
		Sessoes:  catalogo,
		Bloqueio: bloquear,
		Relogio:  relogio,
	}
	consultar := usecase.ConsultarMapa{Poltronas: poltronas}
	confirmar := usecase.ConfirmarReserva{Reservas: reservas, Prazo: indiceDePrazo, Relogio: relogio}
	cancelar := usecase.CancelarReserva{Reservas: reservas, Prazo: indiceDePrazo, Relogio: relogio}
	expirar := usecase.ExpirarReservas{Reservas: reservas, Prazo: indiceDePrazo, Relogio: relogio}
	provisionar := usecase.ProvisionarSessao{Poltronas: poltronas}
	cancelarSessao := usecase.CancelarSessao{
		Reservas: reservas, Prazo: indiceDePrazo, Relogio: relogio,
	}

	if err := adaptadoramqp.ConsumirPagamentoSucesso(ctx, broker, cfg.AMQPPrefetch, confirmar); err != nil {
		return fmt.Errorf("consumidor de pagamento aprovado: %w", err)
	}
	if err := adaptadoramqp.ConsumirPagamentoFalhou(ctx, broker, cfg.AMQPPrefetch, cancelar); err != nil {
		return fmt.Errorf("consumidor de pagamento recusado: %w", err)
	}
	if err := adaptadoramqp.ConsumirSessaoCriada(ctx, broker, cfg.AMQPPrefetch, provisionar); err != nil {
		return fmt.Errorf("consumidor de sessão criada: %w", err)
	}
	if err := adaptadoramqp.ConsumirSessaoCancelada(ctx, broker, cfg.AMQPPrefetch, cancelarSessao); err != nil {
		return fmt.Errorf("consumidor de sessão cancelada: %w", err)
	}

	publicador := &adaptadoramqp.Publicador{Conexao: broker, Banco: banco, Intervalo: time.Second}
	publicador.Iniciar(ctx)

	aoFalhar := func(nome string, err error) {
		slog.Error("rotina periódica falhou", "rotina", nome, "erro", err)
	}

	postgres.RotinaPeriodica(ctx, "varredura-expiracao", cfg.VarreduraExpiracaoInterval,
		func(ctx context.Context) error {
			n, err := expirar.Varrer(ctx)
			if n > 0 {
				obs.ReservasExpiradas(n)
				slog.InfoContext(ctx, "reservas expiradas pela varredura", "quantidade", n)
			}
			return err
		}, aoFalhar)

	postgres.RotinaPeriodica(ctx, "limpeza-idempotencia", time.Hour,
		func(ctx context.Context) error {
			_, err := banco.LimparMensagensProcessadas(ctx, cfg.RetencaoMensagens)
			return err
		}, aoFalhar)

	if prazo != nil {
		prazo.EscutarExpiracoes(ctx, func(ctx context.Context, reservaID string) {
			ctx, span := otel.Tracer("estoque").Start(ctx, "reserva.expirar")
			defer span.End()
			res, err := expirar.ExpirarUma(ctx, reservaID)
			obs.FalharSpan(span, err)
			switch {
			case err != nil:
				slog.ErrorContext(ctx, "falha ao expirar reserva", "reserva_id", reservaID, "erro", err)
			case res == usecase.TransicaoAplicada:
				obs.ReservasExpiradas(1)
				slog.InfoContext(ctx, "reserva expirada", "reserva_id", reservaID)
			}
		})
	}

	verificacoes := []health.Verificacao{
		{Nome: "postgres", Essencial: true, Checar: banco.Verificar},
		{Nome: "rabbitmq", Essencial: true, Checar: func(context.Context) error { return broker.Verificar() }},
	}
	if prazo != nil {
		verificacoes = append(verificacoes, health.Verificacao{Nome: "redis", Essencial: false, Checar: prazo.Verificar})
	}
	admin := &http.Server{
		Addr:              cfg.AdminAddr,
		Handler:           health.Novo(verificacoes...).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = admin.ListenAndServe()
	}()
	defer func() {
		ctxDesligar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = admin.Shutdown(ctxDesligar)
	}()

	autenticador, err := adaptadorhttp.NovoAutenticador(cfg.JWKSURL, cfg.JWTIssuer, cfg.JWTAudience)
	if err != nil {
		return fmt.Errorf("autenticador da API REST: %w", err)
	}
	apiREST := &adaptadorhttp.API{
		Bloqueio: reservar,
		Mapa:     consultar,
		Auth:     autenticador,
		Limite:   cfg.PoltronasMaxPorBloqueio,
		Taxa:     ratelimit.Novo(float64(cfg.RateLimitRPS), cfg.RateLimitBurst),
	}
	rest := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apiREST.Rotas(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	defer func() {
		ctxDesligar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = rest.Shutdown(ctxDesligar)
	}()

	erros := make(chan error, 1)
	go func() {
		if err := rest.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- err
		}
	}()
	slog.Info("estoque no ar", "http", cfg.HTTPAddr, "admin", cfg.AdminAddr, "metricas", obs.EnderecoMetricas())

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():

		ctxDreno, cancelarDreno := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelarDreno()
		if n, err := publicador.Drenar(ctxDreno, 500); err != nil {
			slog.Error("falha ao drenar a caixa de saída no desligamento", "erro", err)
		} else if n > 0 {
			slog.Info("caixa de saída drenada no desligamento", "publicados", n)
		}
		return nil
	}
}
