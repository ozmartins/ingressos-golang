package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type FilmeRepository struct{ pool *pgxpool.Pool }

func NovoFilmeRepository(p *pgxpool.Pool) *FilmeRepository { return &FilmeRepository{pool: p} }

const colunasFilme = `id, titulo, sinopse, duracao_minutos, classificacao_etaria,
                      genero, imagem_url, status`

func lerFilme(scan func(...any) error) (catalogo.Filme, error) {
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
}

func (r *FilmeRepository) Listar(
	ctx context.Context,
	filtro usecase.FiltroFilmes,
	publicos []catalogo.StatusFilme,
	req shared.PageRequest,
) (shared.Page[catalogo.Filme], error) {
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

	sqlPagina := `SELECT ` + colunasFilme + ` FROM filmes WHERE status = ANY($1)
	              ORDER BY titulo, id LIMIT $2 OFFSET $3`
	const sqlTotal = `SELECT COUNT(*) FROM filmes WHERE status = ANY($1)`

	return consultarPaginado(ctx, r.pool, sqlPagina, sqlTotal, filtros, req, lerFilme)
}

func (r *FilmeRepository) BuscarPorID(ctx context.Context, filmeID string) (catalogo.Filme, error) {
	linha := r.pool.QueryRow(ctx, `SELECT `+colunasFilme+` FROM filmes WHERE id = $1`, filmeID)
	f, err := lerFilme(linha.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalogo.Filme{}, shared.NaoEncontrado("filme", filmeID)
	}
	if err != nil {
		return catalogo.Filme{}, err
	}
	return f, nil
}

func (r *FilmeRepository) Criar(ctx context.Context, f catalogo.Filme) error {
	const sqlInserir = `INSERT INTO filmes (id, titulo, sinopse, duracao_minutos,
	                                        classificacao_etaria, genero, imagem_url, status)
	                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, sqlInserir, f.ID, f.Titulo, f.Sinopse, f.DuracaoMinutos,
		f.ClassificacaoEtaria, f.Genero, f.ImagemURL, string(f.Status))
	if err != nil {
		return fmt.Errorf("inserindo filme: %w", err)
	}
	return nil
}

func (r *FilmeRepository) Atualizar(ctx context.Context, f catalogo.Filme) error {
	const sqlAtualizar = `UPDATE filmes SET titulo = $2, sinopse = $3, duracao_minutos = $4,
	                          classificacao_etaria = $5, genero = $6, imagem_url = $7,
	                          status = $8, atualizado_em = CURRENT_TIMESTAMP
	                      WHERE id = $1`
	etiqueta, err := r.pool.Exec(ctx, sqlAtualizar, f.ID, f.Titulo, f.Sinopse, f.DuracaoMinutos,
		f.ClassificacaoEtaria, f.Genero, f.ImagemURL, string(f.Status))
	if err != nil {
		return fmt.Errorf("atualizando filme: %w", err)
	}
	if etiqueta.RowsAffected() == 0 {
		return shared.NaoEncontrado("filme", f.ID)
	}
	return nil
}

func (r *FilmeRepository) MarcarForaDeCartaz(ctx context.Context, filmeID string) error {
	const sqlRemover = `UPDATE filmes SET status = $2, atualizado_em = CURRENT_TIMESTAMP
	                    WHERE id = $1`
	etiqueta, err := r.pool.Exec(ctx, sqlRemover, filmeID, string(catalogo.StatusForaDeCartaz))
	if err != nil {
		return fmt.Errorf("removendo filme do cartaz: %w", err)
	}
	if etiqueta.RowsAffected() == 0 {
		return shared.NaoEncontrado("filme", filmeID)
	}
	return nil
}
