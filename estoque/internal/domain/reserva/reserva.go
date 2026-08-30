// Package reserva modela a intenção de compra sobre um conjunto de poltronas e
// a máquina de estados que a governa.
package reserva

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// Status é o estado da reserva.
type Status string

// Estados possíveis (FR-010).
const (
	Pendente   Status = "PENDENTE"
	Confirmada Status = "CONFIRMADA"
	Expirada   Status = "EXPIRADA"
	Cancelada  Status = "CANCELADA"
)

// Final diz se o estado é terminal. Reserva finalizada é imutável (FR-011).
func (s Status) Final() bool {
	return s == Confirmada || s == Expirada || s == Cancelada
}

// Reserva é uma reserva de poltronas de uma sessão para uma pessoa.
type Reserva struct {
	ID        string
	SessaoID  string
	UsuarioID string
	Rotulos   []string
	ExpiraEm  time.Time
	Status    Status
	CriadoEm  time.Time
}

// Nova cria uma reserva pendente com prazo contado a partir de agora (FR-007).
func Nova(sessaoID, usuarioID string, rotulos []string, agora time.Time, ttl time.Duration) Reserva {
	return Reserva{
		ID:        uuid.NewString(),
		SessaoID:  sessaoID,
		UsuarioID: usuarioID,
		Rotulos:   rotulos,
		ExpiraEm:  agora.Add(ttl).UTC(),
		Status:    Pendente,
		CriadoEm:  agora.UTC(),
	}
}

// Expirou diz se o prazo venceu no instante informado. Só faz sentido para
// reserva pendente: confirmada nunca expira (FR-014).
func (r Reserva) Expirou(agora time.Time) bool {
	return r.Status == Pendente && !agora.Before(r.ExpiraEm)
}

// PodeConfirmar diz se a reserva admite confirmação por pagamento aprovado.
func (r Reserva) PodeConfirmar() bool { return r.Status == Pendente }

// PodeCancelar diz se a reserva admite cancelamento por pagamento recusado.
func (r Reserva) PodeCancelar() bool { return r.Status == Pendente }

// Transicionar leva a reserva a um estado final, recusando qualquer mudança a
// partir de estado já finalizado (FR-011).
func (r Reserva) Transicionar(novo Status) (Reserva, error) {
	if r.Status != Pendente {
		return r, fmt.Errorf("%w: reserva %s já está %s", shared.ErrTransicaoInvalida, r.ID, r.Status)
	}
	if !novo.Final() {
		return r, fmt.Errorf("%w: %s não é estado final", shared.ErrTransicaoInvalida, novo)
	}
	r.Status = novo
	return r, nil
}
