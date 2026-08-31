// Package aviso é o domínio do registro de notificação: o comprovante de uma
// tentativa de avisar a pessoa sobre um ingresso.
package aviso

import (
	"errors"
	"time"
)

// Canal é o meio pelo qual o aviso saiu. Vocabulário fechado (data-model.md §3).
type Canal string

const (
	Email Canal = "EMAIL"
	Push  Canal = "PUSH"
	SMS   Canal = "SMS"
)

// Desfecho é o resultado da tentativa.
type Desfecho string

const (
	Enviado Desfecho = "ENVIADO"
	Falha   Desfecho = "FALHA"
)

// ErrDetalheObrigatorio protege a razão de a tabela existir: um registro de
// falha sem motivo não serve ao reenvio (FR-017).
var ErrDetalheObrigatorio = errors.New("aviso: registro de falha exige detalhe")

// Registro é uma tentativa de aviso.
type Registro struct {
	ID         string
	IngressoID string
	UsuarioID  string
	Canal      Canal
	Desfecho   Desfecho
	Detalhes   string
	EnviadoEm  time.Time
}

// NovoEnviado registra um aviso que saiu.
func NovoEnviado(id, ingressoID, usuarioID string, canal Canal, agora time.Time) Registro {
	return Registro{
		ID: id, IngressoID: ingressoID, UsuarioID: usuarioID,
		Canal: canal, Desfecho: Enviado, EnviadoEm: agora,
	}
}

// NovoFalho registra uma tentativa que não saiu. Exige detalhe.
func NovoFalho(id, ingressoID, usuarioID string, canal Canal, detalhes string, agora time.Time) (Registro, error) {
	if detalhes == "" {
		return Registro{}, ErrDetalheObrigatorio
	}
	return Registro{
		ID: id, IngressoID: ingressoID, UsuarioID: usuarioID,
		Canal: canal, Desfecho: Falha, Detalhes: detalhes, EnviadoEm: agora,
	}, nil
}

// PendenteDeReenvio identifica os registros que uma feature futura de reenvio
// vai procurar (FR-018).
func (r Registro) PendenteDeReenvio() bool { return r.Desfecho == Falha }
