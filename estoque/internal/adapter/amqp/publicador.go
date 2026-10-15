package amqp

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/estoque/internal/adapter/postgres"
)

type Publicador struct {
	Conexao *Conexao
	Banco   *postgres.Banco

	Intervalo time.Duration
	Lote      int
}

func (p *Publicador) Iniciar(ctx context.Context) {
	intervalo := p.Intervalo
	if intervalo <= 0 {
		intervalo = time.Second
	}
	lote := p.Lote
	if lote <= 0 {
		lote = 100
	}

	go func() {
		tique := time.NewTicker(intervalo)
		defer tique.Stop()
		for {
			select {
			case <-ctx.Done():

				return
			case <-tique.C:
				p.drenar(ctx, lote)
			}
		}
	}()
}

func (p *Publicador) Drenar(ctx context.Context, lote int) (int, error) {
	return p.Banco.PendentesParaPublicar(ctx, lote, func(fato postgres.FatoNaCaixa) error {
		return p.publicar(ctx, fato)
	})
}

func (p *Publicador) drenar(ctx context.Context, lote int) {
	_, _ = p.Drenar(ctx, lote)
}

func (p *Publicador) publicar(ctx context.Context, fato postgres.FatoNaCaixa) error {
	ctxPub, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()

	err := p.Conexao.canalPublica.PublishWithContext(ctxPub,
		Exchange, fato.RoutingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    fato.MessageID,
			Timestamp:    time.Now().UTC(),
			Body:         fato.Payload,
		})
	if err != nil {
		return err
	}

	select {
	case confirmacao := <-p.Conexao.confirmacoes:
		if !confirmacao.Ack {
			return errNaoConfirmado
		}
		return nil
	case <-ctxPub.Done():
		return ctxPub.Err()
	}
}

type erroPublicacao string

func (e erroPublicacao) Error() string { return string(e) }

const errNaoConfirmado = erroPublicacao("broker não confirmou a publicação")
