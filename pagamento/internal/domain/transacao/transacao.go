package transacao

import (
	"errors"
	"time"
)

type Status string

const (
	// A transação nasce aqui: o valor já é conhecido, a forma de pagamento
	// ainda não. Reservar uma poltrona e escolher como pagar são decisões
	// distintas, e o serviço espera a segunda.
	AguardandoForma     Status = "AGUARDANDO_FORMA"
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
	ErrFormaJaEscolhida  = errors.New("transacao: forma de pagamento já escolhida")
	ErrReservaExpirada   = errors.New("transacao: prazo da reserva vencido")
	ErrFormaDesconhecida = errors.New("transacao: forma de pagamento desconhecida")
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
	// Prazo da reserva, vindo do fato. Guardado porque a escolha da forma
	// acontece depois: sem ele não há como saber se ainda dá tempo de cobrar.
	ExpiraEm     time.Time
	PagoEm       *time.Time
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

// A transação nasce sem forma de pagamento: o que o fato da reserva traz é o
// valor e o prazo. A forma chega depois, por escolha de quem vai pagar.
func Nova(id, reservaID, usuarioID, valor string, expiraEm, agora time.Time) Transacao {
	return Transacao{
		ID:           id,
		ReservaID:    reservaID,
		UsuarioID:    usuarioID,
		ValorTotal:   valor,
		Status:       AguardandoForma,
		ExpiraEm:     expiraEm,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}
}

// EscolherForma move a transação para a cobrança. Recusa a escolha tardia: uma
// reserva vencida já liberou as poltronas do outro lado, e cobrar por ela seria
// cobrar por assento que outra pessoa pode ter levado.
//
// Recusar é tudo o que ela faz nesse caso — não cancela. Quem cancela por prazo
// vencido é a varredura, num lugar só, e é ela que anuncia o desfecho a quem
// espera. Cancelar aqui também deixaria um cancelamento sem anúncio.
func (t *Transacao) EscolherForma(f FormaPagamento, agora time.Time) error {
	if t.Status != AguardandoForma {
		if t.Status == Processando || t.Status.Final() {
			return ErrFormaJaEscolhida
		}
		return ErrTransicaoInvalida
	}
	if !FormaReconhecida(f) {
		return ErrFormaDesconhecida
	}
	if Expirada(t.ExpiraEm, agora) {
		return ErrReservaExpirada
	}

	t.FormaPagamento = f
	t.Status = Processando
	t.AtualizadoEm = agora
	return nil
}

// Terminal é o que não admite mais transição. `AGUARDANDO_FORMA` e
// `PROCESSANDO` admitem: o primeiro espera a escolha, o segundo espera a
// cobrança.
func (s Status) Final() bool { return s != AguardandoForma && s != Processando }

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
