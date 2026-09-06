package transacao

import (
	"errors"
	"testing"
	"time"
)

var agora = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

func nova() Transacao {
	return Nova("t1", "r1", "u1", "84.00", agora.Add(10*time.Minute), agora)
}

func TestNovaNasceAguardandoForma(t *testing.T) {
	tr := nova()
	if tr.Status != AguardandoForma {
		t.Fatalf("esperava AGUARDANDO_FORMA, veio %s", tr.Status)
	}
	if tr.FormaPagamento != "" {
		t.Fatalf("a forma é escolhida depois; veio %q", tr.FormaPagamento)
	}
	if !tr.ExpiraEm.Equal(agora.Add(10 * time.Minute)) {
		t.Fatalf("o prazo da reserva deveria ser guardado, veio %v", tr.ExpiraEm)
	}
	if tr.ResultadoAnunciado {
		t.Fatal("transação nova não pode nascer anunciada")
	}
	if tr.PagoEm != nil {
		t.Fatal("transação nova não pode ter instante de pagamento")
	}
}

func TestEscolherFormaVaiParaCobranca(t *testing.T) {
	tr := nova()
	if err := tr.EscolherForma(PIX, agora); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if tr.Status != Processando || tr.FormaPagamento != PIX {
		t.Fatalf("esperava PROCESSANDO com PIX, veio %s / %q", tr.Status, tr.FormaPagamento)
	}
}

func TestEscolherFormaRecusaFormaDesconhecida(t *testing.T) {
	tr := nova()
	if err := tr.EscolherForma("BOLETO", agora); !errors.Is(err, ErrFormaDesconhecida) {
		t.Fatalf("erro = %v, esperado ErrFormaDesconhecida", err)
	}
	if tr.Status != AguardandoForma {
		t.Fatalf("uma forma recusada não deveria mover a transação, veio %s", tr.Status)
	}
}

// A escolha tardia não cobra: a reserva vencida já liberou as poltronas do outro
// lado, e cobrar por ela seria cobrar por assento que outra pessoa pode ter
// levado.
//
// Ela recusa sem cancelar. Quem cancela por prazo é a varredura, num lugar só, e
// é ela que anuncia o desfecho — cancelar aqui deixaria um cancelamento que
// ninguém anuncia.
func TestEscolherFormaAposOPrazoRecusaSemCancelar(t *testing.T) {
	tr := nova()
	depoisDoPrazo := tr.ExpiraEm.Add(time.Second)

	if err := tr.EscolherForma(PIX, depoisDoPrazo); !errors.Is(err, ErrReservaExpirada) {
		t.Fatalf("erro = %v, esperado ErrReservaExpirada", err)
	}
	if tr.Status != AguardandoForma {
		t.Fatalf("a transação deveria continuar aguardando, para a varredura cancelá-la; veio %s", tr.Status)
	}
	if tr.FormaPagamento != "" {
		t.Fatalf("não se registra forma numa reserva vencida, veio %q", tr.FormaPagamento)
	}
}

func TestEscolherFormaDuasVezesRecusaASegunda(t *testing.T) {
	tr := nova()
	if err := tr.EscolherForma(PIX, agora); err != nil {
		t.Fatal(err)
	}
	if err := tr.EscolherForma(CartaoCredito, agora); !errors.Is(err, ErrFormaJaEscolhida) {
		t.Fatalf("erro = %v, esperado ErrFormaJaEscolhida", err)
	}
	if tr.FormaPagamento != PIX {
		t.Fatalf("a primeira escolha deveria prevalecer, veio %q", tr.FormaPagamento)
	}
}

func TestEscolherFormaEmEstadoFinalRecusa(t *testing.T) {
	for _, preparar := range []func(*Transacao){
		func(t *Transacao) { _ = t.EscolherForma(PIX, agora); _ = t.Aprovar("gw", agora) },
		func(t *Transacao) { _ = t.EscolherForma(PIX, agora); _ = t.Recusar(MotivoCartaoRecusado, agora) },
		func(t *Transacao) { _ = t.Cancelar(MotivoReservaExpirada, agora) },
	} {
		tr := nova()
		preparar(&tr)
		estado := tr.Status
		if err := tr.EscolherForma(CartaoCredito, agora); err == nil {
			t.Fatalf("de %s não deveria aceitar escolha de forma", estado)
		}
	}
}

// AGUARDANDO_FORMA admite transição: tratá-lo como terminal travaria a escolha.
func TestAguardandoFormaNaoEhTerminal(t *testing.T) {
	if AguardandoForma.Final() {
		t.Fatal("AGUARDANDO_FORMA não é estado terminal")
	}
	if AguardandoForma.Anunciavel() {
		t.Fatal("AGUARDANDO_FORMA não tem desfecho a anunciar")
	}
}

func TestTransicoesPermitidas(t *testing.T) {
	casos := []struct {
		nome     string
		aplicar  func(*Transacao) error
		esperado Status
	}{
		{"aprovar", func(tr *Transacao) error { return tr.Aprovar("gw-1", agora) }, Pago},
		{"recusar", func(tr *Transacao) error { return tr.Recusar(MotivoSaldoInsuficiente, agora) }, Recusado},
		{"cancelar", func(tr *Transacao) error { return tr.Cancelar(MotivoReservaExpirada, agora) }, Cancelado},
		{"verificacao", func(tr *Transacao) error { return tr.MarcarPendenteVerificacao(agora) }, PendenteVerificacao},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			tr := nova()
			if err := c.aplicar(&tr); err != nil {
				t.Fatalf("transição devia ser permitida: %v", err)
			}
			if tr.Status != c.esperado {
				t.Fatalf("esperava %s, veio %s", c.esperado, tr.Status)
			}
			if !tr.Status.Final() {
				t.Fatalf("%s devia ser terminal", tr.Status)
			}
		})
	}
}

func TestNenhumaTransicaoPartindoDeEstadoTerminal(t *testing.T) {
	terminais := []func(*Transacao) error{
		func(tr *Transacao) error { return tr.Aprovar("gw", agora) },
		func(tr *Transacao) error { return tr.Recusar(MotivoCartaoRecusado, agora) },
		func(tr *Transacao) error { return tr.Cancelar(MotivoReservaExpirada, agora) },
		func(tr *Transacao) error { return tr.MarcarPendenteVerificacao(agora) },
	}
	for i, levarAoFinal := range terminais {
		for j, tentarDeNovo := range terminais {
			tr := nova()
			if err := levarAoFinal(&tr); err != nil {
				t.Fatalf("preparação falhou: %v", err)
			}
			antes := tr.Status
			if err := tentarDeNovo(&tr); !errors.Is(err, ErrTransicaoInvalida) {
				t.Fatalf("caso %d→%d: esperava ErrTransicaoInvalida, veio %v", i, j, err)
			}
			if tr.Status != antes {
				t.Fatalf("caso %d→%d: estado mudou de %s para %s", i, j, antes, tr.Status)
			}
		}
	}
}

func TestAprovarRegistraCodigoEInstante(t *testing.T) {
	tr := nova()
	if err := tr.Aprovar("gw-42", agora); err != nil {
		t.Fatal(err)
	}
	if tr.CodigoTransacaoGateway != "gw-42" {
		t.Fatalf("código do adquirente não gravado: %q", tr.CodigoTransacaoGateway)
	}
	if tr.PagoEm == nil || !tr.PagoEm.Equal(agora) {
		t.Fatal("instante do pagamento não gravado (FR-012)")
	}
}

func TestAnunciabilidadePorEstado(t *testing.T) {
	casos := map[Status]bool{
		Processando:         false,
		Pago:                true,
		Recusado:            true,
		Cancelado:           true,
		PendenteVerificacao: false,
	}
	for s, esperado := range casos {
		if s.Anunciavel() != esperado {
			t.Fatalf("%s: esperava anunciável=%v", s, esperado)
		}
	}
}

func TestPendenteVerificacaoNuncaEAnunciada(t *testing.T) {
	tr := nova()
	if err := tr.MarcarPendenteVerificacao(agora); err != nil {
		t.Fatal(err)
	}
	if err := tr.MarcarAnunciado(agora); !errors.Is(err, ErrAnuncioInvalido) {
		t.Fatalf("esperava ErrAnuncioInvalido, veio %v", err)
	}
	if tr.ResultadoAnunciado {
		t.Fatal("PENDENTE_VERIFICACAO não pode ficar marcada como anunciada")
	}
	if tr.AnuncioPendente() {
		t.Fatal("PENDENTE_VERIFICACAO não tem anúncio pendente: ela nunca será anunciada")
	}
}

func TestAnuncioPendenteSoEnquantoNaoAnunciado(t *testing.T) {
	tr := nova()
	if tr.AnuncioPendente() {
		t.Fatal("PROCESSANDO não tem anúncio pendente")
	}
	if err := tr.Aprovar("gw", agora); err != nil {
		t.Fatal(err)
	}
	if !tr.AnuncioPendente() {
		t.Fatal("PAGO não anunciado devia ter anúncio pendente (FR-014)")
	}
	if err := tr.MarcarAnunciado(agora); err != nil {
		t.Fatal(err)
	}
	if tr.AnuncioPendente() {
		t.Fatal("depois de anunciado não há pendência")
	}
}

func TestExpiracao(t *testing.T) {
	prazo := agora
	casos := []struct {
		nome     string
		instante time.Time
		esperado bool
	}{
		{"um segundo antes", prazo.Add(-time.Second), false},
		{"exatamente no prazo", prazo, true},
		{"um segundo depois", prazo.Add(time.Second), true},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if Expirada(prazo, c.instante) != c.esperado {
				t.Fatalf("esperava expirada=%v", c.esperado)
			}
		})
	}
}

func TestFormaReconhecida(t *testing.T) {
	if !FormaReconhecida(PIX) || !FormaReconhecida(CartaoCredito) {
		t.Fatal("PIX e CARTAO_CREDITO devem ser reconhecidos (FR-004)")
	}
	for _, f := range []FormaPagamento{"", "BOLETO", "pix", "CRIPTO"} {
		if FormaReconhecida(f) {
			t.Fatalf("%q não devia ser reconhecida", f)
		}
	}
}
