package amqp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

type reconhecedorFalso struct {
	acks, nacks int
	requeue     bool
	ackErr      error
}

func (r *reconhecedorFalso) Ack(uint64, bool) error { r.acks++; return r.ackErr }
func (r *reconhecedorFalso) Nack(_ uint64, _ bool, requeue bool) error {
	r.nacks++
	r.requeue = requeue
	return nil
}
func (r *reconhecedorFalso) Reject(uint64, bool) error { return nil }

// repoFalso embute a interface para implementar só o que o consumidor usa.
type repoFalso struct {
	usecase.Repositorio
	errCriar error
}

func (r repoFalso) CriarSeAusente(_ context.Context, t transacao.Transacao) (bool, transacao.Transacao, error) {
	return r.errCriar == nil, t, r.errCriar
}

func (r repoFalso) BuscarPorReserva(context.Context, string) (transacao.Transacao, error) {
	return transacao.Transacao{}, usecase.ErrNaoEncontrada
}

type relogioFixo struct{}

func (relogioFixo) Agora() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }

type idFixo struct{}

func (idFixo) Novo() string { return "tx-1" }

const anuncioValido = `{"evento":"reserva.criada","reserva_id":"r-1","usuario_id":"u-1","valor_total":50.00,"expira_em":"2026-01-01T12:10:00Z"}`

func tratar(t *testing.T, corpo string, errCriar error) *reconhecedorFalso {
	t.Helper()
	ack := &reconhecedorFalso{}
	c := &Consumidor{
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Caso: usecase.RegistrarIntencao{
			Repo:    repoFalso{errCriar: errCriar},
			Relogio: relogioFixo{},
			IDs:     idFixo{},
		},
		EmAndamento: &Medidor{},
	}
	c.tratar(context.Background(), amqp.Delivery{Acknowledger: ack, Body: []byte(corpo)})
	return ack
}

func TestAnuncioValidoEhConfirmado(t *testing.T) {
	ack := tratar(t, anuncioValido, nil)
	if ack.acks != 1 || ack.nacks != 0 {
		t.Fatalf("esperava 1 ack e 0 nack, veio %d ack e %d nack", ack.acks, ack.nacks)
	}
}

func TestJSONIlegivelVaiParaQuarentena(t *testing.T) {
	ack := tratar(t, `{não é json`, nil)
	if ack.acks != 0 || ack.nacks != 1 || ack.requeue {
		t.Fatalf("esperava nack sem requeue, veio acks=%d nacks=%d requeue=%v", ack.acks, ack.nacks, ack.requeue)
	}
}

func TestAnuncioInvalidoVaiParaQuarentena(t *testing.T) {
	ack := tratar(t, `{"reserva_id":"r-1"}`, nil)
	if ack.acks != 0 || ack.nacks != 1 || ack.requeue {
		t.Fatalf("esperava nack sem requeue, veio acks=%d nacks=%d requeue=%v", ack.acks, ack.nacks, ack.requeue)
	}
}

func TestFalhaDeInfraEhDevolvidaParaNovaTentativa(t *testing.T) {
	ack := tratar(t, anuncioValido, errors.New("banco fora do ar"))
	if ack.acks != 0 || ack.nacks != 1 || !ack.requeue {
		t.Fatalf("esperava nack com requeue, veio acks=%d nacks=%d requeue=%v", ack.acks, ack.nacks, ack.requeue)
	}
}

func TestFalhaNoAckNaoPanica(t *testing.T) {
	ack := &reconhecedorFalso{ackErr: errors.New("canal fechado")}
	c := &Consumidor{
		Log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Caso: usecase.RegistrarIntencao{Repo: repoFalso{}, Relogio: relogioFixo{}, IDs: idFixo{}},
	}
	c.tratar(context.Background(), amqp.Delivery{Acknowledger: ack, Body: []byte(anuncioValido)})
	if ack.acks != 1 {
		t.Fatalf("esperava 1 tentativa de ack, veio %d", ack.acks)
	}
}
