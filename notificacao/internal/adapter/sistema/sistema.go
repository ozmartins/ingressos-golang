package sistema

import (
	"time"

	"github.com/google/uuid"
)

type Relogio struct{}

func (Relogio) Agora() time.Time { return time.Now().UTC() }

type GeradorID struct{}

func (GeradorID) Novo() string { return uuid.NewString() }
