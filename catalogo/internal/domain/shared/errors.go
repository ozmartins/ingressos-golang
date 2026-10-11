package shared

import (
	"errors"
	"fmt"
)

var (
	ErrValidacao = errors.New("entrada inválida")

	ErrNaoEncontrado = errors.New("recurso não encontrado")

	ErrConflito = errors.New("conflito com o estado atual")

	ErrBancoIndisponivel = errors.New("banco de dados indisponível")
)

type RecursoAusente struct {
	Recurso string
	ID      string
}

func NaoEncontrado(recurso, id string) error {
	return RecursoAusente{Recurso: recurso, ID: id}
}

func (e RecursoAusente) Error() string {
	return fmt.Sprintf("%s: %s %s", ErrNaoEncontrado, e.Recurso, e.ID)
}

func (e RecursoAusente) Unwrap() error { return ErrNaoEncontrado }
