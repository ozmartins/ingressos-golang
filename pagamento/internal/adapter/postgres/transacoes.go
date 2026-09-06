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

const colunas = `id, reserva_id, usuario_id, valor_total::text, coalesce(forma_pagamento,''), status,
	coalesce(codigo_transacao_gateway,''), coalesce(motivo_falha,''),
	cobranca_emitida, resultado_anunciado, expira_em, pago_em, criado_em, atualizado_em`

func (r *Repositorio) CriarSeAusente(ctx context.Context, t transacao.Transacao) (bool, transacao.Transacao, error) {
	const sql = `
		INSERT INTO transacoes_pagamento
			(id, reserva_id, usuario_id, valor_total, status, expira_em, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $4::decimal, $5, $6, $7, $7)
		ON CONFLICT (reserva_id) DO NOTHING
		RETURNING ` + colunas

	// A forma fica de fora do INSERT: a transação nasce sem ela, e a invariante
	// `forma_so_apos_escolha` no banco recusaria qualquer valor aqui.
	linha := r.pool.QueryRow(ctx, sql,
		t.ID, t.ReservaID, t.UsuarioID, t.ValorTotal, string(t.Status), t.ExpiraEm, t.CriadoEm)

	criada, err := scan(linha)
	if err == nil {
		return true, criada, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, transacao.Transacao{}, err
	}

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

func (r *Repositorio) MarcarAnunciado(ctx context.Context, id string, agora time.Time) error {
	const sql = `
		UPDATE transacoes_pagamento
		   SET resultado_anunciado = true, atualizado_em = $2
		 WHERE id = $1 AND status IN ('PAGO','RECUSADO','CANCELADO')`
	_, err := r.pool.Exec(ctx, sql, id, agora)
	return err
}

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

func (r *Repositorio) LiberarCobranca(ctx context.Context, id string, agora time.Time) error {
	const sql = `
		UPDATE transacoes_pagamento
		   SET cobranca_emitida = false, atualizado_em = $2
		 WHERE id = $1 AND status = 'PROCESSANDO'`
	_, err := r.pool.Exec(ctx, sql, id, agora)
	return err
}

func (r *Repositorio) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

type linha interface {
	Scan(dest ...any) error
}

func scan(l linha) (transacao.Transacao, error) {
	var t transacao.Transacao
	var status, forma, codigo, motivo string
	err := l.Scan(&t.ID, &t.ReservaID, &t.UsuarioID, &t.ValorTotal, &forma, &status,
		&codigo, &motivo, &t.CobrancaEmitida, &t.ResultadoAnunciado, &t.ExpiraEm, &t.PagoEm,
		&t.CriadoEm, &t.AtualizadoEm)
	if err != nil {
		return transacao.Transacao{}, err
	}
	t.Status = transacao.Status(status)
	t.FormaPagamento = transacao.FormaPagamento(forma)
	t.CodigoTransacaoGateway = codigo
	t.MotivoFalha = transacao.Motivo(motivo)
	return t, nil
}

// A escolha é condicionada ao estado de origem: duas requisições simultâneas
// disputam a mesma linha, e só a que encontrar AGUARDANDO_FORMA vence.
func (r *Repositorio) RegistrarEscolha(ctx context.Context, t transacao.Transacao) error {
	const sql = `
		UPDATE transacoes_pagamento
		   SET forma_pagamento = $2, status = $3, motivo_falha = nullif($4,''),
		       atualizado_em = $5
		 WHERE id = $1 AND status = 'AGUARDANDO_FORMA'`

	etiqueta, err := r.pool.Exec(ctx, sql, t.ID, formaOuNulo(t), string(t.Status),
		string(t.MotivoFalha), t.AtualizadoEm)
	if err != nil {
		return err
	}
	if etiqueta.RowsAffected() == 0 {
		return usecase.ErrJaFinalizada
	}
	return nil
}

// Um cancelamento por prazo vencido não escolhe forma nenhuma: a coluna fica
// nula, e a invariante do banco admite isso só para o cancelamento.
func formaOuNulo(t transacao.Transacao) any {
	if t.FormaPagamento == "" {
		return nil
	}
	return string(t.FormaPagamento)
}

func (r *Repositorio) AguardandoCobranca(ctx context.Context, limite int) ([]transacao.Transacao, error) {
	const sql = `SELECT ` + colunas + ` FROM transacoes_pagamento
	              WHERE status = 'PROCESSANDO' AND NOT cobranca_emitida
	              ORDER BY criado_em
	              LIMIT $1`
	return consultarLista(ctx, r.pool, sql, limite)
}

func (r *Repositorio) CancelarEsperasVencidas(ctx context.Context, agora time.Time, limite int) ([]transacao.Transacao, error) {
	// O UPDATE ... RETURNING resolve num só passo: as linhas voltam já
	// canceladas, e é sobre elas que o anúncio é montado.
	const sql = `
		UPDATE transacoes_pagamento
		   SET status = 'CANCELADO', motivo_falha = $1, atualizado_em = $2
		 WHERE id IN (
		       SELECT id FROM transacoes_pagamento
		        WHERE status = 'AGUARDANDO_FORMA' AND expira_em <= $2
		        ORDER BY expira_em
		        LIMIT $3
		       FOR UPDATE SKIP LOCKED)
		RETURNING ` + colunas

	return consultarLista(ctx, r.pool, sql, string(transacao.MotivoReservaExpirada), agora, limite)
}

func (r *Repositorio) AnunciosPendentes(ctx context.Context, limite int) ([]transacao.Transacao, error) {
	// PENDENTE_VERIFICACAO fica de fora de propósito: ela não é anunciável, e
	// justamente por isso nunca sai desta consulta para ser republicada.
	const sql = `SELECT ` + colunas + ` FROM transacoes_pagamento
	              WHERE status IN ('PAGO','RECUSADO','CANCELADO') AND NOT resultado_anunciado
	              ORDER BY atualizado_em
	              LIMIT $1`
	return consultarLista(ctx, r.pool, sql, limite)
}

func consultarLista(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) ([]transacao.Transacao, error) {
	linhas, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	var lista []transacao.Transacao
	for linhas.Next() {
		t, err := scan(linhas)
		if err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, linhas.Err()
}
