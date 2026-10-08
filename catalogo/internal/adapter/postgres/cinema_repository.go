package postgres

import (
	"context"
	"fmt"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type CinemaRepository struct{ banco *Banco }

func NovoCinemaRepository(b *Banco) *CinemaRepository { return &CinemaRepository{banco: b} }

func (r cinemaRow) paraDominio() (catalogo.Cinema, error) {
	return catalogo.Cinema{
		ID: r.ID, Nome: r.Nome, Cidade: r.Cidade, Estado: r.Estado,
		Endereco: r.Endereco, Ativo: r.Ativo,
	}, nil
}

func (r *CinemaRepository) Listar(
	ctx context.Context,
	filtro usecase.FiltroCinemas,
	req shared.PageRequest,
) (shared.Page[catalogo.Cinema], error) {
	base := r.banco.conn(ctx).Model(&cinemaRow{})
	// Filtro nulo significa "qualquer situação".
	if filtro.Ativo != nil {
		base = base.Where("ativo = ?", *filtro.Ativo)
	}
	return consultarPaginado(base, "", "nome, id", req, cinemaRow.paraDominio)
}

func (r *CinemaRepository) BuscarPorID(ctx context.Context, cinemaID string) (catalogo.Cinema, error) {
	var linhas []cinemaRow
	if err := r.banco.conn(ctx).Where("id = ?", cinemaID).Limit(1).Find(&linhas).Error; err != nil {
		return catalogo.Cinema{}, fmt.Errorf("lendo cinema: %w", err)
	}
	if len(linhas) == 0 {
		return catalogo.Cinema{}, shared.NaoEncontrado("cinema", cinemaID)
	}
	return linhas[0].paraDominio()
}

func (r *CinemaRepository) Criar(ctx context.Context, c catalogo.Cinema) error {
	linha := cinemaRow{ID: c.ID, Nome: c.Nome, Cidade: c.Cidade, Estado: c.Estado,
		Endereco: c.Endereco, Ativo: c.Ativo}
	if err := r.banco.conn(ctx).Create(&linha).Error; err != nil {
		return fmt.Errorf("inserindo cinema: %w", err)
	}
	return nil
}

func (r *CinemaRepository) Atualizar(ctx context.Context, c catalogo.Cinema) error {
	res := r.banco.conn(ctx).Model(&cinemaRow{}).Where("id = ?", c.ID).Updates(map[string]any{
		"nome": c.Nome, "cidade": c.Cidade, "estado": c.Estado, "endereco": c.Endereco,
		"ativo": c.Ativo, "atualizado_em": gormAgora,
	})
	if res.Error != nil {
		return fmt.Errorf("atualizando cinema: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return shared.NaoEncontrado("cinema", c.ID)
	}
	return nil
}

func (r *CinemaRepository) Desativar(ctx context.Context, cinemaID string) error {
	res := r.banco.conn(ctx).Model(&cinemaRow{}).Where("id = ?", cinemaID).Updates(map[string]any{
		"ativo": false, "atualizado_em": gormAgora,
	})
	if res.Error != nil {
		return fmt.Errorf("desativando cinema: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return shared.NaoEncontrado("cinema", cinemaID)
	}
	return nil
}

func (r *CinemaRepository) Existe(ctx context.Context, cinemaID string) (bool, error) {
	var n int64
	if err := r.banco.conn(ctx).Model(&cinemaRow{}).Where("id = ?", cinemaID).Count(&n).Error; err != nil {
		return false, fmt.Errorf("verificando cinema: %w", err)
	}
	return n > 0, nil
}
