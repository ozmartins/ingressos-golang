package sessao

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

const Agendada = "AGENDADA"

type Sessao struct {
	ID        string
	Status    string
	Inicio    time.Time
	PrecoBase string
}

func (s Sessao) AceitaReserva(agora time.Time) bool {
	return s.Status == Agendada && s.Inicio.After(agora)
}

func (s Sessao) ValorTotal(quantidade int) (string, error) {
	centavos, err := centavosDe(s.PrecoBase)
	if err != nil {
		return "", err
	}
	total := centavos * int64(quantidade)
	return fmt.Sprintf("%d.%02d", total/100, total%100), nil
}

func centavosDe(preco string) (int64, error) {
	invalido := fmt.Errorf("%w: preco_base %q", shared.ErrRespostaInvalidaDoCatalogo, preco)

	inteiro, decimais, temPonto := strings.Cut(strings.TrimSpace(preco), ".")
	if inteiro == "" || (temPonto && (decimais == "" || len(decimais) > 2)) {
		return 0, invalido
	}
	decimais += strings.Repeat("0", 2-len(decimais))

	reais, err := strconv.ParseUint(inteiro, 10, 32)
	if err != nil {
		return 0, invalido
	}
	cents, err := strconv.ParseUint(decimais, 10, 8)
	if err != nil {
		return 0, invalido
	}
	total := int64(reais)*100 + int64(cents)
	if total <= 0 {
		return 0, invalido
	}
	return total, nil
}
