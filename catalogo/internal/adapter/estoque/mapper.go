package estoque

import (
	"errors"
	"fmt"
	"time"

	estoquepb "github.com/oseias/ingressos-golang/catalogo/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/platform/observability"
)

func traduzir(resposta *estoquepb.RespostaBloqueio, desfecho Desfecho, err error) (reserva.ResultadoReserva, error) {
	switch desfecho {
	case DesfechoRecusado, DesfechoFalha:
		return reserva.ResultadoReserva{}, fmt.Errorf("%w: %v", shared.ErrEstoqueIndisponivel, err)
	}

	if resposta == nil {
		return reserva.ResultadoReserva{}, fmt.Errorf("%w: resposta vazia", shared.ErrRespostaInvalidaDoParceiro)
	}

	if !resposta.GetSucesso() {
		return reserva.ResultadoReserva{}, shared.ErrPoltronasIndisponiveis
	}

	resultado := reserva.ResultadoReserva{ReservaID: resposta.GetReservaId()}
	if e := resposta.GetExpiraEm(); e > 0 {
		resultado.ExpiraEm = time.Unix(e, 0).UTC()
	}
	if err := resultado.ValidarIntegridade(); err != nil {
		return reserva.ResultadoReserva{}, err
	}
	return resultado, nil
}

func desfechoFinal(d Desfecho, erroTransporte, erroDominio error) string {
	switch {
	case d == DesfechoRecusado:
		return observability.DesfechoRecusaRapida
	case d == DesfechoFalha && ehTempoExcedido(erroTransporte):
		return observability.DesfechoTimeout
	case d == DesfechoFalha:
		return observability.DesfechoIndisponivel
	case errors.Is(erroDominio, shared.ErrPoltronasIndisponiveis):
		return observability.DesfechoPoltronasIndisponiveis
	case erroDominio != nil:
		return observability.DesfechoIndisponivel
	default:
		return observability.DesfechoSucesso
	}
}
