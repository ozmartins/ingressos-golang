package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

var errConexao = errors.New("conexão recusada")

func TestIndisponivelMarcaErroSemPerderACausa(t *testing.T) {
	err := indisponivel(errConexao)
	if !errors.Is(err, shared.ErrDependenciaIndisponivel) {
		t.Fatalf("devia ser ErrDependenciaIndisponivel, veio %v", err)
	}
	if !errors.Is(err, errConexao) {
		t.Fatalf("devia preservar a causa, veio %v", err)
	}
}

func TestIndisponivelNilContinuaNil(t *testing.T) {
	if err := indisponivel(nil); err != nil {
		t.Fatalf("queria nil, veio %v", err)
	}
}

func TestClassificacaoDeErrosDoPostgres(t *testing.T) {
	travamento := fmt.Errorf("embrulhado: %w", &pgconn.PgError{Code: "55P03"})
	unicidade := fmt.Errorf("embrulhado: %w", &pgconn.PgError{Code: "23505"})

	if !ehConflitoDeTravamento(travamento) || ehConflitoDeTravamento(unicidade) || ehConflitoDeTravamento(errConexao) {
		t.Error("ehConflitoDeTravamento deve reconhecer só 55P03")
	}
	if !ehViolacaoDeUnicidade(unicidade) || ehViolacaoDeUnicidade(travamento) || ehViolacaoDeUnicidade(errConexao) {
		t.Error("ehViolacaoDeUnicidade deve reconhecer só 23505")
	}
}

func TestFalhaDoBancoEhInfraNasConsultas(t *testing.T) {
	b := &Banco{db: abrirBancoFalso(t, comportamento{err: errConexao})}

	_, err := b.LimparMensagensProcessadas(context.Background(), time.Hour)
	if !errors.Is(err, shared.ErrDependenciaIndisponivel) || !errors.Is(err, errConexao) {
		t.Fatalf("queria ErrDependenciaIndisponivel com a causa, veio %v", err)
	}

	err = b.EmTransacao(context.Background(), func(tx *gorm.DB) error {
		_, e := registrarProcessada(tx, "fila", "m1")
		return e
	})
	if !errors.Is(err, shared.ErrDependenciaIndisponivel) {
		t.Fatalf("EmTransacao: queria ErrDependenciaIndisponivel, veio %v", err)
	}
}

func TestRegistrarProcessadaDistingueDuplicata(t *testing.T) {
	for _, c := range []struct {
		afetadas int64
		novo     bool
	}{{1, true}, {0, false}} {
		db := abrirBancoFalso(t, comportamento{afetadas: c.afetadas})
		novo, err := registrarProcessada(db, "fila", "m1")
		if err != nil || novo != c.novo {
			t.Errorf("afetadas=%d: novo=%v err=%v, queria novo=%v", c.afetadas, novo, err, c.novo)
		}
	}
}
