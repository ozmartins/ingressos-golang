//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

func TestEntregasSimultaneasEmitemUmIngresso(t *testing.T) {
	a := subirAmbiente(t)
	caso := a.caso(false)
	reserva, usuario := uuid.NewString(), uuid.NewString()

	const n = 12
	criados := make(chan bool, n)
	pronto := make(chan struct{})
	var wg sync.WaitGroup
	for k := 0; k < n; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-pronto
			d, err := caso.Executar(context.Background(), usecase.Anuncio{
				TransacaoID: uuid.NewString(), ReservaID: reserva,
				UsuarioID: usuario, PagoEm: time.Now().UTC().Format(time.RFC3339),
			})
			criados <- (d == usecase.Confirmar && err == nil)
		}()
	}
	close(pronto)
	wg.Wait()
	close(criados)

	for ok := range criados {
		if !ok {
			t.Error("uma das entregas simultâneas não terminou em Confirmar")
		}
	}
	if got := a.contarIngressos(t); got != 1 {
		t.Errorf("%d ingressos gravados, queria exatamente 1 (SC-001)", got)
	}

	var avisos int
	if err := a.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM registros_notificacao`).Scan(&avisos); err != nil {
		t.Fatal(err)
	}
	if avisos != 1 {
		t.Errorf("%d registros de aviso, queria 1", avisos)
	}
}

func TestPercursoDoAnuncioAoIngresso(t *testing.T) {
	a := subirAmbiente(t)
	c := a.consumidor(t, false)

	ctx, parar := context.WithCancel(context.Background())
	defer parar()
	go func() { _ = c.Consumir(ctx) }()

	reserva, usuario := uuid.NewString(), uuid.NewString()
	a.publicar(t, reserva, usuario)

	esperar(t, 20*time.Second, "o ingresso aparecer para a pessoa", func() bool {
		lista, err := a.Ingressos.ListarPorUsuario(context.Background(), usuario, "")
		return err == nil && len(lista) == 1
	})

	lista, err := a.Ingressos.ListarPorUsuario(context.Background(), usuario, "")
	if err != nil {
		t.Fatal(err)
	}
	i := lista[0]
	if i.Status != ingresso.Valido {
		t.Errorf("status = %q, queria VALIDO", i.Status)
	}
	if i.ReservaID != reserva {
		t.Errorf("reserva = %q, queria %q", i.ReservaID, reserva)
	}
	id, err := a.Assinador.Verificar(i.CodigoQR)
	if err != nil || id != i.ID {
		t.Errorf("o código emitido não se verifica: id=%q err=%v", id, err)
	}

	esperar(t, 10*time.Second, "a fila principal esvaziar", func() bool {
		return a.contarFila(t, fila) == 0
	})
}

func TestAnuncioMalformadoVaiDiretoParaAQuarentena(t *testing.T) {
	a := subirAmbiente(t)
	c := a.consumidor(t, false)

	ctx, parar := context.WithCancel(context.Background())
	defer parar()
	go func() { _ = c.Consumir(ctx) }()

	a.publicarCru(t, []byte(`{"reserva_id":"nao-e-uuid"}`))

	esperar(t, 20*time.Second, "a mensagem chegar à fila morta", func() bool {
		return a.contarFila(t, filaDLQ) == 1
	})
	if got := a.contarIngressos(t); got != 0 {
		t.Errorf("%d ingressos emitidos a partir de anúncio malformado, queria 0 (SC-008)", got)
	}
	if got := a.contarFila(t, fila); got != 0 {
		t.Errorf("%d mensagens ainda na fila principal, queria 0", got)
	}
}

func TestFalhaTransitoriaRetentaAteOLimiteEEntaoVaiParaAQuarentena(t *testing.T) {
	a := subirAmbiente(t)
	c := a.consumidor(t, false)

	a.Pool.Close()

	ctx, parar := context.WithCancel(context.Background())
	defer parar()
	go func() { _ = c.Consumir(ctx) }()

	a.publicar(t, uuid.NewString(), uuid.NewString())

	esperar(t, 40*time.Second, "a mensagem esgotar as tentativas e cair na fila morta", func() bool {
		return a.contarFila(t, filaDLQ) == 1
	})
	if got := a.contarFila(t, fila); got != 0 {
		t.Errorf("%d mensagens ainda na fila principal", got)
	}
}
