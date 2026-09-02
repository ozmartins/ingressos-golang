package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

type Poltronas struct{ banco *Banco }

func NovoRepositorioPoltronas(b *Banco) *Poltronas { return &Poltronas{banco: b} }

func (p *Poltronas) MapaDaSessao(ctx context.Context, sessaoID string) ([]poltrona.Poltrona, error) {
	linhas, err := p.banco.pool.Query(ctx, `
		SELECT id, sessao_id, fileira, numero, rotulo, tipo, status
		  FROM poltronas
		 WHERE sessao_id = $1
		 ORDER BY fileira, numero`, sessaoID)
	if err != nil {
		return nil, indisponivel(err)
	}
	defer linhas.Close()

	var mapa []poltrona.Poltrona
	for linhas.Next() {
		var item poltrona.Poltrona
		if err := linhas.Scan(&item.ID, &item.SessaoID, &item.Fileira, &item.Numero,
			&item.Rotulo, &item.Tipo, &item.Status); err != nil {
			return nil, indisponivel(err)
		}
		mapa = append(mapa, item)
	}
	if err := linhas.Err(); err != nil {
		return nil, indisponivel(err)
	}
	return mapa, nil
}

func (p *Poltronas) ProvisionarMatriz(ctx context.Context, fila, messageID, sessaoID string, matriz []poltrona.Poltrona) (usecase.ResultadoTransicao, error) {
	resultado := usecase.TransicaoAplicada

	err := p.banco.EmTransacao(ctx, func(tx pgx.Tx) error {
		if messageID != "" {
			novo, err := registrarProcessada(ctx, tx, fila, messageID)
			if err != nil {
				return err
			}
			if !novo {
				resultado = usecase.TransicaoIgnoradaDuplicata
				return nil
			}
		}

		var criadas int64
		for _, item := range matriz {
			tag, err := tx.Exec(ctx, `
				INSERT INTO poltronas (id, sessao_id, fileira, numero, rotulo, tipo, status)
				VALUES ($1, $2, $3, $4, $5, $6, 'LIVRE')
				ON CONFLICT (sessao_id, fileira, numero) DO NOTHING`,
				item.ID, item.SessaoID, item.Fileira, item.Numero, item.Rotulo, string(item.Tipo))
			if err != nil {
				return indisponivel(err)
			}
			criadas += tag.RowsAffected()
		}

		if criadas == 0 {
			resultado = usecase.TransicaoIgnoradaDuplicata
		}
		return nil
	})

	return resultado, err
}
