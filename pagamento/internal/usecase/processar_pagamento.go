package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

var ErrIntencaoInvalida = errors.New("usecase: anúncio de reserva inválido")

type Intencao struct {
	Evento         string      `json:"evento"`
	ReservaID      string      `json:"reserva_id"`
	UsuarioID      string      `json:"usuario_id"`
	ValorTotal     json.Number `json:"valor_total"`
	FormaPagamento string      `json:"forma_pagamento"`
	ExpiraEm       string      `json:"expira_em"`
}

func (i Intencao) Validada() (valor string, forma transacao.FormaPagamento, expiraEm time.Time, err error) {
	var problemas []string
	if i.ReservaID == "" {
		problemas = append(problemas, "reserva_id ausente")
	}
	if i.UsuarioID == "" {
		problemas = append(problemas, "usuario_id ausente")
	}

	valor = strings.TrimSpace(i.ValorTotal.String())
	if valor == "" {
		problemas = append(problemas, "valor_total ausente")
	} else if f, e := strconv.ParseFloat(valor, 64); e != nil || f <= 0 {
		problemas = append(problemas, "valor_total deve ser positivo, veio "+strconv.Quote(valor))
	}

	forma = transacao.FormaPagamento(i.FormaPagamento)
	if i.FormaPagamento == "" {
		problemas = append(problemas, "forma_pagamento ausente")
	} else if !transacao.FormaReconhecida(forma) {
		problemas = append(problemas, "forma_pagamento desconhecida: "+strconv.Quote(i.FormaPagamento))
	}

	if i.ExpiraEm == "" {
		problemas = append(problemas, "expira_em ausente")
	} else if t, e := time.Parse(time.RFC3339, i.ExpiraEm); e != nil {
		problemas = append(problemas, "expira_em não é RFC 3339: "+strconv.Quote(i.ExpiraEm))
	} else {
		expiraEm = t
	}

	if len(problemas) > 0 {
		return "", "", time.Time{}, fmt.Errorf("%w: %s", ErrIntencaoInvalida, strings.Join(problemas, "; "))
	}
	return valor, forma, expiraEm, nil
}

type Desfecho int

const (
	Confirmar Desfecho = iota
	Requeue
	Quarentena
)

type ProcessarPagamento struct {
	Repo       Repositorio
	Adquirente Adquirente
	Publicador Publicador
	Relogio    Relogio
	IDs        GeradorID

	PrazoAdquirente time.Duration
}

func (uc ProcessarPagamento) Executar(ctx context.Context, i Intencao) (Desfecho, error) {
	valor, forma, expiraEm, err := i.Validada()
	if err != nil {
		return Quarentena, err
	}

	agora := uc.Relogio.Agora()
	nova := transacao.Nova(uc.IDs.Novo(), i.ReservaID, i.UsuarioID, valor, forma, agora)

	criada, atual, err := uc.Repo.CriarSeAusente(ctx, nova)
	if err != nil {
		return Requeue, err
	}
	if !criada {
		return uc.resolverConflito(ctx, atual)
	}

	if transacao.Expirada(expiraEm, agora) {
		if err := atual.Cancelar(transacao.MotivoReservaExpirada, agora); err != nil {
			return Requeue, err
		}
		return uc.finalizarEAnunciar(ctx, atual)
	}

	return uc.cobrarEResolver(ctx, atual)
}

func (uc ProcessarPagamento) cobrarEResolver(ctx context.Context, atual transacao.Transacao) (Desfecho, error) {
	ganhou, err := uc.Repo.ReivindicarCobranca(ctx, atual.ID, uc.Relogio.Agora())
	if err != nil {
		return Requeue, err
	}
	if !ganhou {
		return Requeue, nil
	}

	ctxCobranca := ctx
	if uc.PrazoAdquirente > 0 {
		var cancelar context.CancelFunc
		ctxCobranca, cancelar = context.WithTimeout(ctx, uc.PrazoAdquirente)
		defer cancelar()
	}

	res, err := uc.Adquirente.Cobrar(ctxCobranca, Cobranca{
		TransacaoID:    atual.ID,
		ReservaID:      atual.ReservaID,
		ValorTotal:     atual.ValorTotal,
		FormaPagamento: atual.FormaPagamento,
	})

	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		res, err = ResultadoCobranca{Desfecho: Indeterminada}, nil
	}

	if err != nil {
		if e := uc.Repo.LiberarCobranca(ctx, atual.ID, uc.Relogio.Agora()); e != nil {
			return Requeue, e
		}
		return Requeue, err
	}

	agora := uc.Relogio.Agora()
	switch res.Desfecho {
	case Aprovada:
		err = atual.Aprovar(res.Codigo, agora)
	case Recusada:
		motivo := res.Motivo
		if motivo == "" {
			motivo = transacao.MotivoRecusadoAdquirente
		}
		err = atual.Recusar(motivo, agora)
	case Indeterminada:
		if err := atual.MarcarPendenteVerificacao(agora); err != nil {
			return Requeue, err
		}
		if err := uc.Repo.Finalizar(ctx, atual); err != nil {
			return Requeue, err
		}
		return Quarentena, nil
	}
	if err != nil {
		return Requeue, err
	}
	return uc.finalizarEAnunciar(ctx, atual)
}

func (uc ProcessarPagamento) resolverConflito(ctx context.Context, atual transacao.Transacao) (Desfecho, error) {
	switch {
	case atual.SeguroRetomar():
		return uc.cobrarEResolver(ctx, atual)

	case !atual.Status.Final():
		return Requeue, nil

	case atual.AnuncioPendente():
		return uc.anunciar(ctx, atual)

	default:
		return Confirmar, nil
	}
}

func (uc ProcessarPagamento) finalizarEAnunciar(ctx context.Context, t transacao.Transacao) (Desfecho, error) {
	if err := uc.Repo.Finalizar(ctx, t); err != nil {
		if errors.Is(err, ErrJaFinalizada) {
			atual, e := uc.Repo.BuscarPorReserva(ctx, t.ReservaID)
			if e != nil {
				return Requeue, e
			}
			return uc.resolverConflito(ctx, atual)
		}
		return Requeue, err
	}
	return uc.anunciar(ctx, t)
}

func (uc ProcessarPagamento) anunciar(ctx context.Context, t transacao.Transacao) (Desfecho, error) {
	fato, err := MontarFato(t)
	if err != nil {
		return Requeue, err
	}
	if err := uc.Publicador.Publicar(ctx, fato); err != nil {
		return Requeue, err
	}
	if err := uc.Repo.MarcarAnunciado(ctx, t.ID, uc.Relogio.Agora()); err != nil {
		return Requeue, err
	}
	return Confirmar, nil
}
