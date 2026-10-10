package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

var errConexao = errors.New("conexão recusada")

func TestBuscarPorIDDistingueNaoEncontradoDeInfra(t *testing.T) {
	ctx := context.Background()

	t.Run("sem linhas é não encontrado", func(t *testing.T) {
		repo := Ingressos{DB: abrirBancoFalso(t, comportamento{})}
		_, err := repo.BuscarPorID(ctx, "i1")
		if !errors.Is(err, usecase.ErrNaoEncontrado) {
			t.Fatalf("queria ErrNaoEncontrado, veio %v", err)
		}
		if errors.Is(err, usecase.ErrDependenciaIndisponivel) {
			t.Fatal("não encontrado não pode ser tratado como infra")
		}
	})

	t.Run("falha do banco é infra", func(t *testing.T) {
		repo := Ingressos{DB: abrirBancoFalso(t, comportamento{err: errConexao})}
		_, err := repo.BuscarPorID(ctx, "i1")
		if !errors.Is(err, usecase.ErrDependenciaIndisponivel) || !errors.Is(err, errConexao) {
			t.Fatalf("queria ErrDependenciaIndisponivel com a causa, veio %v", err)
		}
		if errors.Is(err, usecase.ErrNaoEncontrado) {
			t.Fatal("infra não pode ser tratada como não encontrado")
		}
	})
}

func TestUtilizarTraduzLinhasAfetadas(t *testing.T) {
	ctx := context.Background()
	agora := time.Now()

	casos := []struct {
		nome     string
		afetadas int64
		querOk   bool
	}{
		{"uma linha: baixa autorizada", 1, true},
		{"nenhuma linha: já utilizado ou inexistente", 0, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			repo := Ingressos{DB: abrirBancoFalso(t, comportamento{afetadas: c.afetadas})}
			ok, err := repo.Utilizar(ctx, "i1", agora)
			if err != nil || ok != c.querOk {
				t.Fatalf("ok=%v err=%v, queria ok=%v e nil", ok, err, c.querOk)
			}
		})
	}
}

func TestFalhaDoBancoNasOperacoesViraInfra(t *testing.T) {
	ctx := context.Background()
	db := abrirBancoFalso(t, comportamento{err: errConexao})
	repo := Ingressos{DB: db}

	operacoes := map[string]func() error{
		"CriarSeAusente": func() error {
			_, _, err := repo.CriarSeAusente(ctx, ingresso.Ingresso{ID: "i1", ReservaID: "r1"})
			return err
		},
		"Utilizar":         func() error { _, err := repo.Utilizar(ctx, "i1", time.Now()); return err },
		"ListarPorUsuario": func() error { _, err := repo.ListarPorUsuario(ctx, "u1", ""); return err },
		"Avisos.Registrar": func() error { return Avisos{DB: db}.Registrar(ctx, aviso.Registro{}) },
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
