package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

// ListarIngressos devolve os ingressos da pessoa autenticada (FR-013, FR-014,
// FR-023, FR-024).
type ListarIngressos struct {
	Ingressos Ingressos
}

// Executar lista os ingressos de usuarioID. filtro vazio significa todos os
// estados; filtro fora do vocabulário é ErrStatusDesconhecido — recusa, e não
// ausência de filtro (FR-024).
//
// Não há parâmetro capaz de alcançar ingresso de terceiro: o recorte é o
// usuarioID, aplicado na consulta (FR-014).
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
		// Pessoa sem ingressos recebe listagem vazia, não erro nem nulo.
		return []ingresso.Ingresso{}, nil
	}
	return lista, nil
}
