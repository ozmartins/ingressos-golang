package amqp

import (
	"context"

	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
)

const (
	intervaloPadrao = time.Second
	lotePadrao      = 100
	timeoutPublicar = 5 * time.Second
)

type Publicador struct {
	Conexao *Conexao
	Caixa   *postgres.CaixaDeSaida

	Intervalo time.Duration
	Lote      int
}

func (p *Publicador) Iniciar(ctx context.Context) {
	intervalo := p.Intervalo

	if intervalo <= 0 {
		intervalo = intervaloPadrao
	}

	lote := p.Lote

	if lote <= 0 {
		lote = lotePadrao
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

func (p *Publicador) drenar(ctx context.Context, lote int) {
	_, _ = p.Caixa.Drenar(ctx, lote, func(fato postgres.FatoNaCaixa) error {
		return p.publicar(ctx, fato)
	})
}

func (p *Publicador) publicar(ctx context.Context, fato postgres.FatoNaCaixa) error {
	canal, confirmacoes, err := p.Conexao.canalDePublicacao()

	if err != nil {
		return err
	}

	ctxPublicacao, cancelar := context.WithTimeout(ctx, timeoutPublicar)

	defer cancelar()

	err = canal.PublishWithContext(ctxPublicacao,
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
	case confirmacao := <-confirmacoes:
		if !confirmacao.Ack {
			return errNaoConfirmado
		}
		return nil
	case <-ctxPublicacao.Done():
		return ctxPublicacao.Err()
	}
}

type erroPublicacao string

func (e erroPublicacao) Error() string { return string(e) }

const errNaoConfirmado = erroPublicacao("broker não confirmou a publicação")
