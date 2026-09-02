package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type FilmeRepository struct{ pool *pgxpool.Pool }

func NovoFilmeRepository(p *pgxpool.Pool) *FilmeRepository { return &FilmeRepository{pool: p} }

func (r *FilmeRepository) Listar(
	ctx context.Context,
	filtro usecase.FiltroFilmes,
	publicos []catalogo.StatusFilme,
	req shared.PageRequest,
) (shared.Page[catalogo.Filme], error) {
	const colunas = `id, titulo, sinopse, duracao_minutos, classificacao_etaria,
	                 genero, imagem_url, status`

	var filtros []any
	if filtro.Status != nil {
		filtros = []any{[]string{string(*filtro.Status)}}
	} else {
		lista := make([]string, len(publicos))
		for i, s := range publicos {
			lista[i] = string(s)
		}
		filtros = []any{lista}
	}

	sqlPagina := `SELECT ` + colunas + ` FROM filmes WHERE status = ANY($1)
	              ORDER BY titulo, id LIMIT $2 OFFSET $3`
	const sqlTotal = `SELECT COUNT(*) FROM filmes WHERE status = ANY($1)`

	return consultarPaginado(ctx, r.pool, sqlPagina, sqlTotal, filtros, req,
		func(scan func(...any) error) (catalogo.Filme, error) {
			var f catalogo.Filme
			var status string
			if err := scan(&f.ID, &f.Titulo, &f.Sinopse, &f.DuracaoMinutos,
				&f.ClassificacaoEtaria, &f.Genero, &f.ImagemURL, &status); err != nil {
				return f, fmt.Errorf("lendo filme: %w", err)
			}
			f.Status = catalogo.StatusFilme(status)
			if !f.Status.Valido() {
				return f, fmt.Errorf("filme %s tem status desconhecido %q", f.ID, status)
			}
			return f, nil
		})
}
