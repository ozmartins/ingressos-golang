package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

const (
	reserva1 = "9982a1b3-44c1-4221-a123-902183120192"
	usuario1 = "c394c8b3-76a1-4328-b803-02f5923b7a15"
	trans1   = "e402a129-8812-4211-b123-000129381293"
	pagoEm1  = "2026-08-29T21:35:10Z"
)

func anuncioValido() Anuncio {
	return Anuncio{TransacaoID: trans1, ReservaID: reserva1, UsuarioID: usuario1, PagoEm: pagoEm1}
}

type cenario struct {
	caso  EmitirIngresso
	repo  *ingressosFalsos
	avi   *avisosFalsos
	notif *notificadorFalso
}

func montar(t *testing.T) cenario {
	t.Helper()
	repo, avi, notif := novosIngressos(), &avisosFalsos{}, &notificadorFalso{}
	return cenario{
		caso: EmitirIngresso{
			Ingressos: repo, Avisos: avi, Notificador: notif,
			Assinador: assinadorFalso{}, Relogio: relogioFixo{}, IDs: &idsSequenciais{},
			Log: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		},
		repo: repo, avi: avi, notif: notif,
	}
}

func TestAnuncioValidoEmiteIngresso(t *testing.T) {
	c := montar(t)
	d, err := c.caso.Executar(context.Background(), anuncioValido())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if d != Confirmar {
		t.Errorf("desfecho = %v, queria Confirmar", d)
	}

	i, ok := c.repo.porReserva[reserva1]
	if !ok {
		t.Fatal("nenhum ingresso gravado para a reserva")
	}
	if i.Status != ingresso.Valido {
		t.Errorf("status = %q, queria VALIDO", i.Status)
	}
	if i.UsuarioID != usuario1 {
		t.Errorf("usuário = %q, queria %q", i.UsuarioID, usuario1)
	}
	if i.CodigoQR == "" {
		t.Error("ingresso emitido sem código de acesso")
	}
	if !i.CriadoEm.Equal(instanteFixo) {
		t.Errorf("criadoEm = %v, queria o instante do relógio injetado", i.CriadoEm)
	}
}

// FR-004: reentrega não cria segundo ingresso, não altera o existente, não
// erra — e, pela D6, não dispara aviso novo.
func TestReentregaEInerte(t *testing.T) {
	c := montar(t)
	ctx := context.Background()
	if _, err := c.caso.Executar(ctx, anuncioValido()); err != nil {
		t.Fatalf("primeira execução falhou: %v", err)
	}
	primeiro := c.repo.porReserva[reserva1]

	d, err := c.caso.Executar(ctx, anuncioValido())
	if err != nil {
		t.Fatalf("reentrega devolveu erro: %v", err)
	}
	if d != Confirmar {
		t.Errorf("desfecho da reentrega = %v, queria Confirmar", d)
	}
	if len(c.repo.porID) != 1 {
		t.Errorf("%d ingressos gravados, queria 1", len(c.repo.porID))
	}
	if c.repo.porReserva[reserva1].CodigoQR != primeiro.CodigoQR {
		t.Error("a reentrega alterou o código de acesso do ingresso existente")
	}
	if c.notif.vezes() != 1 {
		t.Errorf("notificador chamado %d vezes, queria 1 (a reentrega não avisa)", c.notif.vezes())
	}
	if len(c.avi.todos()) != 1 {
		t.Errorf("%d registros de aviso, queria 1", len(c.avi.todos()))
	}
}

// FR-004 sob concorrência: duas entregas simultâneas, um ingresso.
func TestEntregasSimultaneasEmitemUmSo(t *testing.T) {
	c := montar(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.caso.Executar(context.Background(), anuncioValido())
		}()
	}
	wg.Wait()
	if len(c.repo.porID) != 1 {
		t.Errorf("%d ingressos gravados, queria 1", len(c.repo.porID))
	}
}

// FR-002: defeito permanente vai para a quarentena, sem retentativa.
func TestAnuncioInvalidoVaiParaQuarentena(t *testing.T) {
	casos := map[string]Anuncio{
		"sem reserva":        {TransacaoID: trans1, UsuarioID: usuario1, PagoEm: pagoEm1},
		"sem usuario":        {TransacaoID: trans1, ReservaID: reserva1, PagoEm: pagoEm1},
		"sem transacao":      {ReservaID: reserva1, UsuarioID: usuario1, PagoEm: pagoEm1},
		"sem pago_em":        {TransacaoID: trans1, ReservaID: reserva1, UsuarioID: usuario1},
		"reserva malformada": {TransacaoID: trans1, ReservaID: "nao-e-uuid", UsuarioID: usuario1, PagoEm: pagoEm1},
		"usuario malformado": {TransacaoID: trans1, ReservaID: reserva1, UsuarioID: "123", PagoEm: pagoEm1},
		"pago_em invalido":   {TransacaoID: trans1, ReservaID: reserva1, UsuarioID: usuario1, PagoEm: "ontem"},
	}
	for nome, a := range casos {
		t.Run(nome, func(t *testing.T) {
			c := montar(t)
			d, err := c.caso.Executar(context.Background(), a)
			if d != Quarentena {
				t.Errorf("desfecho = %v, queria Quarentena", d)
			}
			if !errors.Is(err, ErrAnuncioInvalido) {
				t.Errorf("erro = %v, queria ErrAnuncioInvalido", err)
			}
			if len(c.repo.porID) != 0 {
				t.Error("ingresso foi emitido a partir de anúncio inválido")
			}
			if c.notif.vezes() != 0 {
				t.Error("aviso disparado a partir de anúncio inválido")
			}
		})
	}
}

func TestJSONIlegivelEDefeitoPermanente(t *testing.T) {
	if _, err := DecodificarAnuncio([]byte("{isto nao e json")); !errors.Is(err, ErrAnuncioInvalido) {
		t.Errorf("erro = %v, queria ErrAnuncioInvalido", err)
	}
}

// research D1: campos que o produtor publica a mais são tolerados.
func TestCamposExtrasSaoTolerados(t *testing.T) {
	a, err := DecodificarAnuncio([]byte(`{
      "evento":"PAGAMENTO_SUCESSO","versao":1,"ocorrido_em":"2026-08-29T21:35:10Z",
      "transacao_id":"` + trans1 + `","reserva_id":"` + reserva1 + `",
      "usuario_id":"` + usuario1 + `","valor_total":84.00,"pago_em":"` + pagoEm1 + `",
      "campo_que_ainda_nao_existe":"qualquer coisa"}`))
	if err != nil {
		t.Fatalf("decodificar falhou: %v", err)
	}
	c := montar(t)
	if d, err := c.caso.Executar(context.Background(), a); d != Confirmar || err != nil {
		t.Errorf("desfecho = %v, err = %v; queria Confirmar sem erro", d, err)
	}
}

// FR-022: falha transitória volta para a fila, não vai para a quarentena.
func TestFalhaTransitoriaPedeNovaTentativa(t *testing.T) {
	c := montar(t)
	c.repo.falharCriar = true

	d, err := c.caso.Executar(context.Background(), anuncioValido())
	if d != NovaTentativa {
		t.Errorf("desfecho = %v, queria NovaTentativa", d)
	}
	if !errors.Is(err, errBanco) {
		t.Errorf("erro = %v, queria o erro do banco", err)
	}
	if len(c.repo.porID) != 0 {
		t.Error("ingresso parcial ficou gravado após falha")
	}
}

// ---- User Story 4: o aviso (T042) ----

func TestAvisoEnviadoDeixaRegistro(t *testing.T) {
	c := montar(t)
	if _, err := c.caso.Executar(context.Background(), anuncioValido()); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	regs := c.avi.todos()
	if len(regs) != 1 {
		t.Fatalf("%d registros, queria 1", len(regs))
	}
	r := regs[0]
	if r.Desfecho != aviso.Enviado {
		t.Errorf("desfecho = %q, queria ENVIADO", r.Desfecho)
	}
	if r.Canal != aviso.Email {
		t.Errorf("canal = %q, queria EMAIL", r.Canal)
	}
	if r.IngressoID != c.repo.porReserva[reserva1].ID || r.UsuarioID != usuario1 {
		t.Error("registro não aponta para o ingresso e a pessoa certos")
	}
	if r.Detalhes != "" {
		t.Errorf("registro de envio bem-sucedido com detalhe: %q", r.Detalhes)
	}
}

// FR-018 e FR-025 — o teste mais importante desta feature.
//
// A falha do canal não pode desfazer a emissão (FR-018) NEM virar
// reprocessamento da mensagem (FR-025). Se este teste algum dia passar a
// aceitar Nack, a garantia morre em silêncio.
func TestFalhaDoAvisoNaoDerrubaEmissaoNemReprocessa(t *testing.T) {
	c := montar(t)
	c.notif.falhar = true

	d, err := c.caso.Executar(context.Background(), anuncioValido())

	if d != Confirmar {
		t.Errorf("desfecho = %v, queria Confirmar: falha de aviso NÃO pode reprocessar a mensagem (FR-025)", d)
	}
	if err != nil {
		t.Errorf("erro = %v; a falha do aviso não pode ser propagada (FR-025)", err)
	}

	i, ok := c.repo.porReserva[reserva1]
	if !ok {
		t.Fatal("a falha do aviso desfez a emissão (FR-018)")
	}
	if i.Status != ingresso.Valido {
		t.Errorf("status = %q; o ingresso deve permanecer VALIDO (FR-018)", i.Status)
	}

	regs := c.avi.todos()
	if len(regs) != 1 {
		t.Fatalf("%d registros, queria 1", len(regs))
	}
	if regs[0].Desfecho != aviso.Falha {
		t.Errorf("desfecho = %q, queria FALHA", regs[0].Desfecho)
	}
	if regs[0].Detalhes == "" {
		t.Error("registro de falha sem detalhe (FR-017)")
	}
	if !strings.Contains(regs[0].Detalhes, "recusou") {
		t.Errorf("o detalhe não conta o que aconteceu: %q", regs[0].Detalhes)
	}
	if !regs[0].PendenteDeReenvio() {
		t.Error("registro de falha não é identificável como pendente de reenvio (FR-018)")
	}
}

// Se nem o registro do aviso puder ser gravado, a emissão ainda assim vale.
func TestFalhaAoGravarRegistroNaoDerrubaEmissao(t *testing.T) {
	c := montar(t)
	c.avi.falharGrav = true

	d, err := c.caso.Executar(context.Background(), anuncioValido())
	if d != Confirmar || err != nil {
		t.Errorf("desfecho = %v, err = %v; queria Confirmar sem erro", d, err)
	}
	if _, ok := c.repo.porReserva[reserva1]; !ok {
		t.Error("a emissão foi perdida por causa do registro de aviso")
	}
}
