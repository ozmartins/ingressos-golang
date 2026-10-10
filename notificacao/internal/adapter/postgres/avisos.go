package postgres

import (
	"context"

	"gorm.io/gorm"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
)

type Avisos struct{ DB *gorm.DB }

func (r Avisos) Registrar(ctx context.Context, reg aviso.Registro) error {
	linha := avisoParaLinha(reg)
	if err := r.DB.WithContext(ctx).Create(&linha).Error; err != nil {
		return falhaInfra("registrar aviso", err)
	}
	return nil
}
