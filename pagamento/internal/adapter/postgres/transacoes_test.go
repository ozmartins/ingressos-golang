package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

var errConexao = errors.New("conexão recusada")

func TestBuscarPorReservaDistingueNaoEncontradoDeInfra(t *testing.T) {
	ctx := context.Background()

	t.Run("sem linhas é não encontrado", func(t *testing.T) {
		repo := NovoRepositorio(abrirBancoFalso(t, comportamento{}))
		_, err := repo.BuscarPorReserva(ctx, "r1")
		if !errors.Is(err, usecase.ErrNaoEncontrada) {
			t.Fatalf("queria ErrNaoEncontrada, veio %v", err)
		}
		if errors.Is(err, usecase.ErrDependenciaIndisponivel) {
			t.Fatal("não encontrado não pode ser tratado como infra")
		}
	})

	t.Run("falha do banco é infra", func(t *testing.T) {
		repo := NovoRepositorio(abrirBancoFalso(t, comportamento{err: errConexao}))
		_, err := repo.BuscarPorReserva(ctx, "r1")
		if !errors.Is(err, usecase.ErrDependenciaIndisponivel) || !errors.Is(err, errConexao) {
			t.Fatalf("queria ErrDependenciaIndisponivel com a causa, veio %v", err)
		}
		if errors.Is(err, usecase.ErrNaoEncontrada) {
			t.Fatal("infra não pode ser tratada como não encontrado")
		}
	})
}

func TestEscritasSemLinhaAfetadaSaoErroDeNegocio(t *testing.T) {
	ctx := context.Background()
	repo := NovoRepositorio(abrirBancoFalso(t, comportamento{afetadas: 0}))

	if err := repo.RegistrarEscolha(ctx, transacao.Transacao{ID: "t1"}); !errors.Is(err, usecase.ErrJaFinalizada) {
		t.Errorf("RegistrarEscolha: queria ErrJaFinalizada, veio %v", err)
	}
	if err := repo.Finalizar(ctx, transacao.Transacao{ID: "t1"}); !errors.Is(err, usecase.ErrJaFinalizada) {
		t.Errorf("Finalizar: queria ErrJaFinalizada, veio %v", err)
	}
	if ok, err := repo.ReivindicarCobranca(ctx, "t1", time.Now()); err != nil || ok {
		t.Errorf("ReivindicarCobranca sem linha: ok=%v err=%v, queria false e nil", ok, err)
	}
}

func TestReivindicarCobrancaComLinhaAfetada(t *testing.T) {
	repo := NovoRepositorio(abrirBancoFalso(t, comportamento{afetadas: 1}))
	ok, err := repo.ReivindicarCobranca(context.Background(), "t1", time.Now())
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v, queria true e nil", ok, err)
	}
}

func TestFalhaDoBancoNasOperacoesViraInfra(t *testing.T) {
	ctx := context.Background()
	repo := NovoRepositorio(abrirBancoFalso(t, comportamento{err: errConexao}))
	agora := time.Now()

	operacoes := map[string]func() error{
		"CriarSeAusente":   func() error { _, _, err := repo.CriarSeAusente(ctx, transacao.Transacao{ID: "t1"}); return err },
		"RegistrarEscolha": func() error { return repo.RegistrarEscolha(ctx, transacao.Transacao{ID: "t1"}) },
		"Finalizar":        func() error { return repo.Finalizar(ctx, transacao.Transacao{ID: "t1"}) },
		"MarcarAnunciado":  func() error { return repo.MarcarAnunciado(ctx, "t1", agora) },
		"LiberarCobranca":  func() error { return repo.LiberarCobranca(ctx, "t1", agora) },
		"ReivindicarCobranca": func() error {
			_, err := repo.ReivindicarCobranca(ctx, "t1", agora)
			return err
		},
		"AguardandoCobranca": func() error { _, err := repo.AguardandoCobranca(ctx, 10); return err },
		"CancelarEsperasVencidas": func() error {
			_, err := repo.CancelarEsperasVencidas(ctx, agora, 10)
			return err
		},
		"AnunciosPendentes": func() error { _, err := repo.AnunciosPendentes(ctx, 10); return err },
	}
	for nome, op := range operacoes {
		t.Run(nome, func(t *testing.T) {
			err := op()
			if !errors.Is(err, usecase.ErrDependenciaIndisponivel) || !errors.Is(err, errConexao) {
				t.Fatalf("queria ErrDependenciaIndisponivel com a causa, veio %v", err)
			}
		})
	}
}
