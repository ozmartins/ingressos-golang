// Command estoque é o binário do Servico-Estoque.
//
// Este é o único ponto de composição do serviço (constituição, princípio I):
// aqui a configuração vira adaptadores, os adaptadores viram casos de uso e os
// casos de uso são ligados ao servidor gRPC, aos consumidores e às rotinas de
// fundo. Nenhuma outra parte do código conhece essa montagem.
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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	adaptadoramqp "github.com/oseias/ingressos-golang/estoque/internal/adapter/amqp"
	adaptadorgrpc "github.com/oseias/ingressos-golang/estoque/internal/adapter/grpc"
	adaptadorhttp "github.com/oseias/ingressos-golang/estoque/internal/adapter/http"
	"github.com/oseias/ingressos-golang/estoque/internal/adapter/postgres"
	adaptadorredis "github.com/oseias/ingressos-golang/estoque/internal/adapter/redis"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/config"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/health"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func main() {
	if err := executar(); err != nil {
		fmt.Fprintf(os.Stderr, "estoque não pôde subir: %v\n", err)
		os.Exit(1)
	}
}

func executar() error {
	// 1. Configuração: lida e validada uma vez. Falha aqui aborta a largada.
	cfg, err := config.Carregar()
	if err != nil {
		return err
	}

	ctx, encerrar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer encerrar()

	// 2. Observabilidade.
	obs, err := observability.Iniciar(ctx, cfg.LogLevel, cfg.OTLPEndpoint)
	if err != nil {
		return fmt.Errorf("iniciar observabilidade: %w", err)
	}
	defer func() {
		ctxDesligar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		obs.Desligar(ctxDesligar)
	}()

	// 3. Adaptadores de saída.
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
		prazo, err = adaptadorredis.Abrir(ctx, cfg.RedisURL, obs)
		if err != nil {
			// O índice de prazo é conveniência, não correção: sem ele a
			// varredura continua liberando as reservas (research D2/D4).
			obs.Log.Warn("índice de prazo indisponível na largada",
				"erro", err.Error(), "efeito", "expiração fica só com a varredura")
			prazo = nil
		} else {
			defer prazo.Fechar()
		}
	} else {
		obs.Log.Warn(config.ErrRedisAusente.Error())
	}

	reservas := postgres.NovoRepositorioReservas(banco)
	poltronas := postgres.NovoRepositorioPoltronas(banco)
	relogio := shared.RelogioReal{}

	var indiceDePrazo usecase.IndiceDePrazo
	if prazo != nil {
		indiceDePrazo = prazo
	}

	// 4. Casos de uso.
	bloquear := usecase.BloquearPoltronas{
		Reservas:       reservas,
		Prazo:          indiceDePrazo,
		Relogio:        relogio,
		Log:            obs.Log,
		TTL:            cfg.ReservaTTL,
		Limite:         cfg.PoltronasMaxPorBloqueio,
		TraceContextDe: contextoDeRastreamento,
	}
	consultar := usecase.ConsultarMapa{Poltronas: poltronas}
	confirmar := usecase.ConfirmarReserva{Reservas: reservas, Prazo: indiceDePrazo, Relogio: relogio, Log: obs.Log}
	cancelar := usecase.CancelarReserva{Reservas: reservas, Prazo: indiceDePrazo, Relogio: relogio, Log: obs.Log}
	expirar := usecase.ExpirarReservas{Reservas: reservas, Prazo: indiceDePrazo, Relogio: relogio, Log: obs.Log}
	provisionar := usecase.ProvisionarSessao{Poltronas: poltronas, Log: obs.Log}

	// 5. Consumidores.
	if err := adaptadoramqp.ConsumirPagamentoSucesso(ctx, broker, cfg.AMQPPrefetch, obs, confirmar); err != nil {
		return fmt.Errorf("consumidor de pagamento aprovado: %w", err)
	}
	if err := adaptadoramqp.ConsumirPagamentoFalhou(ctx, broker, cfg.AMQPPrefetch, obs, cancelar); err != nil {
		return fmt.Errorf("consumidor de pagamento recusado: %w", err)
	}
	if err := adaptadoramqp.ConsumirSessaoCriada(ctx, broker, cfg.AMQPPrefetch, obs, provisionar); err != nil {
		return fmt.Errorf("consumidor de sessão criada: %w", err)
	}

	// 6. Rotinas de fundo.
	publicador := &adaptadoramqp.Publicador{Conexao: broker, Banco: banco, Obs: obs, Intervalo: time.Second}
	publicador.Iniciar(ctx)

	aoFalhar := func(nome string, err error) {
		obs.Log.Warn("rotina periódica falhou", "rotina", nome, "erro", err.Error())
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
			if _, err := expirar.ExpirarUma(ctx, reservaID); err != nil {
				obs.Log.Warn("falha ao expirar pelo índice de prazo",
					"reserva_id", reservaID, "erro", err.Error())
			}
		})
	}

	// 7. Porta de administração (saúde e métricas).
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
		if err := admin.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			obs.Log.Error("porta de administração caiu", "erro", err.Error())
		}
	}()
	defer func() {
		ctxDesligar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = admin.Shutdown(ctxDesligar)
	}()

	// 8. API REST: as mesmas operações do gRPC, para o cliente final. Fica em
	// porta própria porque a de administração é outra audiência — operação, não
	// negócio — e porque o gRPC exige mTLS, que o navegador não apresenta.
	autenticador, err := adaptadorhttp.NovoAutenticador(cfg.JWKSURL, cfg.JWTIssuer, cfg.JWTAudience)
	if err != nil {
		return fmt.Errorf("autenticador da API REST: %w", err)
	}
	apiREST := &adaptadorhttp.API{
		Bloqueio: bloquear,
		Mapa:     consultar,
		Auth:     autenticador,
		Limite:   cfg.PoltronasMaxPorBloqueio,
	}
	rest := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apiREST.Rotas(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := rest.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			obs.Log.Error("porta da API REST caiu", "erro", err.Error())
		}
	}()
	defer func() {
		ctxDesligar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = rest.Shutdown(ctxDesligar)
	}()

	// 9. Canal síncrono.
	servidor, err := adaptadorgrpc.NovoServidor(adaptadorgrpc.Opcoes{
		Bloqueio: bloquear, Mapa: consultar, Obs: obs, Config: cfg,
	})
	if err != nil {
		return err
	}

	erros := make(chan error, 1)
	go func() { erros <- servidor.Servir(cfg.GRPCAddr) }()

	obs.Log.Info("estoque no ar",
		"grpc", cfg.GRPCAddr, "admin", cfg.AdminAddr, "rest", cfg.HTTPAddr,
		"reserva_ttl", cfg.ReservaTTL.String(),
		"limite_poltronas", cfg.PoltronasMaxPorBloqueio,
		"mtls", string(cfg.TLSClientAuth))

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():
		obs.Log.Info("encerrando")
		servidor.Encerrar()
		// Última tentativa de drenar a caixa de saída antes de sair.
		ctxDreno, cancelarDreno := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelarDreno()
		if n, err := publicador.Drenar(ctxDreno, 500); err != nil {
			obs.Log.Warn("caixa de saída não drenada no encerramento", "erro", err.Error())
		} else if n > 0 {
			obs.Log.Info("fatos publicados no encerramento", "quantidade", n)
		}
		return nil
	}
}

// contextoDeRastreamento serializa o contexto corrente para viajar com o fato
// até a publicação, que acontece fora desta requisição (FR-044, SC-009).
func contextoDeRastreamento(ctx context.Context) map[string]string {
	portador := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, portador)
	if len(portador) == 0 {
		return nil
	}
	return portador
}
