package poltrona

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type Status string

const (
	Livre     Status = "LIVRE"
	Reservada Status = "RESERVADA"
	Ocupada   Status = "OCUPADA"
)

type Tipo string

const (
	Normal      Tipo = "NORMAL"
	PCD         Tipo = "PCD"
	Namoradeira Tipo = "NAMORADEIRA"
)

func TipoValido(t Tipo) bool {
	switch t {
	case Normal, PCD, Namoradeira:
		return true
	default:
		return false
	}
}

var namespacePoltrona = uuid.MustParse("6f9619ff-8b86-d011-b42d-00c04fc964ff")

type Poltrona struct {
	ID       string
	SessaoID string
	Fileira  string
	Numero   int
	Rotulo   string
	Tipo     Tipo
	Status   Status
}

func Nova(sessaoID, fileira string, numero int, tipo Tipo) (Poltrona, error) {
	fileira = strings.ToUpper(strings.TrimSpace(fileira))
	if sessaoID == "" {
		return Poltrona{}, fmt.Errorf("%w: sessão vazia", shared.ErrSolicitacaoInvalida)
	}
	if fileira == "" || len(fileira) > 5 {
		return Poltrona{}, fmt.Errorf("%w: fileira %q inválida", shared.ErrSolicitacaoInvalida, fileira)
	}
	if numero <= 0 {
		return Poltrona{}, fmt.Errorf("%w: número %d inválido", shared.ErrSolicitacaoInvalida, numero)
	}
	if !TipoValido(tipo) {
		return Poltrona{}, fmt.Errorf("%w: tipo %q desconhecido", shared.ErrSolicitacaoInvalida, tipo)
	}

	return Poltrona{
		ID:       DerivarID(sessaoID, fileira, numero),
		SessaoID: sessaoID,
		Fileira:  fileira,
		Numero:   numero,
		Rotulo:   MontarRotulo(fileira, numero),
		Tipo:     tipo,
		Status:   Livre,
	}, nil
}

func DerivarID(sessaoID, fileira string, numero int) string {
	chave := fmt.Sprintf("%s|%s|%d", sessaoID, strings.ToUpper(fileira), numero)
	return uuid.NewSHA1(namespacePoltrona, []byte(chave)).String()
}

func MontarRotulo(fileira string, numero int) string {
	return strings.ToUpper(strings.TrimSpace(fileira)) + strconv.Itoa(numero)
}

func LerRotulo(rotulo string) (fileira string, numero int, err error) {
	r := strings.ToUpper(strings.TrimSpace(rotulo))
	if r == "" {
		return "", 0, fmt.Errorf("%w: rótulo vazio", shared.ErrSolicitacaoInvalida)
	}

	corte := -1
	for i, c := range r {
		if c >= '0' && c <= '9' {
			corte = i
			break
		}
	}
	if corte <= 0 || corte > 5 {
		return "", 0, fmt.Errorf("%w: rótulo %q fora do formato fileira+número", shared.ErrSolicitacaoInvalida, rotulo)
	}

	fileira = r[:corte]
	for _, c := range fileira {
		if c < 'A' || c > 'Z' {
			return "", 0, fmt.Errorf("%w: rótulo %q fora do formato fileira+número", shared.ErrSolicitacaoInvalida, rotulo)
		}
	}

	numero, err = strconv.Atoi(r[corte:])
	if err != nil || numero <= 0 {
		return "", 0, fmt.Errorf("%w: rótulo %q fora do formato fileira+número", shared.ErrSolicitacaoInvalida, rotulo)
	}
	return fileira, numero, nil
}

func (p Poltrona) PodeSerBloqueada() bool { return p.Status == Livre }

func (p Poltrona) Transicionar(novo Status) (Poltrona, error) {
	permitido := map[Status][]Status{
		Livre:     {Reservada},
		Reservada: {Ocupada, Livre},
		Ocupada:   {},
	}
	for _, destino := range permitido[p.Status] {
		if destino == novo {
			p.Status = novo
			return p, nil
		}
	}
	return p, fmt.Errorf("%w: poltrona %s de %s para %s", shared.ErrTransicaoInvalida, p.Rotulo, p.Status, novo)
}
