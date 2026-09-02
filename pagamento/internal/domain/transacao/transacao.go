package transacao

import (
	"errors"
	"time"
)

type Status string

const (
	Processando         Status = "PROCESSANDO"
	Pago                Status = "PAGO"
	Recusado            Status = "RECUSADO"
	Cancelado           Status = "CANCELADO"
	PendenteVerificacao Status = "PENDENTE_VERIFICACAO"
)

type Motivo string

const (
	MotivoReservaExpirada    Motivo = "RESERVA_EXPIRADA"
	MotivoSaldoInsuficiente  Motivo = "SALDO_INSUFICIENTE"
	MotivoCartaoRecusado     Motivo = "CARTAO_RECUSADO"
	MotivoRecusadoAdquirente Motivo = "RECUSADO_PELO_ADQUIRENTE"
)

type FormaPagamento string

const (
	PIX           FormaPagamento = "PIX"
	CartaoCredito FormaPagamento = "CARTAO_CREDITO"
)

func FormaReconhecida(f FormaPagamento) bool {
	return f == PIX || f == CartaoCredito
}

var (
	ErrTransicaoInvalida = errors.New("transacao: transição inválida a partir de estado terminal")
	ErrAnuncioInvalido   = errors.New("transacao: estado não é anunciável")
)

type Transacao struct {
	ID                     string
	ReservaID              string
	UsuarioID              string
	ValorTotal             string
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

func (s Status) Final() bool { return s != Processando }

func (s Status) Anunciavel() bool {
	return s == Pago || s == Recusado || s == Cancelado
}

func Expirada(expiraEm, agora time.Time) bool {
	return !agora.Before(expiraEm)
}

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

func (t *Transacao) Recusar(motivo Motivo, agora time.Time) error {
	if t.Status.Final() {
		return ErrTransicaoInvalida
	}
	t.Status = Recusado
	t.MotivoFalha = motivo
	t.AtualizadoEm = agora
	return nil
}

func (t *Transacao) Cancelar(motivo Motivo, agora time.Time) error {
	if t.Status.Final() {
		return ErrTransicaoInvalida
	}
	t.Status = Cancelado
	t.MotivoFalha = motivo
	t.AtualizadoEm = agora
	return nil
}

func (t *Transacao) MarcarPendenteVerificacao(agora time.Time) error {
	if t.Status.Final() {
		return ErrTransicaoInvalida
	}
	t.Status = PendenteVerificacao
	t.AtualizadoEm = agora
	return nil
}

func (t *Transacao) MarcarAnunciado(agora time.Time) error {
	if !t.Status.Anunciavel() {
		return ErrAnuncioInvalido
	}
	t.ResultadoAnunciado = true
	t.AtualizadoEm = agora
	return nil
}

func (t Transacao) SeguroRetomar() bool {
	return t.Status == Processando && !t.CobrancaEmitida
}

func (t Transacao) AnuncioPendente() bool {
	return t.Status.Anunciavel() && !t.ResultadoAnunciado
}
