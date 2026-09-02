package aviso

import (
	"errors"
	"time"
)

type Canal string

const (
	Email Canal = "EMAIL"
	Push  Canal = "PUSH"
	SMS   Canal = "SMS"
)

type Desfecho string

const (
	Enviado Desfecho = "ENVIADO"
	Falha   Desfecho = "FALHA"
)

var ErrDetalheObrigatorio = errors.New("aviso: registro de falha exige detalhe")

type Registro struct {
	ID         string
	IngressoID string
	UsuarioID  string
	Canal      Canal
	Desfecho   Desfecho
	Detalhes   string
	EnviadoEm  time.Time
}

func NovoEnviado(id, ingressoID, usuarioID string, canal Canal, agora time.Time) Registro {
	return Registro{
		ID: id, IngressoID: ingressoID, UsuarioID: usuarioID,
		Canal: canal, Desfecho: Enviado, EnviadoEm: agora,
	}
}

func NovoFalho(id, ingressoID, usuarioID string, canal Canal, detalhes string, agora time.Time) (Registro, error) {
	if detalhes == "" {
		return Registro{}, ErrDetalheObrigatorio
	}
	return Registro{
		ID: id, IngressoID: ingressoID, UsuarioID: usuarioID,
		Canal: canal, Desfecho: Falha, Detalhes: detalhes, EnviadoEm: agora,
	}, nil
}

func (r Registro) PendenteDeReenvio() bool { return r.Desfecho == Falha }
