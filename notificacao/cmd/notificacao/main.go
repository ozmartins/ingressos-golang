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

	adaptamqp "github.com/oseias/ingressos-golang/notificacao/internal/adapter/amqp"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/codigo"
	adapthttp "github.com/oseias/ingressos-golang/notificacao/internal/adapter/http"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/notificador/simulado"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/sistema"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/config"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/health"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/obs"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/ratelimit"
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/retry"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func executar() error {
	cfg, err := config.Carregar()
	if err != nil {
		return err
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	telemetria, err := obs.Iniciar(ctx, "notificacao")
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

	assinador, err := codigo.NovoAssinador(cfg.IngressoQRSegredo)
	if err != nil {
		return err
	}

	partida, cancelar := context.WithTimeout(ctx, 30*time.Second)
	defer cancelar()

	banco, err := postgres.Conectar(partida, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer banco.Fechar()

	var conexao *amqp.Connection
	err = retry.Executar(ctx, "conectar ao broker", retry.Dependencia, func() (e error) {
		conexao, e = amqp.Dial(cfg.AMQPURL)
		return e
	})
	if err != nil {
		return fmt.Errorf("conectar ao broker: %w", err)
	}
	defer func() { _ = conexao.Close() }()

	canal, err := conexao.Channel()
	if err != nil {
		return fmt.Errorf("abrir canal: %w", err)
	}
	defer func() { _ = canal.Close() }()

	topologia := adaptamqp.Topologia{
		Exchange: cfg.AMQPExchange, ExchangeDLX: cfg.AMQPExchangeDLX,
		Fila: cfg.AMQPFila, FilaDLQ: cfg.AMQPFilaDLQ,
		Binding: "pagamento.sucesso", LimiteEntregas: cfg.AMQPLimiteEntregas,
	}
	if err := topologia.Declarar(canal); err != nil {
		return err
	}

	ingressos := postgres.Ingressos{DB: banco.DB()}
	emitir := usecase.EmitirIngresso{
		Ingressos:   ingressos,
		Avisos:      postgres.Avisos{DB: banco.DB()},
		Notificador: simulado.Notificador{Falhar: cfg.NotificadorModo == config.NotificarFalhar},
		PrazoAviso:  cfg.NotificadorTimeout,
		Assinador:   assinador,
		Relogio:     sistema.Relogio{},
		IDs:         sistema.GeradorID{},
	}

	auth, err := adapthttp.NovoAutenticador(cfg.JWKSURL, cfg.JWTIssuer, cfg.JWTAud)
	if err != nil {
		return err
	}

	prontidao := health.NovaProntidao()
	prontidao.Registrar("postgres", banco.Verificar)
	prontidao.Registrar("rabbitmq", func(context.Context) error {
		if conexao.IsClosed() {
			return errors.New("conexão com o broker fechada")
		}
		return nil
	})

	api := &adapthttp.API{
		Listagem:  usecase.ListarIngressos{Ingressos: ingressos},
		Validacao: usecase.ValidarIngresso{Ingressos: ingressos, Assinador: assinador, Relogio: sistema.Relogio{}},
		Auth:      auth, Prontidao: prontidao,
		TaxaValidacao: ratelimit.Novo(float64(cfg.RateLimitRPS), cfg.RateLimitBurst),
	}

	servidor := &http.Server{
		Addr:              ":" + cfg.PortaHTTP,
		Handler:           api.Rotas(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	consumidor := &adaptamqp.Consumidor{
		Canal: canal, Fila: cfg.AMQPFila, Prefetch: cfg.AMQPPrefetch,
		Caso: emitir,
	}

	erros := make(chan error, 2)
	slog.Info("notificacao no ar", "http", servidor.Addr, "metricas", obs.EnderecoMetricas())
	go func() {
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- fmt.Errorf("servidor http: %w", err)
		}
	}()
	go func() {
		if err := consumidor.Consumir(ctx); err != nil && !errors.Is(err, context.Canceled) {
			erros <- fmt.Errorf("consumidor amqp: %w", err)
		}
	}()

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():
	}

	desligar, cancelarDesligamento := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelarDesligamento()
	return servidor.Shutdown(desligar)
}
