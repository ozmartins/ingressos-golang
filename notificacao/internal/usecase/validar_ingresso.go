package usecase

import (
	"context"
	"errors"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

type Veredito int

const (
	Autorizada Veredito = iota
	Reuso
	NaoValido
	NaoEncontrado
)

func (v Veredito) String() string {
	switch v {
	case Autorizada:
		return "autorizada"
	case Reuso:
		return "reuso"
	case NaoValido:
		return "nao_valido"
	default:
		return "nao_encontrado"
	}
}

type ResultadoValidacao struct {
	Veredito   Veredito
	Ingresso   ingresso.Ingresso
	TemIgresso bool
}

type ValidarIngresso struct {
	Ingressos Ingressos
	Assinador Assinador
	Relogio   Relogio
}

func (u ValidarIngresso) Executar(ctx context.Context, codigo string) (ResultadoValidacao, error) {
	id, err := u.Assinador.Verificar(codigo)
	if err != nil {
		return ResultadoValidacao{Veredito: NaoEncontrado}, nil
	}

	autorizado, err := u.Ingressos.Utilizar(ctx, id, u.Relogio.Agora())
	if err != nil {
		return ResultadoValidacao{}, err
	}
	if autorizado {
		i, err := u.Ingressos.BuscarPorID(ctx, id)
		if err != nil {
			return ResultadoValidacao{}, err
		}

		return ResultadoValidacao{Veredito: Autorizada, Ingresso: i, TemIgresso: true}, nil
	}

	i, err := u.Ingressos.BuscarPorID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			return ResultadoValidacao{Veredito: NaoEncontrado}, nil
		}
		return ResultadoValidacao{}, err
	}

	v := NaoValido
	if i.Status == ingresso.Utilizado {
		v = Reuso
	}

	return ResultadoValidacao{Veredito: v, Ingresso: i, TemIgresso: true}, nil
}
