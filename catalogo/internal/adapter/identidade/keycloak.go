// Package identidade valida credenciais emitidas pelo provedor externo.
package identidade

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

// ErrCredencialInvalida cobre todos os motivos de recusa. A distinção entre
// "expirado", "assinatura inválida" e "emissor errado" é registrada no log, mas
// não devolvida ao cliente: dizer qual verificação falhou ajuda mais quem ataca
// do que quem integra.
var ErrCredencialInvalida = errors.New("credencial inválida")

// Verificador valida tokens localmente, sem consultar o emissor a cada
// requisição (FR-020). As chaves públicas são buscadas uma vez e reutilizadas; o
// conjunto é reconsultado apenas quando aparece uma chave desconhecida, o que
// também cobre rotação sem reinício.
type Verificador struct {
	verificador *oidc.IDTokenVerifier
}

// NovoVerificador descobre a configuração do emissor e monta o verificador.
func NovoVerificador(ctx context.Context, issuerURL, audiencia string) (*Verificador, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("descobrindo emissor de credenciais em %s: %w", issuerURL, err)
	}
	return &Verificador{
		verificador: provider.Verifier(&oidc.Config{
			ClientID:             audiencia,
			SupportedSigningAlgs: []string{oidc.RS256},
		}),
	}, nil
}

// NovoVerificadorComKeySet monta o verificador a partir de um conjunto de chaves
// já disponível. Existe para os testes exercitarem a verificação sem depender de
// um Keycloak em pé.
func NovoVerificadorComKeySet(ks oidc.KeySet, issuerURL, audiencia string) *Verificador {
	return &Verificador{
		verificador: oidc.NewVerifier(issuerURL, ks, &oidc.Config{
			ClientID:             audiencia,
			SupportedSigningAlgs: []string{oidc.RS256},
		}),
	}
}

// Identidade é o que o serviço precisa saber sobre quem faz a requisição.
type Identidade struct {
	UsuarioID string
}

// Verificar valida o token e extrai a identidade.
//
// Token válido em assinatura mas sem `sub` é recusado: sem saber a quem atribuir
// a reserva, prosseguir criaria um bloqueio sem dono.
func (v *Verificador) Verificar(ctx context.Context, tokenBruto string) (Identidade, error) {
	token, err := v.verificador.Verify(ctx, tokenBruto)
	if err != nil {
		return Identidade{}, fmt.Errorf("%w: %v", ErrCredencialInvalida, err)
	}
	if token.Subject == "" {
		return Identidade{}, fmt.Errorf("%w: credencial sem identificação da pessoa usuária", ErrCredencialInvalida)
	}
	return Identidade{UsuarioID: token.Subject}, nil
}
