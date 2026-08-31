// Package sistema traz os adaptadores triviais das portas de relógio e
// identidade. Existem para que o domínio e os casos de uso não toquem em
// time.Now nem no gerador de UUID (research.md D9).
package sistema

import (
	"time"

	"github.com/google/uuid"
)

// Relogio devolve o instante corrente em UTC.
type Relogio struct{}

func (Relogio) Agora() time.Time { return time.Now().UTC() }

// GeradorID produz UUID v4.
type GeradorID struct{}

func (GeradorID) Novo() string { return uuid.NewString() }
