package reserva

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type Status string

const (
	Pendente   Status = "PENDENTE"
	Confirmada Status = "CONFIRMADA"
	Expirada   Status = "EXPIRADA"
	Cancelada  Status = "CANCELADA"
)

func (s Status) Final() bool {
	return s == Confirmada || s == Expirada || s == Cancelada
}

type Reserva struct {
	ID        string
	SessaoID  string
	UsuarioID string
	Rotulos   []string
	ExpiraEm  time.Time
	Status    Status
	CriadoEm  time.Time
	// Valor a cobrar, como texto decimal. Guardado aqui porque é ele que segue
	// no fato `reserva.criada` para quem cobra; este serviço não o interpreta.
	ValorTotal string
}

func Nova(sol Solicitacao, agora time.Time, ttl time.Duration) Reserva {
	return Reserva{
		ID:         uuid.NewString(),
		SessaoID:   sol.SessaoID,
		UsuarioID:  sol.UsuarioID,
		Rotulos:    sol.Rotulos,
		ExpiraEm:   agora.Add(ttl).UTC(),
		Status:     Pendente,
		CriadoEm:   agora.UTC(),
		ValorTotal: sol.ValorTotal,
	}
}

func (r Reserva) Expirou(agora time.Time) bool {
	return r.Status == Pendente && !agora.Before(r.ExpiraEm)
}

func (r Reserva) PodeConfirmar() bool { return r.Status == Pendente }

func (r Reserva) PodeCancelar() bool { return r.Status == Pendente }

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
