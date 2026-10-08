package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

type SessaoRepository struct{ banco *Banco }

func NovoSessaoRepository(b *Banco) *SessaoRepository { return &SessaoRepository{banco: b} }

func statusVisiveis() []string {
	visiveis := make([]string, len(catalogo.StatusVisiveisNaGrade))
	for i, st := range catalogo.StatusVisiveisNaGrade {
		visiveis[i] = string(st)
	}
	return visiveis
}

func (r sessaoRow) paraDominio() (catalogo.Sessao, error) {
	preco, err := dinheiroDeTexto(r.PrecoBase)
	if err != nil {
		return catalogo.Sessao{}, fmt.Errorf("sessão %s: %w", r.ID, err)
	}
	return catalogo.Sessao{
		ID: r.ID, FilmeID: r.FilmeID, SalaID: r.SalaID,
		DataHoraInicio: r.DataHoraInicio.UTC(), Idioma: catalogo.Idioma(r.Idioma),
		PrecoBase: preco, Status: catalogo.StatusSessao(r.Status),
	}, nil
}

func sessaoParaLinha(s catalogo.Sessao) sessaoRow {
	return sessaoRow{
		ID: s.ID, FilmeID: s.FilmeID, SalaID: s.SalaID, DataHoraInicio: s.DataHoraInicio,
		Idioma: string(s.Idioma), PrecoBase: s.PrecoBase.String(), Status: string(s.Status),
	}
}

func (r sessaoDetalhadaRow) paraDominio() (catalogo.SessaoDetalhada, error) {
	preco, err := dinheiroDeTexto(r.PrecoBase)
	if err != nil {
		return catalogo.SessaoDetalhada{}, fmt.Errorf("sessão %s: %w", r.ID, err)
	}
	return catalogo.SessaoDetalhada{
		ID: r.ID, FilmeID: r.FilmeID, FilmeTitulo: r.FilmeTitulo,
		CinemaID: r.CinemaID, CinemaNome: r.CinemaNome, SalaNumero: r.SalaNumero,
		TipoTela: catalogo.TipoTela(r.TipoTela), DataHoraInicio: r.DataHoraInicio.UTC(),
		Idioma: catalogo.Idioma(r.Idioma), PrecoBase: preco,
	}, nil
}

// A origem, as junções e os filtros da grade. É a mesma consulta que o teste de
// planos observa, por isso mora numa função.
func (r *SessaoRepository) consultaDaGrade(ctx context.Context, filtro usecase.FiltroSessoes) *gorm.DB {
	base := r.banco.conn(ctx).
		Table("sessoes AS s").
		Joins("JOIN filmes f ON f.id = s.filme_id").
		Joins("JOIN salas sa ON sa.id = s.sala_id").
		Joins("JOIN cinemas c ON c.id = sa.cinema_id").
		Where("s.status IN ?", statusVisiveis())

	if filtro.FilmeID != "" {
		base = base.Where("s.filme_id = ?", filtro.FilmeID)
	}
	if filtro.CinemaID != "" {
		base = base.Where("sa.cinema_id = ?", filtro.CinemaID)
	}
	if filtro.Data != nil {
		inicio := time.Date(filtro.Data.Ano, time.Month(filtro.Data.Mes), filtro.Data.Dia, 0, 0, 0, 0, time.UTC)
		base = base.Where("s.data_hora_inicio >= ? AND s.data_hora_inicio < ?", inicio, inicio.AddDate(0, 0, 1))
	}
	return base
}

func (r *SessaoRepository) Consultar(
	ctx context.Context,
	filtro usecase.FiltroSessoes,
	req shared.PageRequest,
) (shared.Page[catalogo.SessaoDetalhada], error) {
	pagina, err := consultarPaginado(r.consultaDaGrade(ctx, filtro), colunasSessaoDetalhada,
		"s.data_hora_inicio, s.id", req, sessaoDetalhadaRow.paraDominio)
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
	var bruto int64
	if err := r.banco.conn(ctx).Model(&sessaoRow{}).Where("status IN ?", statusVisiveis()).Count(&bruto).Error; err != nil {
		return
	}
	if int(bruto) > totalResolvido {
		slog.WarnContext(ctx, "sessões omitidas da grade por referência inválida",
			slog.Int("omitidas", int(bruto)-totalResolvido),
			slog.String("causa", "filme ou sala inexistente"))
	}
}

// A sessão e o fato que a anuncia vão na mesma transação: se o processo morrer
// entre as duas escritas, nenhuma delas aconteceu.
func (r *SessaoRepository) Criar(ctx context.Context, s catalogo.Sessao, fato usecase.FatoPendente) error {
	return r.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		linha := sessaoParaLinha(s)
		if err := tx.Create(&linha).Error; err != nil {
			return fmt.Errorf("inserindo sessão: %w", err)
		}
		return enfileirarFato(tx, fato)
	})
}

func (r *SessaoRepository) Atualizar(ctx context.Context, s catalogo.Sessao, fato usecase.FatoPendente) error {
	return r.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		res := tx.Model(&sessaoRow{}).Where("id = ?", s.ID).Updates(map[string]any{
			"filme_id": s.FilmeID, "sala_id": s.SalaID, "data_hora_inicio": s.DataHoraInicio,
			"idioma": string(s.Idioma), "preco_base": s.PrecoBase.String(),
			"status": string(s.Status), "atualizado_em": gormAgora,
		})
		if res.Error != nil {
			return fmt.Errorf("atualizando sessão: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return shared.NaoEncontrado("sessao", s.ID)
		}
		return enfileirarFato(tx, fato)
	})
}

func (r *SessaoRepository) Cancelar(ctx context.Context, sessaoID string, fato usecase.FatoPendente) error {
	return r.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		res := tx.Model(&sessaoRow{}).Where("id = ?", sessaoID).Updates(map[string]any{
			"status": string(catalogo.SessaoCancelada), "atualizado_em": gormAgora,
		})
		if res.Error != nil {
			return fmt.Errorf("cancelando sessão: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return shared.NaoEncontrado("sessao", sessaoID)
		}
		return enfileirarFato(tx, fato)
	})
}

// A duração de cada sessão concorrente é a do filme dela, então a janela sai do
// próprio SQL: nenhuma linha precisa subir para o Go só para ser descartada. É
// SQL cru porque a soma de um instante com `duracao * INTERVAL` não tem forma
// no construtor do GORM.
func (r *SessaoRepository) SalaOcupada(
	ctx context.Context,
	salaID string,
	inicio, fim time.Time,
	excetoID string,
) (bool, error) {
	const sql = `SELECT EXISTS(
	                 SELECT 1 FROM sessoes s JOIN filmes f ON f.id = s.filme_id
	                 WHERE s.sala_id = ? AND s.status IN ? AND s.id <> ?
	                   AND s.data_hora_inicio < ?
	                   AND s.data_hora_inicio + (f.duracao_minutos * INTERVAL '1 minute') > ?)`

	var ocupada bool
	err := r.banco.conn(ctx).Raw(sql, salaID, statusVisiveis(), excetoID, fim, inicio).Scan(&ocupada).Error
	if err != nil {
		return false, fmt.Errorf("verificando ocupação da sala: %w", err)
	}
	return ocupada, nil
}

func (r *SessaoRepository) BuscarPorID(ctx context.Context, sessaoID string) (catalogo.Sessao, error) {
	var linhas []sessaoRow
	if err := r.banco.conn(ctx).Where("id = ?", sessaoID).Limit(1).Find(&linhas).Error; err != nil {
		return catalogo.Sessao{}, fmt.Errorf("buscando sessão: %w", err)
	}
	if len(linhas) == 0 {
		return catalogo.Sessao{}, shared.NaoEncontrado("sessao", sessaoID)
	}
	return linhas[0].paraDominio()
}
