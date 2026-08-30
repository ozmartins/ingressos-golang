// Package shared reúne os tipos de domínio usados por mais de um agregado.
package shared

import "errors"

// Erros de domínio. São a fonte da categoria que o adaptador gRPC traduz em
// status — ver contracts/erros.md. O núcleo não conhece gRPC; ele só declara
// o que aconteceu, e o adaptador decide como isso aparece no contrato.
var (
	// ErrSolicitacaoInvalida cobre erro do chamador: lista vazia, rótulos
	// repetidos, usuário ausente, rótulo malformado.
	ErrSolicitacaoInvalida = errors.New("solicitação inválida")

	// ErrLimiteExcedido é solicitação acima do máximo de poltronas por bloqueio.
	ErrLimiteExcedido = errors.New("limite de poltronas por bloqueio excedido")

	// ErrSessaoNaoProvisionada é bloqueio para sessão sem matriz de poltronas.
	ErrSessaoNaoProvisionada = errors.New("sessão sem matriz de poltronas provisionada")

	// ErrPoltronaInexistente é rótulo que não existe na sessão informada.
	ErrPoltronaInexistente = errors.New("poltrona inexistente na sessão")

	// ErrPoltronasIndisponiveis é o único desfecho que vira sucesso=false na
	// resposta de bloqueio: ao menos uma poltrona já está tomada.
	ErrPoltronasIndisponiveis = errors.New("poltronas indisponíveis")

	// ErrSessaoDesconhecida é consulta de mapa para sessão que o serviço não conhece.
	ErrSessaoDesconhecida = errors.New("sessão desconhecida")

	// ErrReservaDesconhecida é desfecho de pagamento para reserva inexistente.
	ErrReservaDesconhecida = errors.New("reserva desconhecida")

	// ErrTransicaoInvalida é tentativa de mudar o estado de algo já finalizado.
	ErrTransicaoInvalida = errors.New("transição de estado inválida")

	// ErrDependenciaIndisponivel sinaliza que não foi possível garantir a
	// exclusividade ou gravar o estado. Nunca se concede bloqueio sem garantia.
	ErrDependenciaIndisponivel = errors.New("dependência indisponível")
)
