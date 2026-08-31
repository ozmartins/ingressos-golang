// Package ingresso é o núcleo de domínio do Servico-Notificacao: o ingresso
// digital, seus estados e as transições permitidas. Não importa adaptador,
// banco, rede nem relógio do sistema.
package ingresso

import (
	"errors"
	"time"
)

// Status é o estado de um ingresso. UTILIZADO e CANCELADO são terminais
// (data-model.md §2).
type Status string

const (
	Valido    Status = "VALIDO"
	Utilizado Status = "UTILIZADO"
	Cancelado Status = "CANCELADO"
)

// Reconhecido diz se o estado consta do vocabulário fechado da FR-019. Serve
// também ao filtro da listagem, que recusa estado desconhecido (FR-024).
func Reconhecido(s Status) bool {
	return s == Valido || s == Utilizado || s == Cancelado
}

// Terminal diz se não há saída deste estado.
func (s Status) Terminal() bool { return s == Utilizado || s == Cancelado }

var (
	// ErrTransicaoInvalida é devolvido ao tentar sair de um estado terminal.
	ErrTransicaoInvalida = errors.New("ingresso: transição inválida a partir de estado terminal")
	// ErrDadosObrigatorios é devolvido quando falta identidade para criar.
	ErrDadosObrigatorios = errors.New("ingresso: reserva, usuário, id e código são obrigatórios")
)

// Ingresso é o bilhete digital de uma reserva paga.
//
// Depois de criado, ReservaID, UsuarioID, CodigoQR e CriadoEm nunca mudam
// (FR-020): nenhum método deste pacote os escreve, e Utilizar toca apenas
// Status e UtilizadoEm.
type Ingresso struct {
	ID          string
	ReservaID   string
	UsuarioID   string
	CodigoQR    string
	Status      Status
	UtilizadoEm *time.Time
	CriadoEm    time.Time
}

// Novo cria um ingresso VALIDO. É o único ponto de entrada do domínio.
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

// Utilizar aplica a baixa da portaria (FR-007). Devolve ErrTransicaoInvalida a
// partir de qualquer estado terminal — inclusive CANCELADO, que nesta feature
// não tem gatilho mas cuja defesa já fica no lugar (data-model.md §2).
func (i Ingresso) Utilizar(agora time.Time) (Ingresso, error) {
	if i.Status.Terminal() {
		return i, ErrTransicaoInvalida
	}
	i.Status = Utilizado
	i.UtilizadoEm = &agora
	return i, nil
}

// Cancelar existe para completar a máquina de estados e é exercitada pelos
// testes de transição. Nenhuma operação desta feature a chama (clarificação 2).
func (i Ingresso) Cancelar() (Ingresso, error) {
	if i.Status.Terminal() {
		return i, ErrTransicaoInvalida
	}
	i.Status = Cancelado
	return i, nil
}

// InvarianteInstante é a regra que o banco também guarda: tem instante de
// utilização se, e somente se, está utilizado (data-model.md §1).
func (i Ingresso) InvarianteInstante() bool {
	return (i.Status == Utilizado) == (i.UtilizadoEm != nil)
}
