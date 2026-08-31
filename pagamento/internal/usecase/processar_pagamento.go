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

// ErrIntencaoInvalida marca anúncio que não pode virar cobrança (FR-003, FR-004).
// É falha definitiva: a mensagem vai para a quarentena, não volta para a fila.
var ErrIntencaoInvalida = errors.New("usecase: anúncio de reserva inválido")

// Intencao é o fato reserva.criada consumido (contracts/eventos.md §1).
// sessao_id e poltronas_ids são aceitos e ignorados: nenhum requisito os usa e
// este serviço não é dono desse estado.
type Intencao struct {
	Evento         string      `json:"evento"`
	ReservaID      string      `json:"reserva_id"`
	UsuarioID      string      `json:"usuario_id"`
	ValorTotal     json.Number `json:"valor_total"`
	FormaPagamento string      `json:"forma_pagamento"`
	ExpiraEm       string      `json:"expira_em"`
}

// Validada devolve os campos já convertidos, ou ErrIntencaoInvalida explicando
// o que falta. Toda recusa aqui acontece ANTES de qualquer escrita.
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

// Desfecho diz ao consumidor o que fazer com a mensagem.
type Desfecho int

const (
	// Confirmar — trabalho concluído; a mensagem sai da fila.
	Confirmar Desfecho = iota
	// Requeue — falha transitória ou disputa; a mensagem volta para a fila.
	Requeue
	// Quarentena — falha definitiva ou desfecho indeterminado; vai para a fila morta.
	Quarentena
)

// ProcessarPagamento é o caso de uso do consumo. A ordem de execução é
// invariável e é o que sustenta as garantias do serviço:
//
//	registrar PROCESSANDO → cobrar → gravar estado final → publicar → marcar → confirmar
//
// A mensagem NUNCA é confirmada antes da publicação (FR-014).
type ProcessarPagamento struct {
	Repo       Repositorio
	Adquirente Adquirente
	Publicador Publicador
	Relogio    Relogio
	IDs        GeradorID

	// PrazoAdquirente é o tempo máximo de espera por resposta do meio de
	// pagamento (FR-022). Estourado, o desfecho é indeterminado: não se sabe se
	// a cobrança foi efetivada. Zero desativa o prazo.
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

	// Reserva expirada: nenhuma cobrança é tentada (FR-005, clarificação Q5).
	if transacao.Expirada(expiraEm, agora) {
		if err := atual.Cancelar(transacao.MotivoReservaExpirada, agora); err != nil {
			return Requeue, err
		}
		return uc.finalizarEAnunciar(ctx, atual)
	}

	return uc.cobrarEResolver(ctx, atual)
}

// cobrarEResolver emite a cobrança e aplica o desfecho. Antes de falar com o
// adquirente marca a transação como "cobrança emitida"; se o adquirente devolver
// erro — que pelo contrato da porta significa que nada foi enviado — desfaz a
// marca, para que uma reentrega possa retomar com segurança (FR-008, FR-020).
func (uc ProcessarPagamento) cobrarEResolver(ctx context.Context, atual transacao.Transacao) (Desfecho, error) {
	ganhou, err := uc.Repo.ReivindicarCobranca(ctx, atual.ID, uc.Relogio.Agora())
	if err != nil {
		return Requeue, err
	}
	if !ganhou {
		// Outra execução detém o direito de cobrar esta reserva, ou ela já foi
		// finalizada. De todo modo, não é esta entrega que cobra.
		return Requeue, nil
	}

	// O prazo vive aqui, e não no adaptador: é política do caso de uso, e é o que
	// transforma uma espera indefinida no desfecho indeterminado da FR-022.
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

	// Prazo estourado é indeterminado, venha como desfecho ou como erro: o
	// adaptador pode devolver qualquer um dos dois, e a consequência é a mesma —
	// não se sabe se houve débito, então NÃO se libera o direito de cobrar.
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		res, err = ResultadoCobranca{Desfecho: Indeterminada}, nil
	}

	if err != nil {
		// O adquirente não recebeu a cobrança. Desfaz a marca e devolve a
		// intenção para a fila: a próxima entrega retoma do ponto certo.
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
		// Não se sabe se a cobrança foi efetivada. Nada é anunciado e nada é
		// recobrado; o caso vai para inspeção humana (FR-008, FR-022, D4).
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

// resolverConflito decide o que fazer quando a reserva já tem transação (FR-007).
func (uc ProcessarPagamento) resolverConflito(ctx context.Context, atual transacao.Transacao) (Desfecho, error) {
	switch {
	case atual.SeguroRetomar():
		// A execução anterior morreu antes de emitir a cobrança, ou o adquirente
		// devolveu erro. Nada foi cobrado: é seguro retomar daqui (FR-020).
		return uc.cobrarEResolver(ctx, atual)

	case !atual.Status.Final():
		// Cobrança emitida sem resposta conclusiva, ou outra entrega em curso.
		// Não se pode cobrar de novo nem decidir por ela: devolve à fila.
		// Persistindo, o limite de entregas leva à quarentena (FR-008).
		return Requeue, nil

	case atual.AnuncioPendente():
		// Estado final gravado, resultado nunca publicado — a janela que a
		// FR-014 existe para fechar. Publica a partir do que está gravado, sem
		// tocar no adquirente.
		return uc.anunciar(ctx, atual)

	default:
		// Já processada e já anunciada, ou PENDENTE_VERIFICACAO (que nunca é
		// anunciada). Nada a fazer.
		return Confirmar, nil
	}
}

func (uc ProcessarPagamento) finalizarEAnunciar(ctx context.Context, t transacao.Transacao) (Desfecho, error) {
	if err := uc.Repo.Finalizar(ctx, t); err != nil {
		if errors.Is(err, ErrJaFinalizada) {
			// Outro finalizou entre o nosso INSERT e agora. Releia e decida.
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
		// Publicação falhou: a mensagem NÃO é confirmada, e a transação fica
		// com anúncio pendente. A reentrega republica (FR-014).
		return Requeue, err
	}
	if err := uc.Repo.MarcarAnunciado(ctx, t.ID, uc.Relogio.Agora()); err != nil {
		// Já publicamos. Não confirmar leva a republicação na reentrega, que é
		// aceitável: o contrato declara entrega ao menos uma vez.
		return Requeue, err
	}
	return Confirmar, nil
}
