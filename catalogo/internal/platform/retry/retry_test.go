package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutarRepeteAteSucesso(t *testing.T) {
	n := 0
	err := Executar(context.Background(), "t", Politica{Tentativas: 4, Base: time.Millisecond, Maximo: 2 * time.Millisecond}, func() error {
		n++
		if n < 3 {
			return errors.New("falha")
		}
		return nil
	})
	if err != nil || n != 3 {
		t.Fatalf("err=%v n=%d, queria sucesso em 3", err, n)
	}
}

func TestExecutarNaoRepeteErroNaoRetentavel(t *testing.T) {
	n := 0
	negocio := errors.New("negócio")
	err := Executar(context.Background(), "t", Politica{
		Tentativas: 5, Base: time.Millisecond,
		Retentavel: func(e error) bool { return !errors.Is(e, negocio) },
	}, func() error { n++; return negocio })
	if !errors.Is(err, negocio) || n != 1 {
		t.Fatalf("err=%v n=%d, queria 1 execução", err, n)
	}
}

func TestExecutarEsgotaTentativas(t *testing.T) {
	n := 0
	err := Executar(context.Background(), "t", Politica{Tentativas: 3, Base: time.Millisecond}, func() error { n++; return errors.New("x") })
	if err == nil || n != 3 {
		t.Fatalf("err=%v n=%d, queria 3 execuções e erro", err, n)
	}
}

func TestExecutarRespeitaContexto(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	n := 0
	err := Executar(ctx, "t", Politica{Tentativas: 10, Base: time.Hour, Maximo: time.Hour}, func() error {
		n++
		cancelar()
		return errors.New("x")
	})
	if err == nil || n != 1 {
		t.Fatalf("err=%v n=%d, queria parar após cancelamento", err, n)
	}
}

func TestEsperaLimitadaPeloMaximo(t *testing.T) {
	p := Politica{Base: 100 * time.Millisecond, Maximo: 400 * time.Millisecond}
	for i := 0; i < 50; i++ {
		if e := p.Espera(10); e < 0 || e > p.Maximo {
			t.Fatalf("espera %v fora de [0, %v]", e, p.Maximo)
		}
	}
}
