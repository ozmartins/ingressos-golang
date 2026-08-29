// Package reserva modela a intenção de compra. Nada aqui é persistido por este
// serviço: a titularidade sobre poltronas pertence ao Servico-Estoque
// (constituição, princípio III).
package reserva

import (
	"fmt"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// SolicitacaoReserva é o pedido de bloqueio de um conjunto de poltronas.
type SolicitacaoReserva struct {
	SessaoID     string
	PoltronasIDs []string
	UsuarioID    string
}

// Validar checa o que dá para checar sem sair do processo.
//
// Falhar aqui evita uma ida à rede: é o que faz uma solicitação sem identidade
// ou com lista malformada nunca chegar ao estoque.
func (s SolicitacaoReserva) Validar() error {
	if s.UsuarioID == "" {
		// Credencial válida em assinatura mas sem identidade: não há a quem
		// atribuir a reserva.
		return fmt.Errorf("%w: identidade da pessoa usuária ausente", shared.ErrValidacao)
	}
	if s.SessaoID == "" {
		return fmt.Errorf("%w: sessão não informada", shared.ErrValidacao)
	}
	if len(s.PoltronasIDs) == 0 {
		return fmt.Errorf("%w: informe ao menos uma poltrona", shared.ErrValidacao)
	}
	vistas := make(map[string]struct{}, len(s.PoltronasIDs))
	for _, p := range s.PoltronasIDs {
		if p == "" {
			return fmt.Errorf("%w: identificador de poltrona vazio", shared.ErrValidacao)
		}
		if _, repetida := vistas[p]; repetida {
			return fmt.Errorf("%w: poltrona %q informada mais de uma vez", shared.ErrValidacao, p)
		}
		vistas[p] = struct{}{}
	}
	return nil
}

// ResultadoReserva é a resposta do estoque, já traduzida para o domínio.
type ResultadoReserva struct {
	ReservaID string
	ExpiraEm  time.Time
}

// ValidarIntegridade recusa uma confirmação sem os dados que a tornam útil.
//
// O estoque alegar sucesso sem identificador ou sem prazo de expiração é falha
// de contrato do parceiro: repassar isso como sucesso deixaria o cliente com uma
// reserva que ele não consegue referenciar nem saber quando expira.
func (r ResultadoReserva) ValidarIntegridade() error {
	if r.ReservaID == "" {
		return fmt.Errorf("%w: sucesso sem identificador de reserva", shared.ErrRespostaInvalidaDoParceiro)
	}
	if r.ExpiraEm.IsZero() {
		return fmt.Errorf("%w: sucesso sem prazo de expiração", shared.ErrRespostaInvalidaDoParceiro)
	}
	return nil
}
