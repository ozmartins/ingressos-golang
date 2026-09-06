package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

type AtualizarSala struct {
	Cinemas CinemaRepository
	Salas   SalaRepository
}

// A atualização substitui a sala inteira: o corpo descreve o estado final, e
// campo omitido volta a ser ausente. O cinema é a única exceção — ele faz parte
// do cadastro da sala, não do estado que o PUT redesenha: omitido, permanece o
// atual; informado, precisa repetir o atual.
func (uc AtualizarSala) Executar(ctx context.Context, salaID string, dados catalogo.DadosSala) (catalogo.Sala, error) {
	atual, err := uc.Salas.BuscarPorID(ctx, salaID)
	if err != nil {
		return catalogo.Sala{}, err
	}
	if dados.CinemaID == "" {
		dados.CinemaID = atual.CinemaID
	}
	if dados.CinemaID != atual.CinemaID {
		return catalogo.Sala{}, errConflito("a sala pertence ao cinema %s e não pode mudar de cinema", atual.CinemaID)
	}

	sala, err := catalogo.NovaSala(salaID, dados)
	if err != nil {
		return catalogo.Sala{}, err
	}
	if err := conferirCinemaELiberdadeDoNumero(ctx, uc.Cinemas, uc.Salas, sala, salaID); err != nil {
		return catalogo.Sala{}, err
	}
	if err := uc.Salas.Atualizar(ctx, sala); err != nil {
		return catalogo.Sala{}, err
	}
	return sala, nil
}
