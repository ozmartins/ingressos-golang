//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/notificacao/internal/adapter/sistema"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

func (a *ambiente) emitir(t *testing.T) ingresso.Ingresso {
	t.Helper()
	reserva, usuario := uuid.NewString(), uuid.NewString()
	if _, err := a.caso(false).Executar(context.Background(), usecase.Anuncio{
		TransacaoID: uuid.NewString(), ReservaID: reserva, UsuarioID: usuario,
		PagoEm: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("emitir: %v", err)
	}
	lista, err := a.Ingressos.ListarPorUsuario(context.Background(), usuario, "")
	if err != nil || len(lista) != 1 {
		t.Fatalf("preparação falhou: %v (%d ingressos)", err, len(lista))
	}
	return lista[0]
}

func (a *ambiente) validador() usecase.ValidarIngresso {
	return usecase.ValidarIngresso{
		Ingressos: a.Ingressos, Assinador: a.Assinador,
		Relogio: sistema.Relogio{}, Log: a.Log,
	}
}

// SC-004 / FR-011: duas leituras simultâneas do mesmo código, uma autorização.
//
// A garantia mora no UPDATE condicionado contra um PostgreSQL real; em memória
// ela não seria prova de nada.
func TestLeiturasSimultaneasAutorizamUmaSo(t *testing.T) {
	a := subirAmbiente(t)
	i := a.emitir(t)
	u := a.validador()

	const n = 16
	vereditos := make(chan usecase.Veredito, n)
	pronto := make(chan struct{})
	var wg sync.WaitGroup
	for k := 0; k < n; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-pronto
			r, err := u.Executar(context.Background(), i.CodigoQR)
			if err != nil {
				t.Errorf("validação devolveu erro: %v", err)
				return
			}
			vereditos <- r.Veredito
		}()
	}
	close(pronto)
	wg.Wait()
	close(vereditos)

	autorizadas, reusos := 0, 0
	for v := range vereditos {
		switch v {
		case usecase.Autorizada:
			autorizadas++
		case usecase.Reuso:
			reusos++
		}
	}
	if autorizadas != 1 {
		t.Errorf("%d autorizações, queria exatamente 1 (SC-004)", autorizadas)
	}
	if reusos != n-1 {
		t.Errorf("%d recusas por reuso, queria %d", reusos, n-1)
	}
}

// FR-020: depois da baixa, nada além de status e instante mudou na linha.
func TestBaixaNaoAlteraCamposImutaveis(t *testing.T) {
	a := subirAmbiente(t)
	antes := a.emitir(t)

	if _, err := a.validador().Executar(context.Background(), antes.CodigoQR); err != nil {
		t.Fatalf("validar: %v", err)
	}

	depois, err := a.Ingressos.BuscarPorID(context.Background(), antes.ID)
	if err != nil {
		t.Fatal(err)
	}
	if depois.Status != ingresso.Utilizado {
		t.Fatalf("status = %q; a baixa não aconteceu, o teste não prova nada", depois.Status)
	}
	if depois.ReservaID != antes.ReservaID {
		t.Errorf("reserva_id mudou: %q → %q (FR-020)", antes.ReservaID, depois.ReservaID)
	}
	if depois.UsuarioID != antes.UsuarioID {
		t.Errorf("usuario_id mudou: %q → %q (FR-020)", antes.UsuarioID, depois.UsuarioID)
	}
	if depois.CodigoQR != antes.CodigoQR {
		t.Errorf("codigo_qr mudou (FR-020)")
	}
	if !depois.CriadoEm.Equal(antes.CriadoEm) {
		t.Errorf("criado_em mudou: %v → %v (FR-020)", antes.CriadoEm, depois.CriadoEm)
	}
}

// SC-005: código forjado não passa, e não deixa rastro no acervo.
func TestCodigoForjadoNaoEAceito(t *testing.T) {
	a := subirAmbiente(t)
	i := a.emitir(t)
	u := a.validador()

	forjados := []string{
		"CIN1.aaaa.bbbb",
		i.CodigoQR[:len(i.CodigoQR)-4] + "XXXX",
		"lixo",
	}
	for _, c := range forjados {
		r, err := u.Executar(context.Background(), c)
		if err != nil {
			t.Fatalf("validar %q: %v", c, err)
		}
		if r.Veredito != usecase.NaoEncontrado {
			t.Errorf("código forjado %q recebeu veredito %v (SC-005)", c, r.Veredito)
		}
	}

	depois, err := a.Ingressos.BuscarPorID(context.Background(), i.ID)
	if err != nil {
		t.Fatal(err)
	}
	if depois.Status != ingresso.Valido {
		t.Errorf("o ingresso legítimo foi consumido por tentativa de falsificação: %q", depois.Status)
	}
}
