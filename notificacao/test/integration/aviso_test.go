//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

// SC-007 / FR-018 / FR-025: com o canal falhando, o ingresso continua válido, o
// registro de falha fica com detalhe, e a MENSAGEM É CONFIRMADA — a fila
// principal esvazia e nada vai para a quarentena.
//
// É a asserção sobre a fila que distingue este teste de um teste de banco: sem
// ela, um Nack acidental passaria despercebido.
func TestFalhaDoAvisoNaoImpedeEntradaNemReprocessa(t *testing.T) {
	a := subirAmbiente(t)
	c := a.consumidor(t, true) // notificador em modo de falha

	ctx, parar := context.WithCancel(context.Background())
	defer parar()
	go func() { _ = c.Consumir(ctx) }()

	reserva, usuario := uuid.NewString(), uuid.NewString()
	a.publicar(t, reserva, usuario)

	esperar(t, 20*time.Second, "o ingresso ser emitido apesar da falha do aviso", func() bool {
		lista, err := a.Ingressos.ListarPorUsuario(context.Background(), usuario, "")
		return err == nil && len(lista) == 1
	})

	lista, err := a.Ingressos.ListarPorUsuario(context.Background(), usuario, "")
	if err != nil {
		t.Fatal(err)
	}
	if lista[0].Status != ingresso.Valido {
		t.Errorf("status = %q; a falha do aviso não pode invalidar o ingresso (FR-018)", lista[0].Status)
	}

	var status, detalhes string
	if err := a.Pool.QueryRow(context.Background(),
		`SELECT status, coalesce(detalhes,'') FROM registros_notificacao WHERE ingresso_id = $1`,
		lista[0].ID).Scan(&status, &detalhes); err != nil {
		t.Fatalf("ler registro de aviso: %v", err)
	}
	if status != string(aviso.Falha) {
		t.Errorf("status do registro = %q, queria FALHA", status)
	}
	if detalhes == "" {
		t.Error("registro de falha sem detalhe (FR-017)")
	}

	// A mensagem foi confirmada: nem na fila, nem na quarentena.
	esperar(t, 10*time.Second, "a fila principal esvaziar", func() bool {
		return a.contarFila(t, fila) == 0
	})
	if got := a.contarFila(t, filaDLQ); got != 0 {
		t.Errorf("%d mensagens na quarentena; falha de aviso NÃO pode reprocessar (FR-025)", got)
	}
}
