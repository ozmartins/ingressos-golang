package shared

import "errors"

var (
	ErrSolicitacaoInvalida = errors.New("solicitação inválida")

	ErrLimiteExcedido = errors.New("limite de poltronas por bloqueio excedido")

	ErrSessaoNaoProvisionada = errors.New("sessão sem matriz de poltronas provisionada")

	ErrPoltronaInexistente = errors.New("poltrona inexistente na sessão")

	ErrPoltronasIndisponiveis = errors.New("poltronas indisponíveis")

	ErrSessaoDesconhecida = errors.New("sessão desconhecida")

	ErrReservaDesconhecida = errors.New("reserva desconhecida")

	ErrTransicaoInvalida = errors.New("transição de estado inválida")

	ErrDependenciaIndisponivel = errors.New("dependência indisponível")
)
