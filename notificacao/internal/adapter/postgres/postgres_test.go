package postgres

import (
	"errors"
	"testing"

	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

func TestFalhaInfraMarcaErroSemPerderACausa(t *testing.T) {
	causa := errors.New("conexão recusada")
	err := falhaInfra("buscar ingresso", causa)
	if !errors.Is(err, usecase.ErrDependenciaIndisponivel) {
		t.Fatalf("devia ser ErrDependenciaIndisponivel, veio %v", err)
	}
	if !errors.Is(err, causa) {
		t.Fatalf("devia preservar a causa, veio %v", err)
	}
	if errors.Is(err, usecase.ErrNaoEncontrado) {
		t.Fatal("falha de infra não pode parecer 'não encontrado'")
	}
}
