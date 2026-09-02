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

func (r *SessaoRepository) Consultar(
	ctx context.Context,
	filtro usecase.FiltroSessoes,
	req shared.PageRequest,
) (shared.Page[catalogo.SessaoDetalhada], error) {
	condicoes := []string{"s.status = ANY($1)"}
	visiveis := make([]string, len(catalogo.StatusVisiveisNaGrade))
	for i, st := range catalogo.StatusVisiveisNaGrade {
		visiveis[i] = string(st)
	}
	filtros := []any{visiveis}

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
	visiveis := make([]string, len(catalogo.StatusVisiveisNaGrade))
	for i, st := range catalogo.StatusVisiveisNaGrade {
		visiveis[i] = string(st)
	}
	var bruto int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessoes WHERE status = ANY($1)`, visiveis).Scan(&bruto); err != nil {
		return
	}
	if bruto > totalResolvido {
		slog.WarnContext(ctx, "sessões omitidas da grade por referência inválida",
			slog.Int("omitidas", bruto-totalResolvido),
			slog.String("causa", "filme ou sala inexistente"))
	}
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
		return catalogo.Sessao{}, fmt.Errorf("%w: sessão %s", shared.ErrNaoEncontrado, sessaoID)
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
