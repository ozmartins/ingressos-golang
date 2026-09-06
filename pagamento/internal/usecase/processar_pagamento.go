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

// O anúncio da reserva, como o Servico-Estoque o publica. A forma de pagamento
// não está aqui de propósito: ela é escolhida depois, por quem paga.
type Intencao struct {
	Evento     string      `json:"evento"`
	ReservaID  string      `json:"reserva_id"`
	UsuarioID  string      `json:"usuario_id"`
	ValorTotal json.Number `json:"valor_total"`
	ExpiraEm   string      `json:"expira_em"`
}

func (i Intencao) Validada() (valor string, expiraEm time.Time, err error) {
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

	if i.ExpiraEm == "" {
		problemas = append(problemas, "expira_em ausente")
	} else if t, e := time.Parse(time.RFC3339, i.ExpiraEm); e != nil {
		problemas = append(problemas, "expira_em não é RFC 3339: "+strconv.Quote(i.ExpiraEm))
	} else {
		expiraEm = t
	}

	if len(problemas) > 0 {
		return "", time.Time{}, fmt.Errorf("%w: %s", ErrIntencaoInvalida, strings.Join(problemas, "; "))
	}
	return valor, expiraEm, nil
}

type Desfecho int

const (
	Confirmar Desfecho = iota
	Requeue
	Quarentena
)

// RegistrarIntencao é o que o consumo de `reserva.criada` faz: grava a
// transação sabendo quanto cobrar e até quando, e para aí. A cobrança não sai
// daqui — ela depende de uma escolha que ainda não foi feita.
type RegistrarIntencao struct {
	Repo    Repositorio
	Relogio Relogio
	IDs     GeradorID
}

func (uc RegistrarIntencao) Executar(ctx context.Context, i Intencao) (Desfecho, error) {
	valor, expiraEm, err := i.Validada()
	if err != nil {
		return Quarentena, err
	}

	agora := uc.Relogio.Agora()
	nova := transacao.Nova(uc.IDs.Novo(), i.ReservaID, i.UsuarioID, valor, expiraEm, agora)

	// A idempotência é por `reserva_id`: o fato chega ao menos uma vez, e a
	// repetição não pode criar uma segunda transação nem reabrir a primeira.
	criada, _, err := uc.Repo.CriarSeAusente(ctx, nova)
	if err != nil {
		return Requeue, err
	}
	_ = criada
	return Confirmar, nil
}

// EscolherForma atende a escolha de quem paga. Não cobra: transiciona para
// PROCESSANDO e devolve, para a cobrança acontecer fora da requisição.
type EscolherForma struct {
	Repo    Repositorio
	Relogio Relogio
}

func (uc EscolherForma) Executar(ctx context.Context, reservaID, usuarioID string, forma transacao.FormaPagamento) (transacao.Transacao, error) {
	t, err := uc.Repo.BuscarPorReserva(ctx, reservaID)
	if err != nil {
		return transacao.Transacao{}, err
	}
	// Mesmo recorte da consulta: quem não é dono da reserva não a enxerga, e
	// não distinguimos "não existe" de "não é sua".
	if t.UsuarioID != usuarioID {
		return transacao.Transacao{}, ErrNaoEncontrada
	}

	if err := t.EscolherForma(forma, uc.Relogio.Agora()); err != nil {
		return transacao.Transacao{}, err
	}
	if err := uc.Repo.RegistrarEscolha(ctx, t); err != nil {
		return transacao.Transacao{}, err
	}
	return t, nil
}

type ProcessarPagamento struct {
	Repo       Repositorio
	Adquirente Adquirente
	Publicador Publicador
	Relogio    Relogio
	IDs        GeradorID

	PrazoAdquirente time.Duration
}

// Cobrar leva uma transação que já tem forma escolhida até o desfecho. É o
// varredor que a chama, fora do caminho da requisição: o cliente escolheu a
// forma e não espera o adquirente responder.
func (uc ProcessarPagamento) Cobrar(ctx context.Context, t transacao.Transacao) (Desfecho, error) {
	agora := uc.Relogio.Agora()

	// A reserva pode ter vencido entre a escolha e a cobrança.
	if transacao.Expirada(t.ExpiraEm, agora) && !t.CobrancaEmitida {
		if err := t.Cancelar(transacao.MotivoReservaExpirada, agora); err != nil {
			return Requeue, err
		}
		return uc.finalizarEAnunciar(ctx, t)
	}

	return uc.resolverConflito(ctx, t)
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
