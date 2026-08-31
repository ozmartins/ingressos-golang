// Package usecase orquestra o domínio. Depende apenas de interfaces declaradas
// aqui — nunca de adaptador, driver ou biblioteca de transporte.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

var (
	// ErrNaoEncontrado distingue ausência de erro de infraestrutura.
	ErrNaoEncontrado = errors.New("usecase: ingresso não encontrado")
	// ErrStatusDesconhecido é o filtro de listagem fora do vocabulário (FR-024).
	ErrStatusDesconhecido = errors.New("usecase: estado de filtro não reconhecido")
)

// Relogio existe para que os instantes sejam testáveis sem espera real (D9).
type Relogio interface{ Agora() time.Time }

// GeradorID produz identidades opacas.
type GeradorID interface{ Novo() string }

// Ingressos é a porta de persistência do ingresso (data-model.md §5).
type Ingressos interface {
	// CriarSeAusente é a porta de entrada da idempotência (FR-004, D2).
	// criado=false significa que outra entrega chegou primeiro: esta NÃO emite
	// e NÃO avisa.
	CriarSeAusente(ctx context.Context, ing ingresso.Ingresso) (criado bool, atual ingresso.Ingresso, err error)

	// Utilizar é a escrita condicionada da D4: aplica a baixa somente se o
	// ingresso estiver VALIDO. autorizado=false NÃO diz por quê — o motivo é
	// uma segunda pergunta, feita só quando a primeira já respondeu não.
	Utilizar(ctx context.Context, id string, agora time.Time) (autorizado bool, err error)

	// BuscarPorID devolve ErrNaoEncontrado quando não há ingresso.
	BuscarPorID(ctx context.Context, id string) (ingresso.Ingresso, error)

	// ListarPorUsuario aplica ordenação e filtro (FR-023, FR-024). Filtro vazio
	// significa todos os estados.
	ListarPorUsuario(ctx context.Context, usuarioID string, filtro ingresso.Status) ([]ingresso.Ingresso, error)
}

// Avisos é a porta de persistência do registro de notificação.
type Avisos interface {
	Registrar(ctx context.Context, r aviso.Registro) error
}

// Notificador é a porta de saída do aviso (D6). O único adaptador desta entrega
// é o simulado; a falha dele é capturada, nunca propagada (FR-025).
type Notificador interface {
	// Canal identifica por onde o aviso saiu, para o registro.
	Canal() aviso.Canal
	Avisar(ctx context.Context, ing ingresso.Ingresso) error
}

// Assinador gera e confere o código de acesso (D3).
type Assinador interface {
	Gerar(ingressoID string) string
	Verificar(codigo string) (string, error)
}
