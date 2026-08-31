// Package health separa "o processo está no ar" de "o serviço é capaz de emitir
// e validar ingressos". A distinção importa: um serviço vivo que não alcança o
// banco não deve receber tráfego, mas também não deve ser reiniciado.
package health

import "context"

// Sonda verifica uma dependência.
type Sonda func(context.Context) error

// Prontidao agrega as sondas das dependências de que o serviço precisa.
type Prontidao struct {
	sondas map[string]Sonda
}

func NovaProntidao() *Prontidao { return &Prontidao{sondas: map[string]Sonda{}} }

func (p *Prontidao) Registrar(nome string, s Sonda) { p.sondas[nome] = s }

// Verificar devolve o nome da primeira dependência que falhou, e o erro.
func (p *Prontidao) Verificar(ctx context.Context) (string, error) {
	for nome, s := range p.sondas {
		if err := s(ctx); err != nil {
			return nome, err
		}
	}
	return "", nil
}
