package shared

import "errors"

// Erros sentinela do domínio. Os adaptadores traduzem cada um para a categoria
// correspondente do contrato de erro (constituição, princípio IV).
var (
	// ErrValidacao: entrada malformada ou fora do domínio aceito.
	ErrValidacao = errors.New("entrada inválida")

	// ErrNaoEncontrado: recurso inexistente.
	ErrNaoEncontrado = errors.New("recurso não encontrado")

	// ErrSessaoNaoReservavel: a sessão existe, mas não aceita reservas.
	ErrSessaoNaoReservavel = errors.New("sessão não aceita reservas")

	// ErrPoltronasIndisponiveis: ao menos uma poltrona pedida não está livre.
	ErrPoltronasIndisponiveis = errors.New("poltronas indisponíveis")

	// ErrEstoqueIndisponivel: o serviço de estoque falhou, excedeu o tempo
	// máximo ou está em recusa rápida. Indistinguível para o cliente por
	// exigência da especificação.
	ErrEstoqueIndisponivel = errors.New("serviço de estoque indisponível")

	// ErrRespostaInvalidaDoParceiro: o estoque alegou sucesso sem os dados
	// obrigatórios da reserva.
	ErrRespostaInvalidaDoParceiro = errors.New("resposta inválida do serviço de estoque")
)
