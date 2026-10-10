package shared

import (
	"errors"
	"fmt"
)

var (
	ErrValidacao = errors.New("entrada inválida")

	ErrNaoEncontrado = errors.New("recurso não encontrado")

	ErrConflito = errors.New("conflito com o estado atual")

	ErrSessaoNaoReservavel = errors.New("sessão não aceita reservas")

	ErrPoltronasIndisponiveis = errors.New("poltronas indisponíveis")

	ErrEstoqueIndisponivel = errors.New("serviço de estoque indisponível")

	ErrBancoIndisponivel = errors.New("banco de dados indisponível")

	ErrSolicitacaoRecusadaPeloEstoque = errors.New("solicitação de bloqueio recusada pelo estoque")

	ErrPoltronaInexistente = errors.New("poltrona inexistente na sessão")

	ErrSessaoSemPoltronas = errors.New("sessão ainda sem poltronas provisionadas")

	ErrEstoqueComDefeito = errors.New("falha interna do serviço de estoque")

	ErrRespostaInvalidaDoParceiro = errors.New("resposta inválida do serviço de estoque")
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
