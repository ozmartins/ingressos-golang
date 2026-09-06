package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type SessaoRepository struct{ pool *pgxpool.Pool }

func NovoSessaoRepository(p *pgxpool.Pool) *SessaoRepository { return &SessaoRepository{pool: p} }

// O pgx não converte um `[]StatusSessao` para o `text[]` que o `ANY` espera.
func statusVisiveis() []string {
	visiveis := make([]string, len(catalogo.StatusVisiveisNaGrade))
	for i, st := range catalogo.StatusVisiveisNaGrade {
		visiveis[i] = string(st)
	}
	return visiveis
}

func (r *SessaoRepository) Consultar(
	ctx context.Context,
	filtro usecase.FiltroSessoes,
	req shared.PageRequest,
) (shared.Page[catalogo.SessaoDetalhada], error) {
	condicoes := []string{"s.status = ANY($1)"}
	filtros := []any{statusVisiveis()}

	if filtro.FilmeID != "" {
		filtros = append(filtros, filtro.FilmeID)
		condicoes = append(condicoes, fmt.Sprintf("s.filme_id = $%d", len(filtros)))
	}
	if filtro.CinemaID != "" {
		filtros = append(filtros, filtro.CinemaID)
		condicoes = append(condicoes, fmt.Sprintf("sa.cinema_id = $%d", len(filtros)))
	}
	if filtro.Data != nil {
		inicio := time.Date(filtro.Data.Ano, time.Month(filtro.Data.Mes), filtro.Data.Dia, 0, 0, 0, 0, time.UTC)
		filtros = append(filtros, inicio, inicio.AddDate(0, 0, 1))
		condicoes = append(condicoes, fmt.Sprintf("s.data_hora_inicio >= $%d AND s.data_hora_inicio < $%d", len(filtros)-1, len(filtros)))
	}

	const juncoes = `
		FROM sessoes s
		JOIN filmes  f  ON f.id  = s.filme_id
		JOIN salas   sa ON sa.id = s.sala_id
		JOIN cinemas c  ON c.id  = sa.cinema_id`

	onde := strings.Join(condicoes, " AND ")
	sqlPagina := fmt.Sprintf(`
		SELECT s.id, s.filme_id, f.titulo, c.id, c.nome, sa.numero, sa.tipo_tela,
		       s.data_hora_inicio, s.idioma, s.preco_base%s
		WHERE %s
		ORDER BY s.data_hora_inicio, s.id
		LIMIT $%d OFFSET $%d`, juncoes, onde, len(filtros)+1, len(filtros)+2)

	sqlTotal := fmt.Sprintf(`SELECT COUNT(*)%s WHERE %s`, juncoes, onde)

	pagina, err := consultarPaginado(ctx, r.pool, sqlPagina, sqlTotal, filtros, req,
		func(scan func(...any) error) (catalogo.SessaoDetalhada, error) {
			var (
				d     catalogo.SessaoDetalhada
				tipo  string
				idio  string
				preco pgtype.Numeric
			)
			if err := scan(&d.ID, &d.FilmeID, &d.FilmeTitulo, &d.CinemaID, &d.CinemaNome,
				&d.SalaNumero, &tipo, &d.DataHoraInicio, &idio, &preco); err != nil {
				return d, fmt.Errorf("lendo sessão: %w", err)
			}
			d.TipoTela, d.Idioma = catalogo.TipoTela(tipo), catalogo.Idioma(idio)
			var errPreco error
			if d.PrecoBase, errPreco = dinheiroDeNumeric(preco); errPreco != nil {
				return d, fmt.Errorf("sessão %s: %w", d.ID, errPreco)
			}
			d.DataHoraInicio = d.DataHoraInicio.UTC()
			return d, nil
		})
	if err != nil {
		return shared.Page[catalogo.SessaoDetalhada]{}, err
	}

	r.avisarSobreSessoesOrfas(ctx, filtro, pagina.Total)
	return pagina, nil
}

func (r *SessaoRepository) avisarSobreSessoesOrfas(ctx context.Context, filtro usecase.FiltroSessoes, totalResolvido int) {
	if filtro.CinemaID != "" || filtro.Data != nil || filtro.FilmeID != "" {
		return
	}
	var bruto int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessoes WHERE status = ANY($1)`, statusVisiveis()).Scan(&bruto); err != nil {
		return
	}
	if bruto > totalResolvido {
		slog.WarnContext(ctx, "sessões omitidas da grade por referência inválida",
			slog.Int("omitidas", bruto-totalResolvido),
			slog.String("causa", "filme ou sala inexistente"))
	}
}

// A sessão e o fato que a anuncia vão na mesma transação: se o processo morrer
// entre as duas escritas, nenhuma delas aconteceu.
func (r *SessaoRepository) Criar(ctx context.Context, s catalogo.Sessao, fato usecase.FatoPendente) error {
	return emTransacao(ctx, r.pool, func(tx pgx.Tx) error {
		const sqlInserir = `INSERT INTO sessoes (id, filme_id, sala_id, data_hora_inicio, idioma, preco_base, status)
		                    VALUES ($1, $2, $3, $4, $5, $6, $7)`
		if _, err := tx.Exec(ctx, sqlInserir, s.ID, s.FilmeID, s.SalaID, s.DataHoraInicio,
			string(s.Idioma), s.PrecoBase.String(), string(s.Status)); err != nil {
			return fmt.Errorf("inserindo sessão: %w", err)
		}
		return enfileirarFato(ctx, tx, fato)
	})
}

func (r *SessaoRepository) Atualizar(ctx context.Context, s catalogo.Sessao) error {
	const sqlAtualizar = `UPDATE sessoes SET filme_id = $2, sala_id = $3, data_hora_inicio = $4,
	                          idioma = $5, preco_base = $6, status = $7,
	                          atualizado_em = CURRENT_TIMESTAMP
	                      WHERE id = $1`
	etiqueta, err := r.pool.Exec(ctx, sqlAtualizar, s.ID, s.FilmeID, s.SalaID, s.DataHoraInicio,
		string(s.Idioma), s.PrecoBase.String(), string(s.Status))
	if err != nil {
		return fmt.Errorf("atualizando sessão: %w", err)
	}
	if etiqueta.RowsAffected() == 0 {
		return shared.NaoEncontrado("sessao", s.ID)
	}
	return nil
}

func (r *SessaoRepository) Cancelar(ctx context.Context, sessaoID string) error {
	const sqlCancelar = `UPDATE sessoes SET status = $2, atualizado_em = CURRENT_TIMESTAMP
	                     WHERE id = $1`
	etiqueta, err := r.pool.Exec(ctx, sqlCancelar, sessaoID, string(catalogo.SessaoCancelada))
	if err != nil {
		return fmt.Errorf("cancelando sessão: %w", err)
	}
	if etiqueta.RowsAffected() == 0 {
		return shared.NaoEncontrado("sessao", sessaoID)
	}
	return nil
}

// A duração de cada sessão concorrente é a do filme dela, então a janela sai do
// próprio SQL: nenhuma linha precisa subir para o Go só para ser descartada.
func (r *SessaoRepository) SalaOcupada(
	ctx context.Context,
	salaID string,
	inicio, fim time.Time,
	excetoID string,
) (bool, error) {
	const sql = `SELECT EXISTS(
	                 SELECT 1 FROM sessoes s JOIN filmes f ON f.id = s.filme_id
	                 WHERE s.sala_id = $1 AND s.status = ANY($2) AND s.id <> $3
	                   AND s.data_hora_inicio < $4
	                   AND s.data_hora_inicio + (f.duracao_minutos * INTERVAL '1 minute') > $5)`

	var ocupada bool
	err := r.pool.QueryRow(ctx, sql, salaID, statusVisiveis(), excetoID, fim, inicio).Scan(&ocupada)
	if err != nil {
		return false, fmt.Errorf("verificando ocupação da sala: %w", err)
	}
	return ocupada, nil
}

func (r *SessaoRepository) BuscarPorID(ctx context.Context, sessaoID string) (catalogo.Sessao, error) {
	const sql = `SELECT id, filme_id, sala_id, data_hora_inicio, idioma, preco_base, status
	             FROM sessoes WHERE id = $1`

	var (
		s     catalogo.Sessao
		idio  string
		st    string
		preco pgtype.Numeric
	)
	err := r.pool.QueryRow(ctx, sql, sessaoID).
		Scan(&s.ID, &s.FilmeID, &s.SalaID, &s.DataHoraInicio, &idio, &preco, &st)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalogo.Sessao{}, shared.NaoEncontrado("sessao", sessaoID)
	}
	if err != nil {
		return catalogo.Sessao{}, fmt.Errorf("buscando sessão: %w", err)
	}
	s.Idioma, s.Status = catalogo.Idioma(idio), catalogo.StatusSessao(st)
	if s.PrecoBase, err = dinheiroDeNumeric(preco); err != nil {
		return catalogo.Sessao{}, fmt.Errorf("sessão %s: %w", s.ID, err)
	}
	s.DataHoraInicio = s.DataHoraInicio.UTC()
	return s, nil
}
