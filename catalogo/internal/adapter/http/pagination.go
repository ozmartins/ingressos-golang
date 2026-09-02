package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type LimitesPaginacao struct {
	Padrao int
	Maximo int
}

func lerPaginacao(r *http.Request, lim LimitesPaginacao) (shared.PageRequest, error) {
	numero, err := inteiroDaQuery(r, "page")
	if err != nil {
		return shared.PageRequest{}, err
	}
	tamanho, err := inteiroDaQuery(r, "page_size")
	if err != nil {
		return shared.PageRequest{}, err
	}
	return shared.NovoPageRequest(numero, tamanho, lim.Padrao, lim.Maximo)
}

func inteiroDaQuery(r *http.Request, chave string) (int, error) {
	bruto := r.URL.Query().Get(chave)
	if bruto == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(bruto)
	if err != nil {
		return 0, fmt.Errorf("%w: %s deve ser um número inteiro", shared.ErrValidacao, chave)
	}
	if n < 1 {
		return 0, fmt.Errorf("%w: %s deve ser maior ou igual a 1", shared.ErrValidacao, chave)
	}
	return n, nil
}
