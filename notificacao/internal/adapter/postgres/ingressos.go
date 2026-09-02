package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

type Ingressos struct{ Pool *pgxpool.Pool }

const colunas = `id, reserva_id, usuario_id, codigo_qr, status, utilizado_em, criado_em`

func (r Ingressos) CriarSeAusente(ctx context.Context, i ingresso.Ingresso) (bool, ingresso.Ingresso, error) {
	const sql = `
		INSERT INTO ingressos_emitidos (` + colunas + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (reserva_id) DO NOTHING
		RETURNING ` + colunas

	linha := r.Pool.QueryRow(ctx, sql,
		i.ID, i.ReservaID, i.UsuarioID, i.CodigoQR, string(i.Status), i.UtilizadoEm, i.CriadoEm)

	criado, err := ler(linha)
	if err == nil {
		return true, criado, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, ingresso.Ingresso{}, fmt.Errorf("inserir ingresso: %w", err)
	}

	atual, err := r.buscarPorReserva(ctx, i.ReservaID)
	if err != nil {
		return false, ingresso.Ingresso{}, err
	}
	return false, atual, nil
}

func (r Ingressos) Utilizar(ctx context.Context, id string, agora time.Time) (bool, error) {
	const sql = `
		UPDATE ingressos_emitidos
		   SET status = 'UTILIZADO', utilizado_em = $2
		 WHERE id = $1 AND status = 'VALIDO'`

	tag, err := r.Pool.Exec(ctx, sql, id, agora)
	if err != nil {
		return false, fmt.Errorf("dar baixa no ingresso: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r Ingressos) BuscarPorID(ctx context.Context, id string) (ingresso.Ingresso, error) {
	const sql = `SELECT ` + colunas + ` FROM ingressos_emitidos WHERE id = $1`
	i, err := ler(r.Pool.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ingresso.Ingresso{}, usecase.ErrNaoEncontrado
	}
	if err != nil {
		return ingresso.Ingresso{}, fmt.Errorf("buscar ingresso: %w", err)
	}
	return i, nil
}

func (r Ingressos) ListarPorUsuario(ctx context.Context, usuarioID string, filtro ingresso.Status) ([]ingresso.Ingresso, error) {
	const sql = `
		SELECT ` + colunas + `
		  FROM ingressos_emitidos
		 WHERE usuario_id = $1
		   AND ($2 = '' OR status = $2)
		 ORDER BY criado_em DESC, id DESC`

	linhas, err := r.Pool.Query(ctx, sql, usuarioID, string(filtro))
	if err != nil {
		return nil, fmt.Errorf("listar ingressos: %w", err)
	}
	defer linhas.Close()

	lista := []ingresso.Ingresso{}
	for linhas.Next() {
		i, err := ler(linhas)
		if err != nil {
			return nil, fmt.Errorf("ler ingresso: %w", err)
		}
		lista = append(lista, i)
	}
	return lista, linhas.Err()
}

func (r Ingressos) buscarPorReserva(ctx context.Context, reservaID string) (ingresso.Ingresso, error) {
	const sql = `SELECT ` + colunas + ` FROM ingressos_emitidos WHERE reserva_id = $1`
	i, err := ler(r.Pool.QueryRow(ctx, sql, reservaID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ingresso.Ingresso{}, usecase.ErrNaoEncontrado
	}
	if err != nil {
		return ingresso.Ingresso{}, fmt.Errorf("buscar ingresso por reserva: %w", err)
	}
	return i, nil
}

type escaneavel interface{ Scan(dest ...any) error }

func ler(s escaneavel) (ingresso.Ingresso, error) {
	var i ingresso.Ingresso
	var status string
	err := s.Scan(&i.ID, &i.ReservaID, &i.UsuarioID, &i.CodigoQR, &status, &i.UtilizadoEm, &i.CriadoEm)
	i.Status = ingresso.Status(status)
	return i, err
}
