package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

var (
	ErrNaoEncontrado      = errors.New("usecase: ingresso não encontrado")
	ErrStatusDesconhecido = errors.New("usecase: estado de filtro não reconhecido")
)

type Relogio interface{ Agora() time.Time }

type GeradorID interface{ Novo() string }

type Ingressos interface {
	CriarSeAusente(ctx context.Context, ing ingresso.Ingresso) (criado bool, atual ingresso.Ingresso, err error)

	Utilizar(ctx context.Context, id string, agora time.Time) (autorizado bool, err error)

	BuscarPorID(ctx context.Context, id string) (ingresso.Ingresso, error)

	ListarPorUsuario(ctx context.Context, usuarioID string, filtro ingresso.Status) ([]ingresso.Ingresso, error)
}

type Avisos interface {
	Registrar(ctx context.Context, r aviso.Registro) error
}

type Notificador interface {
	Canal() aviso.Canal
	Avisar(ctx context.Context, ing ingresso.Ingresso) error
}

type Assinador interface {
	Gerar(ingressoID string) string
	Verificar(codigo string) (string, error)
}
