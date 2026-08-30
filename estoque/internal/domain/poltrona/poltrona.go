// Package poltrona modela o assento de uma sessão e as transições de estado que
// ele admite. Não conhece banco, rede nem contrato de transporte.
package poltrona

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// Status é o estado de ocupação de uma poltrona.
type Status string

// Estados possíveis (FR-010).
const (
	Livre     Status = "LIVRE"
	Reservada Status = "RESERVADA"
	Ocupada   Status = "OCUPADA"
)

// Tipo é a categoria física do assento.
type Tipo string

// Tipos possíveis.
const (
	Normal      Tipo = "NORMAL"
	PCD         Tipo = "PCD"
	Namoradeira Tipo = "NAMORADEIRA"
)

// TipoValido diz se t é um tipo reconhecido. Layout com tipo desconhecido
// invalida a mensagem inteira (FR-035).
func TipoValido(t Tipo) bool {
	switch t {
	case Normal, PCD, Namoradeira:
		return true
	default:
		return false
	}
}

// namespacePoltrona é o namespace UUID v5 do serviço. Fixo por definição: mudar
// este valor mudaria a identidade de toda poltrona já provisionada.
var namespacePoltrona = uuid.MustParse("6f9619ff-8b86-d011-b42d-00c04fc964ff")

// Poltrona é um assento de uma sessão.
type Poltrona struct {
	ID       string
	SessaoID string
	Fileira  string
	Numero   int
	Rotulo   string
	Tipo     Tipo
	Status   Status
}

// Nova monta uma poltrona no estado livre, derivando rótulo e identificador de
// forma determinística a partir de sessão, fileira e número (research D6). É
// isso que torna o provisionamento idempotente: reprocessar o mesmo fato
// recalcula o mesmo identificador e colide na chave.
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

// DerivarID devolve o UUID v5 determinístico de sessao|fileira|numero.
func DerivarID(sessaoID, fileira string, numero int) string {
	chave := fmt.Sprintf("%s|%s|%d", sessaoID, strings.ToUpper(fileira), numero)
	return uuid.NewSHA1(namespacePoltrona, []byte(chave)).String()
}

// MontarRotulo devolve a identidade de negócio da poltrona ("A1").
func MontarRotulo(fileira string, numero int) string {
	return strings.ToUpper(strings.TrimSpace(fileira)) + strconv.Itoa(numero)
}

// LerRotulo decompõe "A1" em fileira e número. O contrato aceita rótulos com
// fileira de uma ou mais letras seguida de dígitos.
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

// PodeSerBloqueada diz se a poltrona está disponível para um novo bloqueio.
func (p Poltrona) PodeSerBloqueada() bool { return p.Status == Livre }

// Transicionar aplica a mudança de estado, recusando as que o domínio não
// admite (FR-010). A poltrona ocupada é terminal nesta feature.
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
