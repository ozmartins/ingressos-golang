package shared

import "errors"

var (
	ErrValidacao = errors.New("entrada inválida")

	ErrNaoEncontrado = errors.New("recurso não encontrado")

	ErrSessaoNaoReservavel = errors.New("sessão não aceita reservas")

	ErrPoltronasIndisponiveis = errors.New("poltronas indisponíveis")

	ErrEstoqueIndisponivel = errors.New("serviço de estoque indisponível")

	ErrRespostaInvalidaDoParceiro = errors.New("resposta inválida do serviço de estoque")
)
