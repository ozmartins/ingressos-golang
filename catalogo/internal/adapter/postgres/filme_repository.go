package postgres

import (
	"context"
	"fmt"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type FilmeRepository struct{ banco *Banco }

func NovoFilmeRepository(b *Banco) *FilmeRepository { return &FilmeRepository{banco: b} }

func (r filmeRow) paraDominio() (catalogo.Filme, error) {
	f := catalogo.Filme{
		ID: r.ID, Titulo: r.Titulo, Sinopse: r.Sinopse, DuracaoMinutos: r.DuracaoMinutos,
		ClassificacaoEtaria: r.ClassificacaoEtaria, Genero: r.Genero, ImagemURL: r.ImagemURL,
		Status: catalogo.StatusFilme(r.Status),
	}
	if !f.Status.Valido() {
		return f, fmt.Errorf("filme %s tem status desconhecido %q", f.ID, r.Status)
	}
	return f, nil
}

func (r *FilmeRepository) Listar(
	ctx context.Context,
	filtro usecase.FiltroFilmes,
	publicos []catalogo.StatusFilme,
	req shared.PageRequest,
) (shared.Page[catalogo.Filme], error) {
	var status []string
	if filtro.Status != nil {
		status = []string{string(*filtro.Status)}
	} else {
		status = make([]string, len(publicos))
		for i, s := range publicos {
			status[i] = string(s)
		}
	}

	base := r.banco.conn(ctx).Model(&filmeRow{}).Where("status IN ?", status)
	return consultarPaginado(base, "", "titulo, id", req, filmeRow.paraDominio)
}

func (r *FilmeRepository) BuscarPorID(ctx context.Context, filmeID string) (catalogo.Filme, error) {
	var linhas []filmeRow
	if err := r.banco.conn(ctx).Where("id = ?", filmeID).Limit(1).Find(&linhas).Error; err != nil {
		return catalogo.Filme{}, fmt.Errorf("lendo filme: %w", err)
	}
	if len(linhas) == 0 {
		return catalogo.Filme{}, shared.NaoEncontrado("filme", filmeID)
	}
	return linhas[0].paraDominio()
}

func (r *FilmeRepository) Criar(ctx context.Context, f catalogo.Filme) error {
	linha := filmeRow{
		ID: f.ID, Titulo: f.Titulo, Sinopse: f.Sinopse, DuracaoMinutos: f.DuracaoMinutos,
		ClassificacaoEtaria: f.ClassificacaoEtaria, Genero: f.Genero, ImagemURL: f.ImagemURL,
		Status: string(f.Status),
	}
	if err := r.banco.conn(ctx).Create(&linha).Error; err != nil {
		return fmt.Errorf("inserindo filme: %w", err)
	}
	return nil
}

func (r *FilmeRepository) Atualizar(ctx context.Context, f catalogo.Filme) error {
	res := r.banco.conn(ctx).Model(&filmeRow{}).Where("id = ?", f.ID).Updates(map[string]any{
		"titulo": f.Titulo, "sinopse": f.Sinopse, "duracao_minutos": f.DuracaoMinutos,
		"classificacao_etaria": f.ClassificacaoEtaria, "genero": f.Genero,
		"imagem_url": f.ImagemURL, "status": string(f.Status),
		"atualizado_em": gormAgora,
	})
	if res.Error != nil {
		return fmt.Errorf("atualizando filme: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return shared.NaoEncontrado("filme", f.ID)
	}
	return nil
}

func (r *FilmeRepository) MarcarForaDeCartaz(ctx context.Context, filmeID string) error {
	res := r.banco.conn(ctx).Model(&filmeRow{}).Where("id = ?", filmeID).Updates(map[string]any{
		"status": string(catalogo.StatusForaDeCartaz), "atualizado_em": gormAgora,
	})
	if res.Error != nil {
		return fmt.Errorf("removendo filme do cartaz: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return shared.NaoEncontrado("filme", filmeID)
	}
	return nil
}
