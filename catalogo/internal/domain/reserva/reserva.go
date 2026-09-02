package reserva

import (
	"fmt"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type SolicitacaoReserva struct {
	SessaoID     string
	PoltronasIDs []string
	UsuarioID    string
}

func (s SolicitacaoReserva) Validar() error {
	if s.UsuarioID == "" {
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

type ResultadoReserva struct {
	ReservaID string
	ExpiraEm  time.Time
}

func (r ResultadoReserva) ValidarIntegridade() error {
	if r.ReservaID == "" {
		return fmt.Errorf("%w: sucesso sem identificador de reserva", shared.ErrRespostaInvalidaDoParceiro)
	}
	if r.ExpiraEm.IsZero() {
		return fmt.Errorf("%w: sucesso sem prazo de expiração", shared.ErrRespostaInvalidaDoParceiro)
	}
	return nil
}
