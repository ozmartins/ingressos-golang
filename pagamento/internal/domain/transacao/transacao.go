// Package transacao é o núcleo de domínio do Servico-Pagamento: a transação de
// pagamento, seus estados, as transições permitidas e a regra de expiração.
// Não importa adaptador, banco, rede nem relógio do sistema.
package transacao

import (
	"errors"
	"time"
)

// Status é o estado de uma transação. Os quatro estados finais são terminais;
// só três deles são anunciáveis (data-model.md §2).
type Status string

const (
	Processando         Status = "PROCESSANDO"
	Pago                Status = "PAGO"
	Recusado            Status = "RECUSADO"
	Cancelado           Status = "CANCELADO"
	PendenteVerificacao Status = "PENDENTE_VERIFICACAO"
)

// Motivo é o vocabulário fechado de motivos de falha (data-model.md §3).
type Motivo string

const (
	MotivoReservaExpirada    Motivo = "RESERVA_EXPIRADA"
	MotivoSaldoInsuficiente  Motivo = "SALDO_INSUFICIENTE"
	MotivoCartaoRecusado     Motivo = "CARTAO_RECUSADO"
	MotivoRecusadoAdquirente Motivo = "RECUSADO_PELO_ADQUIRENTE"
)

// FormaPagamento é a forma reconhecida pelo serviço (FR-004).
type FormaPagamento string

const (
	PIX           FormaPagamento = "PIX"
	CartaoCredito FormaPagamento = "CARTAO_CREDITO"
)

// FormaReconhecida diz se a forma consta do contrato de entrada.
func FormaReconhecida(f FormaPagamento) bool {
	return f == PIX || f == CartaoCredito
}

var (
	// ErrTransicaoInvalida é devolvido quando se tenta sair de um estado terminal.
	ErrTransicaoInvalida = errors.New("transacao: transição inválida a partir de estado terminal")
	// ErrAnuncioInvalido é devolvido ao marcar como anunciado algo que não é anunciável.
	ErrAnuncioInvalido = errors.New("transacao: estado não é anunciável")
)

// Transacao é a tentativa de cobrança de uma reserva.
type Transacao struct {
	ID                     string
	ReservaID              string
	UsuarioID              string
	ValorTotal             string // decimal em texto: não se representa dinheiro em float
	FormaPagamento         FormaPagamento
	Status                 Status
	CodigoTransacaoGateway string
	MotivoFalha            Motivo
	CobrancaEmitida        bool
	ResultadoAnunciado     bool
	PagoEm                 *time.Time
	CriadoEm               time.Time
	AtualizadoEm           time.Time
}

// Nova cria uma transação em PROCESSANDO. É o único ponto de entrada do domínio:
// nenhuma transação nasce em estado final.
func Nova(id, reservaID, usuarioID, valor string, forma FormaPagamento, agora time.Time) Transacao {
	return Transacao{
		ID:             id,
		ReservaID:      reservaID,
		UsuarioID:      usuarioID,
		ValorTotal:     valor,
		FormaPagamento: forma,
		Status:         Processando,
		CriadoEm:       agora,
		AtualizadoEm:   agora,
	}
}

// Final diz se o estado não admite mais transição.
func (s Status) Final() bool { return s != Processando }

// Anunciavel diz se o estado gera anúncio de resultado. PENDENTE_VERIFICACAO é o
// único estado final que nunca é anunciado (FR-010, FR-022).
func (s Status) Anunciavel() bool {
	return s == Pago || s == Recusado || s == Cancelado
}

// Expirada diz se o prazo da reserva já passou no instante informado (FR-005).
// Sem folga de tolerância, por decisão registrada na clarificação Q5.
func Expirada(expiraEm, agora time.Time) bool {
	return !agora.Before(expiraEm)
}

// Aprovar leva a transação a PAGO. Erra se o estado atual já for terminal.
func (t *Transacao) Aprovar(codigoGateway string, agora time.Time) error {
	if t.Status.Final() {
		return ErrTransicaoInvalida
	}
	t.Status = Pago
	t.CodigoTransacaoGateway = codigoGateway
	pago := agora
	t.PagoEm = &pago
	t.AtualizadoEm = agora
	return nil
}

// Recusar leva a transação a RECUSADO, com o motivo devolvido pelo adquirente.
func (t *Transacao) Recusar(motivo Motivo, agora time.Time) error {
	if t.Status.Final() {
		return ErrTransicaoInvalida
	}
	t.Status = Recusado
	t.MotivoFalha = motivo
	t.AtualizadoEm = agora
	return nil
}

// Cancelar leva a transação a CANCELADO — hoje, só por reserva expirada (FR-005).
// Nenhuma cobrança foi tentada quando isto acontece.
func (t *Transacao) Cancelar(motivo Motivo, agora time.Time) error {
	if t.Status.Final() {
		return ErrTransicaoInvalida
	}
	t.Status = Cancelado
	t.MotivoFalha = motivo
	t.AtualizadoEm = agora
	return nil
}

// MarcarPendenteVerificacao é o desfecho de ausência de resposta do adquirente:
// não se sabe se a cobrança foi efetivada, então nada é anunciado (FR-022).
func (t *Transacao) MarcarPendenteVerificacao(agora time.Time) error {
	if t.Status.Final() {
		return ErrTransicaoInvalida
	}
	t.Status = PendenteVerificacao
	t.AtualizadoEm = agora
	return nil
}

// MarcarAnunciado registra que o resultado já foi publicado (FR-014). Só vale a
// partir de estado final anunciável.
func (t *Transacao) MarcarAnunciado(agora time.Time) error {
	if !t.Status.Anunciavel() {
		return ErrAnuncioInvalido
	}
	t.ResultadoAnunciado = true
	t.AtualizadoEm = agora
	return nil
}

// SeguroRetomar diz se uma reentrega pode tentar a cobrança de uma transação que
// ficou em PROCESSANDO. Só é seguro quando nenhuma cobrança chegou a ser emitida:
// a execução anterior morreu antes de falar com o adquirente, ou o adquirente
// devolveu erro — que, pelo contrato da porta, significa que nada foi enviado.
//
// Quando a cobrança foi emitida e não houve resposta conclusiva, retomar
// arriscaria cobrar duas vezes, e a FR-008 proíbe. O caso vira quarentena.
func (t Transacao) SeguroRetomar() bool {
	return t.Status == Processando && !t.CobrancaEmitida
}

// AnuncioPendente diz se há resultado gravado que ainda não foi publicado — a
// condição que faz a reentrega republicar em vez de ignorar (FR-014).
func (t Transacao) AnuncioPendente() bool {
	return t.Status.Anunciavel() && !t.ResultadoAnunciado
}
