// Package postgres é o adaptador de persistência. O SQL é escrito à mão de
// propósito: o ON CONFLICT abaixo é a garantia central do serviço, e um ORM o
// esconderia (research.md D10).
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

type Repositorio struct{ pool *pgxpool.Pool }

func NovoRepositorio(pool *pgxpool.Pool) *Repositorio { return &Repositorio{pool: pool} }

const colunas = `id, reserva_id, usuario_id, valor_total::text, forma_pagamento, status,
	coalesce(codigo_transacao_gateway,''), coalesce(motivo_falha,''),
	cobranca_emitida, resultado_anunciado, pago_em, criado_em, atualizado_em`

// CriarSeAusente é a porta de entrada da idempotência (FR-006, research.md D2).
// Sob entregas simultâneas da mesma reserva, exatamente uma execução recebe
// criada=true; as demais recebem a transação vigente e NÃO devem cobrar.
func (r *Repositorio) CriarSeAusente(ctx context.Context, t transacao.Transacao) (bool, transacao.Transacao, error) {
	const sql = `
		INSERT INTO transacoes_pagamento
			(id, reserva_id, usuario_id, valor_total, forma_pagamento, status, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $4::decimal, $5, $6, $7, $7)
		ON CONFLICT (reserva_id) DO NOTHING
		RETURNING ` + colunas

	linha := r.pool.QueryRow(ctx, sql,
		t.ID, t.ReservaID, t.UsuarioID, t.ValorTotal, string(t.FormaPagamento), string(t.Status), t.CriadoEm)

	criada, err := scan(linha)
	if err == nil {
		return true, criada, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, transacao.Transacao{}, err
	}

	// Conflito: outra entrega chegou primeiro. Lê a vigente para decidir o que
	// fazer com esta mensagem (usecase.processarConflito).
	atual, err := r.BuscarPorReserva(ctx, t.ReservaID)
	if err != nil {
		return false, transacao.Transacao{}, err
	}
	return false, atual, nil
}

func (r *Repositorio) BuscarPorReserva(ctx context.Context, reservaID string) (transacao.Transacao, error) {
	const sql = `SELECT ` + colunas + ` FROM transacoes_pagamento WHERE reserva_id = $1`
	t, err := scan(r.pool.QueryRow(ctx, sql, reservaID))
	if errors.Is(err, pgx.ErrNoRows) {
		return transacao.Transacao{}, usecase.ErrNaoEncontrada
	}
	return t, err
}

// Finalizar grava o estado final. A cláusula WHERE condiciona a escrita a
// PROCESSANDO: se outro já finalizou, zero linhas são afetadas e devolvemos
// ErrJaFinalizada, para que o chamador releia em vez de sobrescrever.
func (r *Repositorio) Finalizar(ctx context.Context, t transacao.Transacao) error {
	const sql = `
		UPDATE transacoes_pagamento
		   SET status = $2,
		       codigo_transacao_gateway = nullif($3,''),
		       motivo_falha = nullif($4,''),
		       pago_em = $5,
		       atualizado_em = $6
		 WHERE id = $1 AND status = 'PROCESSANDO'`

	tag, err := r.pool.Exec(ctx, sql,
		t.ID, string(t.Status), t.CodigoTransacaoGateway, string(t.MotivoFalha), t.PagoEm, t.AtualizadoEm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return usecase.ErrJaFinalizada
	}
	return nil
}

// MarcarAnunciado fecha a janela da FR-014. Só vale a partir de estado final
// anunciável — a restrição CHECK do banco recusa o resto.
func (r *Repositorio) MarcarAnunciado(ctx context.Context, id string, agora time.Time) error {
	const sql = `
		UPDATE transacoes_pagamento
		   SET resultado_anunciado = true, atualizado_em = $2
		 WHERE id = $1 AND status IN ('PAGO','RECUSADO','CANCELADO')`
	_, err := r.pool.Exec(ctx, sql, id, agora)
	return err
}

// ReivindicarCobranca é o direito EXCLUSIVO de chamar o adquirente por esta
// transação. O UPDATE condicionado a cobranca_emitida = false é atômico no banco:
// sob entregas simultâneas, exatamente uma execução recebe true. Sem isso, duas
// entregas poderiam ambas ver "ninguém cobrou ainda" e cobrar a mesma reserva.
func (r *Repositorio) ReivindicarCobranca(ctx context.Context, id string, agora time.Time) (bool, error) {
	const sql = `
		UPDATE transacoes_pagamento
		   SET cobranca_emitida = true, atualizado_em = $2
		 WHERE id = $1 AND status = 'PROCESSANDO' AND cobranca_emitida = false`
	tag, err := r.pool.Exec(ctx, sql, id, agora)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// LiberarCobranca devolve o direito de cobrar. Só é chamada quando o adquirente
// devolve ERRO — que, pelo contrato da porta, significa que nada foi enviado.
// Nunca é chamada no desfecho indeterminado: lá não se sabe, e a reserva fica
// travada de propósito (FR-008).
func (r *Repositorio) LiberarCobranca(ctx context.Context, id string, agora time.Time) error {
	const sql = `
		UPDATE transacoes_pagamento
		   SET cobranca_emitida = false, atualizado_em = $2
		 WHERE id = $1 AND status = 'PROCESSANDO'`
	_, err := r.pool.Exec(ctx, sql, id, agora)
	return err
}

// Ping é a sonda de prontidão do armazenamento (FR-025).
func (r *Repositorio) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

type linha interface {
	Scan(dest ...any) error
}

func scan(l linha) (transacao.Transacao, error) {
	var t transacao.Transacao
	var status, forma, codigo, motivo string
	err := l.Scan(&t.ID, &t.ReservaID, &t.UsuarioID, &t.ValorTotal, &forma, &status,
		&codigo, &motivo, &t.CobrancaEmitida, &t.ResultadoAnunciado, &t.PagoEm, &t.CriadoEm, &t.AtualizadoEm)
	if err != nil {
		return transacao.Transacao{}, err
	}
	t.Status = transacao.Status(status)
	t.FormaPagamento = transacao.FormaPagamento(forma)
	t.CodigoTransacaoGateway = codigo
	t.MotivoFalha = transacao.Motivo(motivo)
	return t, nil
}
