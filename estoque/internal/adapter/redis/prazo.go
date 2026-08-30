// Package redis implementa o índice de prazo das reservas.
//
// Este adaptador NÃO participa da correção do bloqueio: a exclusividade mora na
// transação do PostgreSQL (research D2). O que ele entrega é pontualidade —
// uma chave por reserva com TTL, cuja expiração dispara a liberação em segundos
// em vez de esperar o próximo ciclo da varredura. Perder o Redis inteiro atrasa
// a liberação; nunca permite venda dupla.
package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
)

const prefixoChave = "reserva:"

// Indice é o índice de prazo.
type Indice struct {
	cliente *goredis.Client
	obs     *observability.Observabilidade
}

// Abrir conecta ao Redis e verifica que ele responde.
func Abrir(ctx context.Context, url string, obs *observability.Observabilidade) (*Indice, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("REDIS_URL malformada: %w", err)
	}
	cliente := goredis.NewClient(opts)
	if err := cliente.Ping(ctx).Err(); err != nil {
		cliente.Close()
		return nil, fmt.Errorf("redis não respondeu: %w", err)
	}
	return &Indice{cliente: cliente, obs: obs}, nil
}

// Marcar cria a chave de prazo da reserva com TTL até o vencimento.
func (i *Indice) Marcar(ctx context.Context, reservaID string, expiraEm time.Time) error {
	ttl := time.Until(expiraEm)
	if ttl <= 0 {
		return nil
	}
	return i.cliente.Set(ctx, prefixoChave+reservaID, "1", ttl).Err()
}

// Liberar apaga a chave de prazo (reserva finalizada por outro caminho).
func (i *Indice) Liberar(ctx context.Context, reservaID string) error {
	return i.cliente.Del(ctx, prefixoChave+reservaID).Err()
}

// Verificar diz se o Redis responde (usado pela prontidão como degradação).
func (i *Indice) Verificar(ctx context.Context) error { return i.cliente.Ping(ctx).Err() }

// Fechar encerra o cliente.
func (i *Indice) Fechar() { _ = i.cliente.Close() }

// EscutarExpiracoes assina as notificações de chave expirada e chama aoExpirar
// para cada reserva vencida.
//
// A entrega é best effort: notificação perdida em reinício ou desconexão não
// perde a reserva, porque a varredura no banco é a autoridade (D4).
func (i *Indice) EscutarExpiracoes(ctx context.Context, aoExpirar func(context.Context, string)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// __keyevent@N__:expired é o canal de chaves expiradas. Exige
			// notify-keyspace-events com o flag Ex no servidor.
			assinatura := i.cliente.PSubscribe(ctx, "__keyevent@*__:expired")
			canal := assinatura.Channel()

			i.obs.Log.Info("escutando expirações do índice de prazo")

		escuta:
			for {
				select {
				case <-ctx.Done():
					_ = assinatura.Close()
					return
				case msg, aberto := <-canal:
					if !aberto {
						break escuta
					}
					if !strings.HasPrefix(msg.Payload, prefixoChave) {
						continue
					}
					aoExpirar(ctx, strings.TrimPrefix(msg.Payload, prefixoChave))
				}
			}

			_ = assinatura.Close()
			i.obs.Log.Warn("assinatura de expirações caiu; reconectando",
				"nota", "a varredura periódica continua cobrindo a expiração")
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
}
