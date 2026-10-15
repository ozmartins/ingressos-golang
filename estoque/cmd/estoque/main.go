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

	adaptadoramqp "github.com/oseias/ingressos-golang/estoque/internal/adapter/amqp"
	adaptadorcatalogo "github.com/oseias/ingressos-golang/estoque/internal/adapter/catalogo"
	adaptadorhttp "github.com/oseias/ingressos-golang/estoque/internal/adapter/http"
	"github.com/oseias/ingressos-golang/estoque/internal/adapter/postgres"
	adaptadorredis "github.com/oseias/ingressos-golang/estoque/internal/adapter/redis"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/config"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/health"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func main() {
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
			prazo = nil
		} else {
			defer prazo.Fechar()
		}
	} else {
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
	reservar := usecase.ReservarPoltronas{
		Sessoes: adaptadorcatalogo.NovoCliente(adaptadorcatalogo.Opcoes{
			URLBase:         cfg.CatalogoURL,
			Timeout:         cfg.CatalogoTimeout,
			FalhasParaAbrir: uint32(cfg.BreakerFalhas),
			IntervaloAberto: cfg.BreakerIntervalo,
		}),
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
	}

	postgres.RotinaPeriodica(ctx, "varredura-expiracao", cfg.VarreduraExpiracaoInterval,
		func(ctx context.Context) error {
			_, err := expirar.Varrer(ctx)
			return err
		}, aoFalhar)

	postgres.RotinaPeriodica(ctx, "limpeza-idempotencia", time.Hour,
		func(ctx context.Context) error {
			_, err := banco.LimparMensagensProcessadas(ctx, cfg.RetencaoMensagens)
			return err
		}, aoFalhar)

	if prazo != nil {
		prazo.EscutarExpiracoes(ctx, func(ctx context.Context, reservaID string) {
			_, _ = expirar.ExpirarUma(ctx, reservaID)
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

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():

		ctxDreno, cancelarDreno := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelarDreno()
		if n, err := publicador.Drenar(ctxDreno, 500); err != nil {
		} else if n > 0 {
		}
		return nil
	}
}
