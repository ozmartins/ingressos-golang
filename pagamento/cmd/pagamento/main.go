package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	amqp091 "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/pagamento/internal/adapter/adquirente/simulado"
	adaptamqp "github.com/oseias/ingressos-golang/pagamento/internal/adapter/amqp"
	adapthttp "github.com/oseias/ingressos-golang/pagamento/internal/adapter/http"
	"github.com/oseias/ingressos-golang/pagamento/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/config"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/health"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/observability"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

type relogio struct{}

func (relogio) Agora() time.Time { return time.Now().UTC() }

type ids struct{}

func (ids) Novo() string { return uuid.NewString() }

func main() {
	if err := executar(); err != nil {
		os.Stderr.WriteString("pagamento: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func executar() error {
	cfg, err := config.Carregar()
	if err != nil {
		return err
	}

	log := observability.Logger(cfg.NivelLog)
	propagador := observability.Propagador()

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	pool, err := postgres.Abrir(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}
	repo := postgres.NovoRepositorio(pool)

	conexao, err := amqp091.Dial(cfg.AMQPURL)
	if err != nil {
		return err
	}
	defer conexao.Close()

	canalTopologia, err := conexao.Channel()
	if err != nil {
		return err
	}
	topologia := adaptamqp.Topologia{
		Exchange: cfg.AMQPExchange, ExchangeDLX: cfg.AMQPExchangeDLX,
		Fila: cfg.AMQPFilaReserva, FilaDLQ: cfg.AMQPFilaReservaDLQ,
		Binding: "reserva.criada", LimiteEntregas: cfg.AMQPLimiteEntregas,
	}
	if err := topologia.Declarar(canalTopologia); err != nil {
		return err
	}
	_ = canalTopologia.Close()

	canalPub, err := conexao.Channel()
	if err != nil {
		return err
	}
	publicador, err := adaptamqp.NovoPublicador(canalPub, cfg.AMQPExchange, propagador)
	if err != nil {
		return err
	}

	canalCons, err := conexao.Channel()
	if err != nil {
		return err
	}

	processar := usecase.ProcessarPagamento{
		Repo:       repo,
		Adquirente: simulado.Adquirente{Demora: cfg.AdquirenteTimeout * 3},
		Publicador: publicador,
		Relogio:    relogio{},
		IDs:        ids{},

		PrazoAdquirente: cfg.AdquirenteTimeout,
	}

	consumidor := &adaptamqp.Consumidor{
		Canal: canalCons, Fila: cfg.AMQPFilaReserva, Prefetch: cfg.AMQPPrefetch,
		Caso: processar, Log: log, Propagador: propagador,
		EmAndamento: &adaptamqp.Medidor{},
	}

	auth, err := adapthttp.NovoAutenticador(cfg.JWKSURL, cfg.JWTIssuer, cfg.JWTAud)
	if err != nil {
		return err
	}

	prontidao := health.NovaProntidao()
	prontidao.Registrar("banco", repo.Ping)
	prontidao.Registrar("canal-de-eventos", func(context.Context) error {
		if conexao.IsClosed() {
			return errors.New("conexão AMQP fechada")
		}
		return nil
	})

	api := &adapthttp.API{
		Consulta:  usecase.ConsultarPagamento{Repo: repo},
		Auth:      auth,
		Prontidao: prontidao,
		Log:       log,
	}

	servidor := &http.Server{
		Addr:              ":" + cfg.PortaHTTP,
		Handler:           api.Rotas(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	erros := make(chan error, 2)
	go func() {
		log.Info("servidor HTTP no ar", "porta", cfg.PortaHTTP)
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- err
		}
	}()
	go func() {
		log.Info("consumindo intenções", "fila", cfg.AMQPFilaReserva, "prefetch", cfg.AMQPPrefetch)
		if err := consumidor.Consumir(ctx); err != nil && !errors.Is(err, context.Canceled) {
			erros <- err
		}
	}()

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():
		log.Info("desligando")
		desligar, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		return servidor.Shutdown(desligar)
	}
}
