package postgres

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

// Maior lote de poltronas por INSERT: 1000 linhas x 7 colunas fica muito abaixo
// do limite de 65535 parâmetros do protocolo.
const loteProvisionamento = 1000

type Poltronas struct{ banco *Banco }

func NovoRepositorioPoltronas(b *Banco) *Poltronas { return &Poltronas{banco: b} }

func (p *Poltronas) MapaDaSessao(ctx context.Context, sessaoID string) ([]poltrona.Poltrona, error) {
	linhas, err := p.banco.db.WithContext(ctx).Model(&poltronaRow{}).
		Select("id", "sessao_id", "fileira", "numero", "rotulo", "tipo", "status").
		Where("sessao_id = ?", sessaoID).
		Order("fileira, numero").
		Rows()
	if err != nil {
		return nil, indisponivel(err)
	}
	defer linhas.Close()

	var mapa []poltrona.Poltrona
	for linhas.Next() {
		var item poltrona.Poltrona
		var tipo, status string
		if err := linhas.Scan(&item.ID, &item.SessaoID, &item.Fileira, &item.Numero,
			&item.Rotulo, &tipo, &status); err != nil {
			return nil, indisponivel(err)
		}
		item.Tipo, item.Status = poltrona.Tipo(tipo), poltrona.Status(status)
		mapa = append(mapa, item)
	}
	if err := linhas.Err(); err != nil {
		return nil, indisponivel(err)
	}
	return mapa, nil
}

func (p *Poltronas) ProvisionarMatriz(ctx context.Context, fila, messageID, sessaoID string, matriz []poltrona.Poltrona) (usecase.ResultadoTransicao, error) {
	resultado := usecase.TransicaoAplicada

	err := p.banco.EmTransacao(ctx, func(tx *gorm.DB) error {
		if messageID != "" {
			novo, err := registrarProcessada(tx, fila, messageID)
			if err != nil {
				return err
			}
			if !novo {
				resultado = usecase.TransicaoIgnoradaDuplicata
				return nil
			}
		}

		var criadas int64
		if len(matriz) > 0 {
			linhas := make([]poltronaRow, 0, len(matriz))
			for _, item := range matriz {
				linhas = append(linhas, poltronaRow{
					ID: item.ID, SessaoID: item.SessaoID, Fileira: item.Fileira, Numero: item.Numero,
					Rotulo: item.Rotulo, Tipo: string(item.Tipo), Status: string(poltrona.Livre),
				})
			}
			res := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "sessao_id"}, {Name: "fileira"}, {Name: "numero"}},
				DoNothing: true,
			}).CreateInBatches(&linhas, loteProvisionamento)
			if res.Error != nil {
				return indisponivel(res.Error)
			}
			criadas = res.RowsAffected
		}

		if criadas == 0 {
			resultado = usecase.TransicaoIgnoradaDuplicata
		}
		return nil
	})

	return resultado, err
}
