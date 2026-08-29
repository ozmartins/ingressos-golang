package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// LimitesPaginacao são os valores vigentes, vindos da configuração.
type LimitesPaginacao struct {
	Padrao int
	Maximo int
}

// lerPaginacao traduz page e page_size em PageRequest.
//
// Ausentes recebem os padrões; valores não numéricos ou fora da faixa são
// recusados. O teto vigente vai no detalhe do erro, porque é informação de
// tempo de execução: o contrato publicado diz que existe um teto, não qual é.
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

// inteiroDaQuery devolve 0 quando o parâmetro está ausente — o valor que sinaliza
// "aplique o padrão" rio abaixo.
//
// Um zero *explícito* é caso diferente: está fora da faixa que o contrato aceita
// e é recusado aqui. Sem essa distinção, page=0 seria silenciosamente tratado
// como page=1, e quem integra nunca descobriria o próprio erro.
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
