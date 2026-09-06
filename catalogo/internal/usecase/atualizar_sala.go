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
// campo omitido volta a ser ausente.
func (uc AtualizarSala) Executar(ctx context.Context, cinemaID, salaID string, dados catalogo.DadosSala) (catalogo.Sala, error) {
	atual, err := uc.Salas.BuscarPorID(ctx, salaID)
	if err != nil {
		return catalogo.Sala{}, err
	}
	if err := conferirPosse(atual, cinemaID); err != nil {
		return catalogo.Sala{}, err
	}

	dados.CinemaID = cinemaID
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
