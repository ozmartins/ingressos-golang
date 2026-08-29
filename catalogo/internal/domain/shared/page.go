// Package shared reúne os tipos de domínio compartilhados por todos os agregados.
package shared

import "fmt"

// Limites de paginação. Os valores efetivos vêm da configuração; estes são os
// padrões de referência usados quando nada é informado.
const (
	TamanhoPaginaPadrao = 20
	TamanhoPaginaMaximo = 100
)

// PageRequest é a posição e o tamanho pedidos numa consulta de coleção.
type PageRequest struct {
	Numero  int
	Tamanho int
}

// NovoPageRequest valida e normaliza os parâmetros de paginação.
//
// Numero e Tamanho ausentes (zero) recebem os padrões. Tamanho acima do teto é
// recusado, nunca reduzido em silêncio: SC-008 exige que o teto seja observável
// pelo cliente, e reduzir sem avisar esconde que ele pediu algo inválido.
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

// Offset é o deslocamento correspondente à posição pedida.
func (p PageRequest) Offset() int { return (p.Numero - 1) * p.Tamanho }

// Limit é o número máximo de itens da página.
func (p PageRequest) Limit() int { return p.Tamanho }

// Page é uma fatia de uma coleção, com o total que atende aos filtros.
type Page[T any] struct {
	Itens      []T
	Total      int
	Numero     int
	Tamanho    int
	TemProxima bool
}

// NovaPage monta a página a partir dos itens já recortados e do total filtrado.
// Itens nil vira fatia vazia: uma posição além do fim devolve página vazia, não
// erro (FR-005).
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
