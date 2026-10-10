package usecase

import (
	"fmt"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func errConflito(formato string, args ...any) error {
	return fmt.Errorf("%w: %s", shared.ErrConflito, fmt.Sprintf(formato, args...))
}
