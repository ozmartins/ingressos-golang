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

// traduzir converte o desfecho da chamada em resultado ou erro de domínio.
//
// O mapeamento é o coração da integração:
//   - respondeu com sucesso  → resultado, validado quanto à integridade
//   - respondeu sem sucesso  → poltronas indisponíveis (conflito, não falha)
//   - falhou ou excedeu      → estoque indisponível
//   - recusa rápida ativa    → estoque indisponível, indistinguível do acima
func traduzir(resposta *estoquepb.RespostaBloqueio, desfecho Desfecho, err error) (reserva.ResultadoReserva, error) {
	switch desfecho {
	case DesfechoRecusado, DesfechoFalha:
		// O motivo real vai para o log e para a métrica; ao cliente, uma única
		// resposta padronizada, por exigência da especificação.
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
	// Sucesso sem identificador ou sem prazo é falha de contrato do parceiro:
	// nunca é repassado como sucesso.
	if err := resultado.ValidarIntegridade(); err != nil {
		return reserva.ResultadoReserva{}, err
	}
	return resultado, nil
}

// desfechoFinal escolhe o rótulo da métrica a partir do que de fato aconteceu.
// É a única superfície onde tempo excedido e recusa rápida se distinguem.
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
