package shared

import "fmt"

const (
	TamanhoPaginaPadrao = 20
	TamanhoPaginaMaximo = 100
)

type PageRequest struct {
	Numero  int
	Tamanho int
}

func NovoPageRequest(numero, tamanho, tamanhoPadrao, tamanhoMaximo int) (PageRequest, error) {
	if tamanhoPadrao <= 0 {
		tamanhoPadrao = TamanhoPaginaPadrao
	}
	if tamanhoMaximo <= 0 {
		tamanhoMaximo = TamanhoPaginaMaximo
	}
	if numero == 0 {
		numero = 1
	}
	if tamanho == 0 {
		tamanho = tamanhoPadrao
	}
	if numero < 1 {
		return PageRequest{}, fmt.Errorf("%w: page deve ser maior ou igual a 1", ErrValidacao)
	}
	if tamanho < 1 {
		return PageRequest{}, fmt.Errorf("%w: page_size deve ser maior ou igual a 1", ErrValidacao)
	}
	if tamanho > tamanhoMaximo {
		return PageRequest{}, fmt.Errorf("%w: page_size acima do máximo aceito (%d)", ErrValidacao, tamanhoMaximo)
	}
	return PageRequest{Numero: numero, Tamanho: tamanho}, nil
}

func (p PageRequest) Offset() int { return (p.Numero - 1) * p.Tamanho }

func (p PageRequest) Limit() int { return p.Tamanho }

type Page[T any] struct {
	Itens      []T
	Total      int
	Numero     int
	Tamanho    int
	TemProxima bool
}

func NovaPage[T any](itens []T, total int, req PageRequest) Page[T] {
	if itens == nil {
		itens = []T{}
	}
	return Page[T]{
		Itens:      itens,
		Total:      total,
		Numero:     req.Numero,
		Tamanho:    req.Tamanho,
		TemProxima: req.Numero*req.Tamanho < total,
	}
}
