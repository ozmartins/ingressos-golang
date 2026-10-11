// Package sessao guarda a visão que o estoque tem da sessão do catálogo: só o
// que é preciso para decidir se ela aceita reserva e quanto custa.
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

// AceitaReserva espelha a regra do catálogo: sessão agendada e ainda por começar.
func (s Sessao) AceitaReserva(agora time.Time) bool {
	return s.Status == Agendada && s.Inicio.After(agora)
}

// ValorTotal devolve preço_base × quantidade como decimal com duas casas ("84.00").
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
