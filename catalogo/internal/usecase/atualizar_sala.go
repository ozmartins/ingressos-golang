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
// campo omitido volta a ser ausente. O cinema e a planta são as exceções — os
// dois são do cadastro da sala, não do estado que o PUT redesenha: omitidos,
// permanecem os atuais; informados, precisam repetir os atuais.
//
// A planta é imutável porque as sessões já anunciadas dela carregam a matriz de
// poltronas que valia quando foram criadas. Redesenhar a sala deixaria essas
// matrizes descrevendo assentos que não existem mais — e uma poltrona vendida
// numa fileira removida não tem para onde ir. Quem precisa de outra planta
// desativa a sala e cadastra outra: o `DELETE` é lógico e devolve o número, que
// fica livre para a sala que a substituir.
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
	if dados.Fileiras == nil {
		dados.Fileiras = atual.Layout.Dados()
	}

	sala, err := catalogo.NovaSala(salaID, dados)
	if err != nil {
		return catalogo.Sala{}, err
	}
	// Comparada depois de construída, e não campo a campo do corpo: é a planta
	// normalizada — ordenada, com os tipos preenchidos — que precisa bater, para
	// que reenviar a mesma planta em outra ordem não seja recusado.
	if !sala.Layout.Igual(atual.Layout) {
		return catalogo.Sala{}, errConflito(
			"a planta da sala não pode ser redesenhada; desative esta sala e cadastre outra")
	}
	if err := conferirCinemaELiberdadeDoNumero(ctx, uc.Cinemas, uc.Salas, sala, salaID); err != nil {
		return catalogo.Sala{}, err
	}
	if err := uc.Salas.Atualizar(ctx, sala); err != nil {
		return catalogo.Sala{}, err
	}
	return sala, nil
}
