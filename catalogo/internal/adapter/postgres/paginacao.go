package postgres

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func consultarPaginado[L, T any](
	base *gorm.DB,
	selecionar string,
	ordem string,
	req shared.PageRequest,
	converter func(L) (T, error),
) (shared.Page[T], error) {
	var vazia shared.Page[T]

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return vazia, fmt.Errorf("contando registros: %w", err)
	}

	pagina := base.Session(&gorm.Session{})
	if selecionar != "" {
		pagina = pagina.Select(selecionar)
	}
	var linhas []L
	if err := pagina.Order(ordem).Limit(req.Limit()).Offset(req.Offset()).Find(&linhas).Error; err != nil {
		return vazia, fmt.Errorf("consultando página: %w", err)
	}

	var itens []T
	for _, l := range linhas {
		item, err := converter(l)
		if err != nil {
			return vazia, err
		}
		itens = append(itens, item)
	}
	return shared.NovaPage(itens, int(total), req), nil
}
