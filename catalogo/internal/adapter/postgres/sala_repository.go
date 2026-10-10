package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

const indiceNumeroDaSalaAtiva = "idx_salas_cinema_numero_ativa"

type SalaRepository struct{ banco *Banco }

func NovoSalaRepository(b *Banco) *SalaRepository { return &SalaRepository{banco: b} }

type fileiraJSON struct {
	Fileira  string `json:"fileira"`
	Assentos int    `json:"assentos"`
	Tipo     string `json:"tipo"`
}

func (r salaRow) paraDominio() (catalogo.Sala, error) {
	var layout []fileiraJSON
	if err := json.Unmarshal(r.Layout, &layout); err != nil {
		return catalogo.Sala{}, fmt.Errorf("lendo sala: layout inválido: %w", err)
	}
	fileiras := make([]catalogo.Fileira, 0, len(layout))
	for _, f := range layout {
		fileiras = append(fileiras, catalogo.Fileira{
			Letra: f.Fileira, Assentos: f.Assentos, Tipo: catalogo.TipoPoltrona(f.Tipo)})
	}
	return catalogo.Sala{
		ID: r.ID, CinemaID: r.CinemaID, Numero: r.Numero,
		TipoTela: catalogo.TipoTela(r.TipoTela), Layout: catalogo.LayoutSala{Fileiras: fileiras},
		Ativo: r.Ativo,
	}, nil
}

// A checagem prévia de NumeroEmUso não fecha a corrida entre duas requisições:
// quem perde cai no índice único, e isso é conflito de negócio, não falha de infra.
func erroDeEscritaDeSala(err error, acao string, numero int) error {
	if violaRestricao(err, indiceNumeroDaSalaAtiva) {
		return fmt.Errorf("%w: o cinema já tem uma sala ativa de número %d", shared.ErrConflito, numero)
	}
	return fmt.Errorf("%s sala: %w", acao, err)
}

func layoutParaJSON(l catalogo.LayoutSala) ([]byte, error) {
	fileiras := make([]fileiraJSON, 0, len(l.Fileiras))
	for _, f := range l.Fileiras {
		fileiras = append(fileiras, fileiraJSON{
			Fileira: f.Letra, Assentos: f.Assentos, Tipo: string(f.Tipo)})
	}
	return json.Marshal(fileiras)
}

func (r *SalaRepository) Listar(
	ctx context.Context,
	filtro usecase.FiltroSalas,
	req shared.PageRequest,
) (shared.Page[catalogo.Sala], error) {
	base := r.banco.conn(ctx).Model(&salaRow{})
	if filtro.Ativo != nil {
		base = base.Where("ativo = ?", *filtro.Ativo)
	}
	if filtro.CinemaID != "" {
		base = base.Where("cinema_id = ?", filtro.CinemaID)
	}
	return consultarPaginado(base, "", "cinema_id, numero, id", req, salaRow.paraDominio)
}

func (r *SalaRepository) BuscarPorID(ctx context.Context, salaID string) (catalogo.Sala, error) {
	var linhas []salaRow
	if err := r.banco.conn(ctx).Where("id = ?", salaID).Limit(1).Find(&linhas).Error; err != nil {
		return catalogo.Sala{}, fmt.Errorf("lendo sala: %w", err)
	}
	if len(linhas) == 0 {
		return catalogo.Sala{}, shared.NaoEncontrado("sala", salaID)
	}
	return linhas[0].paraDominio()
}

func (r *SalaRepository) Criar(ctx context.Context, s catalogo.Sala) error {
	layout, err := layoutParaJSON(s.Layout)
	if err != nil {
		return fmt.Errorf("inserindo sala: %w", err)
	}
	linha := salaRow{ID: s.ID, CinemaID: s.CinemaID, Numero: s.Numero,
		TipoTela: string(s.TipoTela), Layout: layout, Ativo: s.Ativo}
	if err := r.banco.conn(ctx).Create(&linha).Error; err != nil {
		return erroDeEscritaDeSala(err, "inserindo", s.Numero)
	}
	return nil
}

func (r *SalaRepository) Atualizar(ctx context.Context, s catalogo.Sala) error {
	layout, err := layoutParaJSON(s.Layout)
	if err != nil {
		return fmt.Errorf("atualizando sala: %w", err)
	}
	res := r.banco.conn(ctx).Model(&salaRow{}).Where("id = ?", s.ID).Updates(map[string]any{
		"numero": s.Numero, "tipo_tela": string(s.TipoTela), "layout": layout,
		"ativo": s.Ativo, "atualizado_em": gormAgora,
	})
	if res.Error != nil {
		return erroDeEscritaDeSala(res.Error, "atualizando", s.Numero)
	}
	if res.RowsAffected == 0 {
		return shared.NaoEncontrado("sala", s.ID)
	}
	return nil
}

func (r *SalaRepository) Desativar(ctx context.Context, salaID string) error {
	res := r.banco.conn(ctx).Model(&salaRow{}).Where("id = ?", salaID).Updates(map[string]any{
		"ativo": false, "atualizado_em": gormAgora,
	})
	if res.Error != nil {
		return fmt.Errorf("desativando sala: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return shared.NaoEncontrado("sala", salaID)
	}
	return nil
}

func (r *SalaRepository) NumeroEmUso(ctx context.Context, cinemaID string, numero int, excetoID string) (bool, error) {
	var n int64
	err := r.banco.conn(ctx).Model(&salaRow{}).
		Where("cinema_id = ? AND numero = ? AND ativo AND id <> ?", cinemaID, numero, excetoID).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("verificando número da sala: %w", err)
	}
	return n > 0, nil
}
