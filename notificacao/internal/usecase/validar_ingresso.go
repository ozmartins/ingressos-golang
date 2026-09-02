package usecase

import (
	"context"
	"log/slog"

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
	Log       *slog.Logger
}

func (u ValidarIngresso) Executar(ctx context.Context, codigo string) (ResultadoValidacao, error) {
	id, err := u.Assinador.Verificar(codigo)
	if err != nil {
		u.log().Warn("código recusado na verificação de autenticidade",
			"desfecho", NaoEncontrado.String())
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
		u.log().Info("entrada autorizada", "ingresso_id", id, "desfecho", Autorizada.String())
		return ResultadoValidacao{Veredito: Autorizada, Ingresso: i, TemIgresso: true}, nil
	}

	i, err := u.Ingressos.BuscarPorID(ctx, id)
	if err != nil {
		if err == ErrNaoEncontrado {
			u.log().Warn("código autêntico sem ingresso correspondente",
				"ingresso_id", id, "desfecho", NaoEncontrado.String())
			return ResultadoValidacao{Veredito: NaoEncontrado}, nil
		}
		return ResultadoValidacao{}, err
	}

	v := NaoValido
	if i.Status == ingresso.Utilizado {
		v = Reuso
	}
	u.log().Warn("entrada negada", "ingresso_id", id, "status", string(i.Status), "desfecho", v.String())
	return ResultadoValidacao{Veredito: v, Ingresso: i, TemIgresso: true}, nil
}

func (u ValidarIngresso) log() *slog.Logger {
	if u.Log != nil {
		return u.Log
	}
	return slog.Default()
}
