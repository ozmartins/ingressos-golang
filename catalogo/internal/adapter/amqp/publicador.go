package amqp

import (
	"context"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/postgres"
)

const (
	intervaloPadrao = time.Second
	lotePadrao      = 100
	timeoutPublicar = 5 * time.Second
)

// Drena a caixa de saída fora do caminho da requisição. O broker fora do ar não
// impede a criação de uma sessão: o fato fica na caixa e sai quando ele voltar.
type Publicador struct {
	Conexao   *Conexao
	Caixa     *postgres.CaixaDeSaida
	Log       *slog.Logger
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
				p.Log.Info("publicador da caixa de saída encerrado")
				return
			case <-tique.C:
				p.drenar(ctx, lote)
			}
		}
	}()
}

func (p *Publicador) drenar(ctx context.Context, lote int) {
	var (
		falhas        int
		primeiraFalha error
	)
	n, err := p.Caixa.Drenar(ctx, lote, func(fato postgres.FatoNaCaixa) error {
		if err := p.publicar(ctx, fato); err != nil {
			falhas++
			if primeiraFalha == nil {
				primeiraFalha = err
			}
			return err
		}
		return nil
	})
	if err != nil {
		p.Log.Warn("falha ao drenar a caixa de saída", slog.Any("erro", err))
		return
	}
	// Um fato que não sai fica na caixa e volta no próximo tique. Sem este
	// registro a fila travaria em silêncio, com a contagem de tentativas subindo
	// e ninguém sabendo por quê.
	if falhas > 0 {
		p.Log.Warn("fatos não publicados; serão reenviados",
			slog.Int("quantidade", falhas), slog.Any("erro", primeiraFalha))
	}
	if n > 0 {
		p.Log.Info("fatos publicados", slog.Int("quantidade", n))
	}
}

func (p *Publicador) publicar(ctx context.Context, fato postgres.FatoNaCaixa) error {
	canal, confirmacoes, err := p.Conexao.canalDePublicacao()
	if err != nil {
		return err
	}

	cabecalhos := amqp.Table{}
	for chave, valor := range fato.TraceContext {
		cabecalhos[chave] = valor
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
			Headers:      cabecalhos,
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
