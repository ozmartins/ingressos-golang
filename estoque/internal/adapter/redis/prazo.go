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

type Indice struct {
	cliente *goredis.Client
	obs     *observability.Observabilidade
}

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

func (i *Indice) Marcar(ctx context.Context, reservaID string, expiraEm time.Time) error {
	ttl := time.Until(expiraEm)
	if ttl <= 0 {
		return nil
	}
	return i.cliente.Set(ctx, prefixoChave+reservaID, "1", ttl).Err()
}

func (i *Indice) Liberar(ctx context.Context, reservaID string) error {
	return i.cliente.Del(ctx, prefixoChave+reservaID).Err()
}

func (i *Indice) Verificar(ctx context.Context) error { return i.cliente.Ping(ctx).Err() }

func (i *Indice) Fechar() { _ = i.cliente.Close() }

func (i *Indice) EscutarExpiracoes(ctx context.Context, aoExpirar func(context.Context, string)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

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
