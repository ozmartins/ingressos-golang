package amqp

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oseias/ingressos-golang/estoque/internal/adapter/postgres"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
)

// Publicador drena a caixa de saída. Roda fora do caminho da requisição: a
// resposta do bloqueio não espera a publicação (FR-009), e o fato sobrevive a um
// broker indisponível no instante da concessão (SC-005).
type Publicador struct {
	Conexao   *Conexao
	Banco     *postgres.Banco
	Obs       *observability.Observabilidade
	Intervalo time.Duration
	Lote      int
}

// Iniciar roda o laço de publicação até o contexto ser cancelado.
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
				p.Obs.Log.Info("publicador da caixa de saída encerrado")
				return
			case <-tique.C:
				p.drenar(ctx, lote)
			}
		}
	}()
}

// Drenar publica um lote imediatamente. Exposto para os testes e para o
// encerramento gracioso.
func (p *Publicador) Drenar(ctx context.Context, lote int) (int, error) {
	return p.Banco.PendentesParaPublicar(ctx, lote, func(fato postgres.FatoNaCaixa) error {
		return p.publicar(ctx, fato)
	})
}

func (p *Publicador) drenar(ctx context.Context, lote int) {
	n, err := p.Drenar(ctx, lote)
	if err != nil {
		p.Obs.Log.Warn("falha ao drenar a caixa de saída", "erro", err.Error())
		return
	}
	if n > 0 {
		p.Obs.Log.Info("fatos publicados", "quantidade", n)
	}
}

func (p *Publicador) publicar(ctx context.Context, fato postgres.FatoNaCaixa) error {
	// Reinjeta o contexto capturado quando o fato ocorreu — não o do instante
	// da publicação, que seria um span solto (FR-044).
	headers := amqp.Table{}
	for chave, valor := range fato.TraceContext {
		headers[chave] = valor
	}

	ctxPub, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()

	err := p.Conexao.canalPublica.PublishWithContext(ctxPub,
		Exchange, fato.RoutingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    fato.MessageID,
			Timestamp:    time.Now().UTC(),
			Headers:      headers,
			Body:         fato.Payload,
		})
	if err != nil {
		return err
	}

	// Só consideramos publicado depois do confirm do broker.
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
