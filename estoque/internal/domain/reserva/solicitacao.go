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
	// Texto decimal, informado por quem tem autoridade sobre o preço. Este
	// serviço confere apenas o formato: quanto vale a reserva é decisão do
	// catálogo, e recalcular aqui duplicaria a regra de preço.
	ValorTotal string
}

func NovaSolicitacao(sessaoID, usuarioID string, rotulos []string, valorTotal string, limite int) (Solicitacao, error) {
	sessaoID = strings.TrimSpace(sessaoID)
	usuarioID = strings.TrimSpace(usuarioID)
	valorTotal = strings.TrimSpace(valorTotal)

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

	if err := valorValido(valorTotal); err != nil {
		return Solicitacao{}, err
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

	return Solicitacao{
		SessaoID:   sessaoID,
		UsuarioID:  usuarioID,
		Rotulos:    canonicos,
		ValorTotal: valorTotal,
	}, nil
}

// O valor é conferido à mão, e não por `ParseFloat`, porque ele vira cobrança:
// o formato aceito é o mesmo que o catálogo produz — inteiro com até duas casas
// decimais, sem sinal, sem expoente. `ParseFloat` aceitaria "1e2" e "-0.01".
func valorValido(v string) error {
	invalido := fmt.Errorf("%w: valor_total deve ser um decimal como \"84.00\", com até duas casas", shared.ErrSolicitacaoInvalida)
	if v == "" {
		return fmt.Errorf("%w: valor_total não informado", shared.ErrSolicitacaoInvalida)
	}

	inteiro, decimais, temPonto := strings.Cut(v, ".")
	if inteiro == "" || (temPonto && (decimais == "" || len(decimais) > 2)) {
		return invalido
	}
	zerado := true
	for _, parte := range []string{inteiro, decimais} {
		for _, c := range parte {
			if c < '0' || c > '9' {
				return invalido
			}
			if c != '0' {
				zerado = false
			}
		}
	}
	if zerado {
		return fmt.Errorf("%w: valor_total deve ser maior que zero", shared.ErrSolicitacaoInvalida)
	}
	return nil
}
