package usecase

import (
	"context"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type CriarSala struct {
	Cinemas CinemaRepository
	Salas   SalaRepository
	GerarID func() string
}

func (uc CriarSala) Executar(ctx context.Context, dados catalogo.DadosSala) (catalogo.Sala, error) {
	sala, err := catalogo.NovaSala(uc.GerarID(), dados)
	if err != nil {
		return catalogo.Sala{}, err
	}
	if err := conferirCinemaELiberdadeDoNumero(ctx, uc.Cinemas, uc.Salas, sala, ""); err != nil {
		return catalogo.Sala{}, err
	}
	if err := uc.Salas.Criar(ctx, sala); err != nil {
		return catalogo.Sala{}, err
	}
	return sala, nil
}

// As duas verificações andam juntas nas duas escritas: o cinema informado
// precisa existir, e o número precisa estar livre entre as salas ativas dele.
func conferirCinemaELiberdadeDoNumero(
	ctx context.Context,
	cinemas CinemaRepository,
	salas SalaRepository,
	sala catalogo.Sala,
	excetoID string,
) error {
	existe, err := cinemas.Existe(ctx, sala.CinemaID)
	if err != nil {
		return err
	}
	if !existe {
		return shared.NaoEncontrado("cinema", sala.CinemaID)
	}
	if !sala.Ativo {
		// O índice único vale entre as salas ativas: uma sala desativada não
		// disputa o número com ninguém.
		return nil
	}
	emUso, err := salas.NumeroEmUso(ctx, sala.CinemaID, sala.Numero, excetoID)
	if err != nil {
		return err
	}
	if emUso {
		return errConflito("o cinema já tem uma sala ativa de número %d", sala.Numero)
	}
	return nil
}
