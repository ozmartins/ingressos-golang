package usecase

import (
	"context"
	"log/slog"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

// Veredito é a resposta da portaria (contracts/erros.md §3).
type Veredito int

const (
	// Autorizada — o ingresso estava válido e foi utilizado neste ato.
	Autorizada Veredito = iota
	// Reuso — já utilizado antes.
	Reuso
	// NaoValido — cancelado.
	NaoValido
	// NaoEncontrado — código malformado, assinatura inválida ou inexistente.
	// Os três casos são o MESMO veredito, por exigência da FR-010.
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

// ResultadoValidacao é o que a portaria recebe.
type ResultadoValidacao struct {
	Veredito   Veredito
	Ingresso   ingresso.Ingresso
	TemIgresso bool
}

// ValidarIngresso é a baixa do ingresso na entrada da sala (FR-006 a FR-011).
type ValidarIngresso struct {
	Ingressos Ingressos
	Assinador Assinador
	Relogio   Relogio
	Log       *slog.Logger
}

// Executar confere a assinatura, tenta a baixa e só então classifica a recusa.
//
// A ordem é deliberada: um código forjado é recusado sem tocar no banco
// (FR-010), e a autorização é uma escrita condicionada, de modo que leituras
// simultâneas do mesmo código produzam no máximo uma autorização (FR-011).
func (u ValidarIngresso) Executar(ctx context.Context, codigo string) (ResultadoValidacao, error) {
	id, err := u.Assinador.Verificar(codigo)
	if err != nil {
		// Nunca registra o código apresentado: log é copiado e lido por muita
		// gente, e um código em log é um ingresso em log (FR-021).
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

	// Só aqui se pergunta o motivo — fora do caminho de sucesso (research.md D4).
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
