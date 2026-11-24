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
	amqp091 "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/pagamento/internal/adapter/adquirente/simulado"
	adaptamqp "github.com/oseias/ingressos-golang/pagamento/internal/adapter/amqp"
	adapthttp "github.com/oseias/ingressos-golang/pagamento/internal/adapter/http"
	"github.com/oseias/ingressos-golang/pagamento/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/config"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/health"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/obs"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/ratelimit"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/retry"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

type relogio struct{}

func (relogio) Agora() time.Time { return time.Now().UTC() }

type ids struct{}

func (ids) Novo() string { return uuid.NewString() }

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
		_, _ = os.Stderr.WriteString("pagamento: " + err.Error() + "\n")
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

	telemetria, err := obs.Iniciar(ctx, "pagamento")
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
		tentativa, cancelarTentativa := context.WithTimeout(ctx, 10*time.Second)
		defer cancelarTentativa()
		banco, e = postgres.Conectar(tentativa, cfg.DatabaseURL)
		return e
	})
	if err != nil {
		return err
	}
	defer banco.Fechar()
	repo := postgres.NovoRepositorio(banco.DB())

	var conexao *amqp091.Connection
	err = retry.Executar(ctx, "conectar ao broker", retry.Dependencia, func() (e error) {
		conexao, e = amqp091.Dial(cfg.AMQPURL)
		return e
	})
	if err != nil {
		return err
	}
	defer func() { _ = conexao.Close() }()

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
	publicador, err := adaptamqp.NovoPublicador(canalPub, cfg.AMQPExchange)
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

	registrar := usecase.RegistrarIntencao{Repo: repo, Relogio: relogio{}, IDs: ids{}}

	consumidor := &adaptamqp.Consumidor{
		Canal: canalCons, Fila: cfg.AMQPFilaReserva, Prefetch: cfg.AMQPPrefetch,
		Caso:        registrar,
		EmAndamento: &adaptamqp.Medidor{},
	}

	varrer := usecase.VarrerCobrancas{
		Repo: repo, Cobranca: processar, Relogio: relogio{},
		Lote: cfg.VarreduraLote, ComRequestID: obs.ComRequestID,
	}
	postgres.RotinaPeriodica(ctx, "varredura-cobrancas", cfg.VarreduraIntervalo,
		varrer.Executar, func(nome string, err error) {
			slog.Error("rotina periódica falhou", "rotina", nome, "erro", err)
		})

	auth, err := adapthttp.NovoAutenticador(cfg.JWKSURL, cfg.JWTIssuer, cfg.JWTAud)
	if err != nil {
		return err
	}

	prontidao := health.NovaProntidao()
	prontidao.Registrar("banco", banco.Verificar)
	prontidao.Registrar("canal-de-eventos", func(context.Context) error {
		if conexao.IsClosed() {
			return errors.New("conexão AMQP fechada")
		}
		return nil
	})

	api := &adapthttp.API{
		Consulta:  usecase.ConsultarPagamento{Repo: repo},
		Escolha:   usecase.EscolherForma{Repo: repo, Relogio: relogio{}},
		Auth:      auth,
		Prontidao: prontidao,
		Taxa:      ratelimit.Novo(float64(cfg.RateLimitRPS), cfg.RateLimitBurst),
	}

	servidor := &http.Server{
		Addr:              ":" + cfg.PortaHTTP,
		Handler:           api.Rotas(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	erros := make(chan error, 2)
	slog.Info("pagamento no ar", "http", servidor.Addr, "metricas", obs.EnderecoMetricas())
	go func() {
		if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- err
		}
	}()
	go func() {
		if err := consumidor.Consumir(ctx); err != nil && !errors.Is(err, context.Canceled) {
			erros <- err
		}
	}()

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():

		desligar, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		return servidor.Shutdown(desligar)
	}
}
