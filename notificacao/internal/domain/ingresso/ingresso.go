package ingresso

import (
	"errors"
	"time"
)

type Status string

const (
	Valido    Status = "VALIDO"
	Utilizado Status = "UTILIZADO"
	Cancelado Status = "CANCELADO"
)

func Reconhecido(s Status) bool {
	return s == Valido || s == Utilizado || s == Cancelado
}

func (s Status) Terminal() bool { return s == Utilizado || s == Cancelado }

var (
	ErrTransicaoInvalida = errors.New("ingresso: transição inválida a partir de estado terminal")
	ErrDadosObrigatorios = errors.New("ingresso: reserva, usuário, id e código são obrigatórios")
)

type Ingresso struct {
	ID          string
	ReservaID   string
	UsuarioID   string
	CodigoQR    string
	Status      Status
	UtilizadoEm *time.Time
	CriadoEm    time.Time
}

func Novo(id, reservaID, usuarioID, codigoQR string, agora time.Time) (Ingresso, error) {
	if id == "" || reservaID == "" || usuarioID == "" || codigoQR == "" {
		return Ingresso{}, ErrDadosObrigatorios
	}
	return Ingresso{
		ID:        id,
		ReservaID: reservaID,
		UsuarioID: usuarioID,
		CodigoQR:  codigoQR,
		Status:    Valido,
		CriadoEm:  agora,
	}, nil
}

func (i Ingresso) Utilizar(agora time.Time) (Ingresso, error) {
	if i.Status.Terminal() {
		return i, ErrTransicaoInvalida
	}
	i.Status = Utilizado
	i.UtilizadoEm = &agora
	return i, nil
}

func (i Ingresso) Cancelar() (Ingresso, error) {
	if i.Status.Terminal() {
		return i, ErrTransicaoInvalida
	}
	i.Status = Cancelado
	return i, nil
}

func (i Ingresso) InvarianteInstante() bool {
	return (i.Status == Utilizado) == (i.UtilizadoEm != nil)
}
