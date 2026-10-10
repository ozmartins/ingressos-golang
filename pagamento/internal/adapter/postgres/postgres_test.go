package postgres

import (
	"errors"
	"testing"

	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

func TestFalhaInfraMarcaErroSemPerderACausa(t *testing.T) {
	causa := errors.New("conexão recusada")
	err := falhaInfra(causa)
	if !errors.Is(err, usecase.ErrDependenciaIndisponivel) {
		t.Fatalf("devia ser ErrDependenciaIndisponivel, veio %v", err)
	}
	if !errors.Is(err, causa) {
		t.Fatalf("devia preservar a causa, veio %v", err)
	}
	if errors.Is(err, usecase.ErrNaoEncontrada) {
		t.Fatal("falha de infra não pode parecer 'não encontrado'")
	}
}

func TestFalhaInfraNilContinuaNil(t *testing.T) {
	if err := falhaInfra(nil); err != nil {
		t.Fatalf("esperava nil, veio %v", err)
	}
}
