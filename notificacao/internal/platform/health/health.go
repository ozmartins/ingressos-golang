package health

import "context"

type Sonda func(context.Context) error

type Prontidao struct {
	sondas map[string]Sonda
}

func NovaProntidao() *Prontidao { return &Prontidao{sondas: map[string]Sonda{}} }

func (p *Prontidao) Registrar(nome string, s Sonda) { p.sondas[nome] = s }

func (p *Prontidao) Verificar(ctx context.Context) (string, error) {
	for nome, s := range p.sondas {
		if err := s(ctx); err != nil {
			return nome, err
		}
	}
	return "", nil
}
