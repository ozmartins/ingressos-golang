// Command notificacao emite ingressos digitais a partir de pagamentos
// confirmados e os disponibiliza para a pessoa e para a portaria do cinema.
package main

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/oseias/ingressos-golang/notificacao/internal/platform/observability"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	if err := executar(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func executar() error {
	// Configuração primeiro: variável ausente ou malformada impede subir, e o
	// erro lista todas de uma vez (research.md D11).
	cfg, err := config.Carregar()
	if err != nil {
		return err
	}
	log := observability.Logger(cfg.NivelLog)
	propagador := observability.Propagador()

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	assinador, err := codigo.NovoAssinador(cfg.IngressoQRSegredo)
	if err != nil {
		return err
	}

	partida, cancelar := context.WithTimeout(ctx, 30*time.Second)
	defer cancelar()

	pool, err := postgres.Conectar(partida, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	conexao, err := amqp.Dial(cfg.AMQPURL)
	if err != nil {
		return fmt.Errorf("conectar ao broker: %w", err)
	}
	defer conexao.Close()

	canal, err := conexao.Channel()
	if err != nil {
		return fmt.Errorf("abrir canal: %w", err)
	}
	defer canal.Close()

	// Consumir de fila sem fila morta é perder mensagem em silêncio: se a
	// topologia não puder ser garantida, o processo não sobe.
	topologia := adaptamqp.Topologia{
		Exchange: cfg.AMQPExchange, ExchangeDLX: cfg.AMQPExchangeDLX,
		Fila: cfg.AMQPFila, FilaDLQ: cfg.AMQPFilaDLQ,
		Binding: "pagamento.sucesso", LimiteEntregas: cfg.AMQPLimiteEntregas,
	}
	if err := topologia.Declarar(canal); err != nil {
		return err
	}

	ingressos := postgres.Ingressos{Pool: pool}
	emitir := usecase.EmitirIngresso{
		Ingressos:   ingressos,
		Avisos:      postgres.Avisos{Pool: pool},
		Notificador: simulado.Notificador{Falhar: cfg.NotificadorModo == config.NotificarFalhar, Log: log},
		Assinador:   assinador,
		Relogio:     sistema.Relogio{},
		IDs:         sistema.GeradorID{},
		Log:         log,
	}

	auth, err := adapthttp.NovoAutenticador(cfg.JWKSURL, cfg.JWTIssuer, cfg.JWTAud)
	if err != nil {
		return err
	}
	chave, err := adapthttp.NovaChavePortaria(cfg.PortariaAPIKey)
	if err != nil {
		return err
	}

	prontidao := health.NovaProntidao()
	prontidao.Registrar("postgres", func(c context.Context) error { return pool.Ping(c) })
	prontidao.Registrar("rabbitmq", func(context.Context) error {
		if conexao.IsClosed() {
			return errors.New("conexão com o broker fechada")
		}
		return nil
	})

	api := &adapthttp.API{
		Listagem:  usecase.ListarIngressos{Ingressos: ingressos},
		Validacao: usecase.ValidarIngresso{Ingressos: ingressos, Assinador: assinador, Relogio: sistema.Relogio{}, Log: log},
		Auth:      auth, Chave: chave, Prontidao: prontidao, Log: log,
	}

	servidor := &http.Server{
		Addr:              ":" + cfg.PortaHTTP,
		Handler:           api.Rotas(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	consumidor := &adaptamqp.Consumidor{
		Canal: canal, Fila: cfg.AMQPFila, Prefetch: cfg.AMQPPrefetch,
		Caso: emitir, Log: log, Propagador: propagador,
	}

	erros := make(chan error, 2)
	go func() {
		log.Info("servidor no ar", "porta", cfg.PortaHTTP)
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- fmt.Errorf("servidor http: %w", err)
		}
	}()
	go func() {
		log.Info("consumindo anúncios de pagamento", "fila", cfg.AMQPFila, "prefetch", cfg.AMQPPrefetch)
		if err := consumidor.Consumir(ctx); err != nil && !errors.Is(err, context.Canceled) {
			erros <- fmt.Errorf("consumidor amqp: %w", err)
		}
	}()

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():
		log.Info("desligando")
	}

	desligar, cancelarDesligamento := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelarDesligamento()
	return servidor.Shutdown(desligar)
}
