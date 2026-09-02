package amqp

import (
	"context"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
)

type ErroDefinitivo struct{ Err error }

func (e ErroDefinitivo) Error() string { return e.Err.Error() }
func (e ErroDefinitivo) Unwrap() error { return e.Err }

func Definitivo(err error) error { return ErroDefinitivo{Err: err} }

type Manipulador func(ctx context.Context, msg amqp.Delivery) (desfecho string, err error)

type Consumidor struct {
	Conexao  *Conexao
	Fila     string
	Prefetch int
	Obs      *observability.Observabilidade
	Trata    Manipulador

	total   metric.Int64Counter
	duracao metric.Float64Histogram
}

func (c *Consumidor) Iniciar(ctx context.Context) error {
	if err := c.prepararMetricas(); err != nil {
		return err
	}

	canal, err := c.Conexao.Canal()
	if err != nil {
		return err
	}
	if err := canal.Qos(c.Prefetch, 0, false); err != nil {
		canal.Close()
		return err
	}

	entregas, err := canal.Consume(c.Fila, "", false, false, false, false, nil)
	if err != nil {
		canal.Close()
		return err
	}

	c.Obs.Log.Info("consumidor no ar", "fila", c.Fila, "prefetch", c.Prefetch)

	go func() {
		defer canal.Close()
		for {
			select {
			case <-ctx.Done():
				c.Obs.Log.Info("consumidor encerrado", "fila", c.Fila)
				return
			case msg, aberto := <-entregas:
				if !aberto {
					c.Obs.Log.Warn("canal de entregas fechado", "fila", c.Fila)
					return
				}
				c.processar(ctx, msg)
			}
		}
	}()
	return nil
}

func (c *Consumidor) processar(ctx context.Context, msg amqp.Delivery) {
	inicio := time.Now()

	ctxMsg := c.Obs.Propagador.Extract(ctx, carregadorDeHeaders(msg.Headers))
	ctxMsg, span := c.Obs.Tracer.Start(ctxMsg, "consumir "+c.Fila, trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	desfecho, err := c.Trata(ctxMsg, msg)

	switch {
	case err == nil:
		if errAck := msg.Ack(false); errAck != nil {
			c.Obs.Log.Error("falha ao confirmar mensagem", "fila", c.Fila, "erro", errAck.Error())
		}

	case errors.As(err, &ErroDefinitivo{}):
		desfecho = "dlq"
		c.Obs.Log.Warn("mensagem descartada para a fila-morta",
			"fila", c.Fila, "message_id", msg.MessageId, "motivo", err.Error())
		if errNack := msg.Nack(false, false); errNack != nil {
			c.Obs.Log.Error("falha ao descartar mensagem", "fila", c.Fila, "erro", errNack.Error())
		}

	default:
		desfecho = "requeue"
		c.Obs.Log.Warn("mensagem devolvida à fila",
			"fila", c.Fila, "message_id", msg.MessageId, "motivo", err.Error())
		time.Sleep(500 * time.Millisecond)
		if errNack := msg.Nack(false, true); errNack != nil {
			c.Obs.Log.Error("falha ao devolver mensagem", "fila", c.Fila, "erro", errNack.Error())
		}
	}

	atributos := metric.WithAttributes(
		observability.Atributo("fila", c.Fila),
		observability.Atributo("desfecho", desfecho),
	)
	c.total.Add(ctxMsg, 1, atributos)
	c.duracao.Record(ctxMsg, float64(time.Since(inicio).Microseconds())/1000.0, atributos)

	c.Obs.LogOperacao(ctxMsg, "consumir", desfecho, inicio,
		"fila", c.Fila, "message_id", msg.MessageId)
}

func (c *Consumidor) prepararMetricas() error {
	total, err := c.Obs.Medidor.Int64Counter("estoque.consumo.total",
		metric.WithDescription("mensagens consumidas por fila e desfecho"))
	if err != nil {
		return err
	}
	duracao, err := c.Obs.Medidor.Float64Histogram("estoque.consumo.duracao",
		metric.WithDescription("duração do processamento por fila"), metric.WithUnit("ms"))
	if err != nil {
		return err
	}
	c.total, c.duracao = total, duracao
	return nil
}

type carregadorDeHeaders amqp.Table

var _ propagation.TextMapCarrier = carregadorDeHeaders{}

func (c carregadorDeHeaders) Get(chave string) string {
	if v, ok := c[chave].(string); ok {
		return v
	}
	return ""
}

func (c carregadorDeHeaders) Set(chave, valor string) { c[chave] = valor }

func (c carregadorDeHeaders) Keys() []string {
	chaves := make([]string, 0, len(c))
	for k := range c {
		chaves = append(chaves, k)
	}
	return chaves
}
