package reserva

import (
	"fmt"
	"strings"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type Solicitacao struct {
	SessaoID  string
	UsuarioID string
	Rotulos   []string
}

func NovaSolicitacao(sessaoID, usuarioID string, rotulos []string, limite int) (Solicitacao, error) {
	sessaoID = strings.TrimSpace(sessaoID)
	usuarioID = strings.TrimSpace(usuarioID)

	if sessaoID == "" {
		return Solicitacao{}, fmt.Errorf("%w: sessão não informada", shared.ErrSolicitacaoInvalida)
	}
	if usuarioID == "" {
		return Solicitacao{}, fmt.Errorf("%w: identidade da pessoa usuária não informada", shared.ErrSolicitacaoInvalida)
	}
	if len(rotulos) == 0 {
		return Solicitacao{}, fmt.Errorf("%w: nenhuma poltrona solicitada", shared.ErrSolicitacaoInvalida)
	}
	if limite > 0 && len(rotulos) > limite {
		return Solicitacao{}, fmt.Errorf("%w: %d solicitadas, máximo %d", shared.ErrLimiteExcedido, len(rotulos), limite)
	}

	vistos := make(map[string]struct{}, len(rotulos))
	canonicos := make([]string, 0, len(rotulos))
	for _, bruto := range rotulos {
		fileira, numero, err := poltrona.LerRotulo(bruto)
		if err != nil {
			return Solicitacao{}, err
		}
		rotulo := poltrona.MontarRotulo(fileira, numero)
		if _, repetido := vistos[rotulo]; repetido {
			return Solicitacao{}, fmt.Errorf("%w: poltrona %s repetida na solicitação", shared.ErrSolicitacaoInvalida, rotulo)
		}
		vistos[rotulo] = struct{}{}
		canonicos = append(canonicos, rotulo)
	}

	return Solicitacao{SessaoID: sessaoID, UsuarioID: usuarioID, Rotulos: canonicos}, nil
}
