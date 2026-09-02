package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

type ListarIngressos struct {
	Ingressos Ingressos
}

func (u ListarIngressos) Executar(ctx context.Context, usuarioID string, filtro string) ([]ingresso.Ingresso, error) {
	s := ingresso.Status(filtro)
	if filtro != "" && !ingresso.Reconhecido(s) {
		return nil, ErrStatusDesconhecido
	}
	if filtro == "" {
		s = ""
	}
	lista, err := u.Ingressos.ListarPorUsuario(ctx, usuarioID, s)
	if err != nil {
		return nil, err
	}
	if lista == nil {
		return []ingresso.Ingresso{}, nil
	}
	return lista, nil
}
