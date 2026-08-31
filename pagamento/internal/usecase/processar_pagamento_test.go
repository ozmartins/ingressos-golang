package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

// ---------- T021: caminho feliz e as recusas ----------

func TestCobrancaAprovada(t *testing.T) {
	uc, repo, adq, pub := cenario(ResultadoCobranca{Desfecho: Aprovada, Codigo: "gw-9"})

	d, err := uc.Executar(context.Background(), intencaoValida())
	if err != nil || d != Confirmar {
		t.Fatalf("esperava Confirmar sem erro, veio %v / %v", d, err)
	}
	if adq.Cobrancas != 1 {
		t.Fatalf("esperava 1 cobrança, veio %d", adq.Cobrancas)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.Status != transacao.Pago || tr.CodigoTransacaoGateway != "gw-9" || tr.PagoEm == nil {
		t.Fatalf("transação mal gravada: %+v", tr)
	}
	if !tr.ResultadoAnunciado {
		t.Fatal("resultado devia estar marcado como anunciado (FR-014)")
	}
	if got := pub.rotas(); len(got) != 1 || got[0] != RoutingKeySucesso {
		t.Fatalf("esperava um pagamento.sucesso, veio %v", got)
	}
}

func TestCobrancaRecusada(t *testing.T) {
	uc, repo, _, pub := cenario(ResultadoCobranca{Desfecho: Recusada, Motivo: transacao.MotivoSaldoInsuficiente})

	if d, err := uc.Executar(context.Background(), intencaoValida()); err != nil || d != Confirmar {
		t.Fatalf("esperava Confirmar, veio %v / %v", d, err)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.Status != transacao.Recusado || tr.MotivoFalha != transacao.MotivoSaldoInsuficiente {
		t.Fatalf("esperava RECUSADO com motivo, veio %+v", tr)
	}
	if got := pub.rotas(); len(got) != 1 || got[0] != RoutingKeyFalhou {
		t.Fatalf("esperava um pagamento.falhou, veio %v", got)
	}
}

// Recusa sem motivo mapeado cai no motivo genérico do vocabulário fechado.
func TestRecusaSemMotivoUsaGenerico(t *testing.T) {
	uc, repo, _, _ := cenario(ResultadoCobranca{Desfecho: Recusada})
	if _, err := uc.Executar(context.Background(), intencaoValida()); err != nil {
		t.Fatal(err)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.MotivoFalha != transacao.MotivoRecusadoAdquirente {
		t.Fatalf("esperava motivo genérico, veio %q", tr.MotivoFalha)
	}
}

// FR-005 / clarificação Q5: reserva vencida não chega ao adquirente.
func TestReservaExpiradaNaoCobra(t *testing.T) {
	uc, repo, adq, pub := cenario(ResultadoCobranca{Desfecho: Aprovada})
	i := intencaoValida()
	i.ExpiraEm = prazoIdo.Format(time.RFC3339)

	if d, err := uc.Executar(context.Background(), i); err != nil || d != Confirmar {
		t.Fatalf("esperava Confirmar, veio %v / %v", d, err)
	}
	if adq.Cobrancas != 0 {
		t.Fatalf("reserva expirada NÃO pode ser cobrada, houve %d cobranças", adq.Cobrancas)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.Status != transacao.Cancelado || tr.MotivoFalha != transacao.MotivoReservaExpirada {
		t.Fatalf("esperava CANCELADO/RESERVA_EXPIRADA, veio %+v", tr)
	}
	if got := pub.rotas(); len(got) != 1 || got[0] != RoutingKeyFalhou {
		t.Fatalf("esperava pagamento.falhou, veio %v", got)
	}
}

// FR-003 / FR-004: cada forma de anúncio inválido vai para a quarentena sem
// criar transação e sem tocar no adquirente.
func TestAnuncioInvalidoVaiParaQuarentena(t *testing.T) {
	casos := map[string]func(*Intencao){
		"sem reserva":        func(i *Intencao) { i.ReservaID = "" },
		"sem usuario":        func(i *Intencao) { i.UsuarioID = "" },
		"sem valor":          func(i *Intencao) { i.ValorTotal = "" },
		"valor zero":         func(i *Intencao) { i.ValorTotal = "0.00" },
		"valor negativo":     func(i *Intencao) { i.ValorTotal = "-10.00" },
		"sem forma":          func(i *Intencao) { i.FormaPagamento = "" },
		"forma desconhecida": func(i *Intencao) { i.FormaPagamento = "BOLETO" },
		"sem prazo":          func(i *Intencao) { i.ExpiraEm = "" },
		"prazo malformado":   func(i *Intencao) { i.ExpiraEm = "ontem" },
	}
	for nome, quebrar := range casos {
		t.Run(nome, func(t *testing.T) {
			uc, repo, adq, pub := cenario(ResultadoCobranca{Desfecho: Aprovada})
			i := intencaoValida()
			quebrar(&i)

			d, err := uc.Executar(context.Background(), i)
			if d != Quarentena {
				t.Fatalf("esperava Quarentena, veio %v", d)
			}
			if !errors.Is(err, ErrIntencaoInvalida) {
				t.Fatalf("esperava ErrIntencaoInvalida, veio %v", err)
			}
			if adq.Cobrancas != 0 {
				t.Fatal("anúncio inválido não pode gerar cobrança")
			}
			if len(repo.porReserva) != 0 {
				t.Fatal("anúncio inválido não pode criar transação")
			}
			if len(pub.Publicados) != 0 {
				t.Fatal("anúncio inválido não pode ser anunciado")
			}
		})
	}
}

// A ordem invariável: gravar o estado final ANTES de publicar (FR-014).
func TestOrdemGravarPublicarMarcar(t *testing.T) {
	uc, repo, _, pub := cenario(ResultadoCobranca{Desfecho: Aprovada, Codigo: "gw"})
	pub.erro = errInfra

	d, err := uc.Executar(context.Background(), intencaoValida())
	if d != Requeue || !errors.Is(err, errInfra) {
		t.Fatalf("falha ao publicar deve devolver Requeue, veio %v / %v", d, err)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.Status != transacao.Pago {
		t.Fatal("o estado final deve estar gravado mesmo com a publicação falhando")
	}
	if tr.ResultadoAnunciado {
		t.Fatal("não se marca como anunciado o que não foi publicado")
	}
}

// Falha de infraestrutura do adquirente: nada decidido, mensagem volta à fila.
func TestAdquirenteIndisponivelDevolveAFila(t *testing.T) {
	uc, repo, _, pub := cenario(ResultadoCobranca{})
	uc.Adquirente = &adquirenteFalso{erro: errInfra}

	d, err := uc.Executar(context.Background(), intencaoValida())
	if d != Requeue || !errors.Is(err, errInfra) {
		t.Fatalf("esperava Requeue com o erro de infra, veio %v / %v", d, err)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.Status != transacao.Processando {
		t.Fatalf("transação devia seguir PROCESSANDO, veio %s", tr.Status)
	}
	if len(pub.Publicados) != 0 {
		t.Fatal("nada pode ser anunciado sem desfecho")
	}
}

// ---------- T038: o desfecho indeterminado (FR-022, SC-009) ----------

func TestDesfechoIndeterminadoNaoAnunciaEVaiParaQuarentena(t *testing.T) {
	uc, repo, adq, pub := cenario(ResultadoCobranca{Desfecho: Indeterminada})

	d, err := uc.Executar(context.Background(), intencaoValida())
	if err != nil {
		t.Fatalf("desfecho indeterminado não é erro do processamento: %v", err)
	}
	if d != Quarentena {
		t.Fatalf("esperava Quarentena, veio %v", d)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.Status != transacao.PendenteVerificacao {
		t.Fatalf("esperava PENDENTE_VERIFICACAO, veio %s", tr.Status)
	}
	if tr.ResultadoAnunciado {
		t.Fatal("PENDENTE_VERIFICACAO nunca é marcada como anunciada")
	}
	if len(pub.Publicados) != 0 {
		t.Fatalf("nenhum anúncio pode ser publicado, veio %v", pub.rotas())
	}
	if adq.Cobrancas != 1 {
		t.Fatalf("esperava exatamente uma tentativa, veio %d", adq.Cobrancas)
	}
}

// O prazo do adquirente precisa ser exercido de verdade: um adquirente LENTO
// (não um que devolve Indeterminada de mão beijada) tem de produzir o desfecho
// indeterminado. Este teste existe porque a versão anterior tratava o desfecho
// sem nunca aplicar o prazo que o gera — e todos os testes passavam, porque
// todos injetavam Indeterminada diretamente.
func TestPrazoDoAdquirenteProduzDesfechoIndeterminado(t *testing.T) {
	repo, pub := novoRepo(), &publicadorFalso{}
	uc := ProcessarPagamento{
		Repo: repo, Adquirente: adquirenteLento{},
		Publicador: pub, Relogio: relogioFixo{instante}, IDs: idsFixos{"t-1"},
		PrazoAdquirente: 50 * time.Millisecond,
	}

	d, err := uc.Executar(context.Background(), intencaoValida())
	if err != nil {
		t.Fatalf("prazo estourado não é erro do processamento: %v", err)
	}
	if d != Quarentena {
		t.Fatalf("esperava Quarentena, veio %v", d)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.Status != transacao.PendenteVerificacao {
		t.Fatalf("esperava PENDENTE_VERIFICACAO, veio %s", tr.Status)
	}
	// A marca de cobrança emitida NÃO pode ser liberada: não se sabe se houve
	// débito, e liberar permitiria uma recobrança (FR-008).
	if !tr.CobrancaEmitida {
		t.Fatal("prazo estourado não pode liberar o direito de cobrar")
	}
	if len(pub.Publicados) != 0 {
		t.Fatalf("nada pode ser anunciado, veio %v", pub.rotas())
	}
}

// Sem prazo configurado, o caso de uso não impõe deadline algum.
func TestSemPrazoConfiguradoNaoHaDeadline(t *testing.T) {
	repo := novoRepo()
	uc := ProcessarPagamento{
		Repo: repo, Adquirente: &adquirenteFalso{resultado: ResultadoCobranca{Desfecho: Aprovada, Codigo: "gw"}},
		Publicador: &publicadorFalso{}, Relogio: relogioFixo{instante}, IDs: idsFixos{"t-1"},
	}
	if _, err := uc.Executar(context.Background(), intencaoValida()); err != nil {
		t.Fatal(err)
	}
	tr, _ := repo.BuscarPorReserva(context.Background(), "r-1")
	if tr.Status != transacao.Pago {
		t.Fatalf("esperava PAGO, veio %s", tr.Status)
	}
}

// ---------- T026: o ramo de conflito, estado a estado (FR-007, FR-008, FR-014) ----------

func TestReentregaEmCadaEstado(t *testing.T) {
	base := func() transacao.Transacao {
		return transacao.Nova("t-existente", "r-1", "u-1", "84.00", transacao.PIX, instante)
	}
	casos := []struct {
		nome      string
		preparar  func() transacao.Transacao
		desfecho  Desfecho
		republica bool
		// cobrancas é quantas vezes o adquirente pode ser chamado. Zero em quase
		// todos os casos: reentrega não recobra. A exceção é a retomada segura.
		cobrancas int
	}{
		{
			// A execução anterior morreu antes de emitir a cobrança, ou o
			// adquirente devolveu erro. Nada foi cobrado: retomar é seguro, e é
			// o que faz uma falha transitória se completar (FR-020).
			nome:      "processando sem cobranca emitida: retoma com seguranca",
			preparar:  base,
			desfecho:  Confirmar,
			republica: true,
			cobrancas: 1,
		},
		{
			// Cobrança emitida sem resposta conclusiva: não se sabe se houve
			// débito. Não se recobra nem se decide (FR-008).
			nome: "processando com cobranca emitida: nao recobra, volta para a fila",
			preparar: func() transacao.Transacao {
				t := base()
				t.CobrancaEmitida = true
				return t
			},
			desfecho:  Requeue,
			republica: false,
			cobrancas: 0,
		},
		{
			nome: "pago e ja anunciado: nada a fazer",
			preparar: func() transacao.Transacao {
				t := base()
				_ = t.Aprovar("gw", instante)
				_ = t.MarcarAnunciado(instante)
				return t
			},
			desfecho:  Confirmar,
			republica: false,
		},
		{
			nome: "pago com anuncio pendente: republica",
			preparar: func() transacao.Transacao {
				t := base()
				_ = t.Aprovar("gw", instante)
				return t
			},
			desfecho:  Confirmar,
			republica: true,
		},
		{
			nome: "recusado com anuncio pendente: republica",
			preparar: func() transacao.Transacao {
				t := base()
				_ = t.Recusar(transacao.MotivoCartaoRecusado, instante)
				return t
			},
			desfecho:  Confirmar,
			republica: true,
		},
		{
			nome: "cancelado com anuncio pendente: republica",
			preparar: func() transacao.Transacao {
				t := base()
				_ = t.Cancelar(transacao.MotivoReservaExpirada, instante)
				return t
			},
			desfecho:  Confirmar,
			republica: true,
		},
		{
			nome: "pendente de verificacao: nunca republica",
			preparar: func() transacao.Transacao {
				t := base()
				_ = t.MarcarPendenteVerificacao(instante)
				return t
			},
			desfecho:  Confirmar,
			republica: false,
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			uc, repo, adq, pub := cenario(ResultadoCobranca{Desfecho: Aprovada, Codigo: "novo"})
			repo.semear(c.preparar())

			d, err := uc.Executar(context.Background(), intencaoValida())
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if d != c.desfecho {
				t.Fatalf("esperava %v, veio %v", c.desfecho, d)
			}
			// A garantia central: reentrega só chega ao adquirente quando é
			// comprovadamente seguro, e nunca mais de uma vez.
			if adq.Cobrancas != c.cobrancas {
				t.Fatalf("esperava %d cobranças, houve %d", c.cobrancas, adq.Cobrancas)
			}
			if c.republica && len(pub.Publicados) != 1 {
				t.Fatalf("esperava republicação, veio %d anúncios", len(pub.Publicados))
			}
			if !c.republica && len(pub.Publicados) != 0 {
				t.Fatalf("não devia republicar, veio %v", pub.rotas())
			}
		})
	}
}

// SC-002 no nível do caso de uso: entregas simultâneas cobram uma única vez.
//
// A asserção sobre anúncios é deliberadamente "um ou mais, todos idênticos", e
// não "exatamente um". Duas execuções concorrentes da mesma reserva podem ambas
// observar o estado final com anúncio pendente e publicar — é a contrapartida
// declarada de garantir que nenhum resultado se perca (FR-014, research.md D3),
// e o contrato promete entrega ao menos uma vez, com deduplicação por reserva_id
// no consumidor (contracts/eventos.md §4). O que NÃO pode variar é o dinheiro:
// uma cobrança, uma transação, e todo anúncio referindo a mesma transação.
func TestEntregasSimultaneasCobramUmaVez(t *testing.T) {
	uc, repo, adq, pub := cenario(ResultadoCobranca{Desfecho: Aprovada, Codigo: "gw"})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = uc.Executar(context.Background(), intencaoValida())
		}()
	}
	wg.Wait()

	if adq.Cobrancas != 1 {
		t.Fatalf("esperava exatamente 1 cobrança, veio %d", adq.Cobrancas)
	}
	if len(repo.porReserva) != 1 {
		t.Fatalf("esperava 1 transação, veio %d", len(repo.porReserva))
	}
	if len(pub.Publicados) == 0 {
		t.Fatal("o resultado precisa ser anunciado ao menos uma vez")
	}
	for _, f := range pub.Publicados {
		if f.MessageID != "t-1" || f.RoutingKey != RoutingKeySucesso {
			t.Fatalf("anúncio divergente entre as entregas: %+v", f)
		}
	}
}

// ---------- T022: conformidade dos fatos com contracts/eventos.md ----------

func TestFatoSucessoConformeContrato(t *testing.T) {
	tr := transacao.Nova("t-1", "r-1", "u-1", "84.00", transacao.PIX, instante)
	if err := tr.Aprovar("gw-1", instante); err != nil {
		t.Fatal(err)
	}
	f, err := MontarFato(tr)
	if err != nil {
		t.Fatal(err)
	}
	if f.RoutingKey != "pagamento.sucesso" || f.MessageID != "t-1" {
		t.Fatalf("roteamento errado: %+v", f)
	}

	var m map[string]any
	if err := json.Unmarshal(f.Payload, &m); err != nil {
		t.Fatal(err)
	}
	for _, campo := range []string{"evento", "versao", "ocorrido_em", "transacao_id",
		"reserva_id", "usuario_id", "valor_total", "pago_em"} {
		if _, ok := m[campo]; !ok {
			t.Fatalf("campo %q ausente no payload (contracts/eventos.md §2)", campo)
		}
	}
	if m["evento"] != "PAGAMENTO_SUCESSO" {
		t.Fatalf("evento errado: %v", m["evento"])
	}
	// valor_total é número no contrato, não texto.
	if !strings.Contains(string(f.Payload), `"valor_total":84.00`) {
		t.Fatalf("valor_total deve ser numérico e preservar centavos: %s", f.Payload)
	}
	if _, err := time.Parse(time.RFC3339, m["pago_em"].(string)); err != nil {
		t.Fatalf("pago_em não é RFC 3339: %v", m["pago_em"])
	}
}

func TestFatoFalhouConformeContrato(t *testing.T) {
	tr := transacao.Nova("t-1", "r-1", "u-1", "84.00", transacao.PIX, instante)
	if err := tr.Recusar(transacao.MotivoSaldoInsuficiente, instante); err != nil {
		t.Fatal(err)
	}
	f, err := MontarFato(tr)
	if err != nil {
		t.Fatal(err)
	}
	if f.RoutingKey != "pagamento.falhou" {
		t.Fatalf("roteamento errado: %s", f.RoutingKey)
	}
	var m map[string]any
	if err := json.Unmarshal(f.Payload, &m); err != nil {
		t.Fatal(err)
	}
	if m["evento"] != "PAGAMENTO_FALHOU" || m["motivo"] != "SALDO_INSUFICIENTE" {
		t.Fatalf("payload fora do contrato: %v", m)
	}
	// Compatibilidade com o consumidor real: o estoque lê reserva_id.
	if m["reserva_id"] != "r-1" {
		t.Fatalf("reserva_id é o que o Servico-Estoque consome: %v", m["reserva_id"])
	}
}

func TestPendenteVerificacaoNuncaViraFato(t *testing.T) {
	tr := transacao.Nova("t-1", "r-1", "u-1", "84.00", transacao.PIX, instante)
	if err := tr.MarcarPendenteVerificacao(instante); err != nil {
		t.Fatal(err)
	}
	if _, err := MontarFato(tr); err == nil {
		t.Fatal("PENDENTE_VERIFICACAO não pode virar fato publicável")
	}
}
