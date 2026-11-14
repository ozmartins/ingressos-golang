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

	"github.com/google/uuid"

	adaptadoramqp "github.com/oseias/ingressos-golang/catalogo/internal/adapter/amqp"
	adapterhttp "github.com/oseias/ingressos-golang/catalogo/internal/adapter/http"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/config"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/health"
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

	ctx, pararSinais := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer pararSinais()

	banco, err := postgres.Abrir(ctx, cfg.DatabaseURL)

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

	erros := make(chan error, 1)

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

	return servidor.Shutdown(ctxDesligamento)
}
