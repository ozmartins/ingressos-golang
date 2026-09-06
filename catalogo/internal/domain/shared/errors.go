package shared

import (
	"errors"
	"fmt"
)

var (
	ErrValidacao = errors.New("entrada inválida")

	ErrNaoEncontrado = errors.New("recurso não encontrado")

	// Entrada bem-formada que colide com o que já está gravado — número de sala
	// repetido, sala ocupada no horário. Não é validação: o mesmo corpo seria
	// aceito contra outro estado do catálogo.
	ErrConflito = errors.New("conflito com o estado atual")

	ErrSessaoNaoReservavel = errors.New("sessão não aceita reservas")

	ErrPoltronasIndisponiveis = errors.New("poltronas indisponíveis")

	ErrEstoqueIndisponivel = errors.New("serviço de estoque indisponível")

	ErrRespostaInvalidaDoParceiro = errors.New("resposta inválida do serviço de estoque")
)

// Qual recurso faltou muda a resposta ao cliente, e nem sempre é o recurso do
// caminho: a escrita de uma sessão falha por causa do filme ou da sala. O erro
// carrega o nome em vez de deixar quem responde adivinhar pela URL.
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
