package shared

import (
	"errors"
	"testing"
)

func TestNovoPageRequestAplicaPadroes(t *testing.T) {
	req, err := NovoPageRequest(0, 0, 20, 100)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if req.Numero != 1 || req.Tamanho != 20 {
		t.Fatalf("esperava página 1 tamanho 20, obteve %d/%d", req.Numero, req.Tamanho)
	}
}

func TestNovoPageRequestRecusaTamanhoAcimaDoTeto(t *testing.T) {
	// SC-008: o teto é recusado, nunca reduzido em silêncio.
	_, err := NovoPageRequest(1, 500, 20, 100)
	if !errors.Is(err, ErrValidacao) {
		t.Fatalf("esperava ErrValidacao, obteve %v", err)
	}
}

func TestNovoPageRequestRecusaValoresNegativos(t *testing.T) {
	for _, tc := range []struct{ numero, tamanho int }{{-1, 20}, {1, -5}} {
		if _, err := NovoPageRequest(tc.numero, tc.tamanho, 20, 100); !errors.Is(err, ErrValidacao) {
			t.Fatalf("numero=%d tamanho=%d: esperava ErrValidacao, obteve %v", tc.numero, tc.tamanho, err)
		}
	}
}

func TestNovaPageCalculaTemProxima(t *testing.T) {
	req, _ := NovoPageRequest(1, 20, 20, 100)
	if p := NovaPage([]string{"a"}, 137, req); !p.TemProxima {
		t.Fatal("página 1 de 137 registros deveria ter próxima")
	}
	req7, _ := NovoPageRequest(7, 20, 20, 100)
	if p := NovaPage([]string{"a"}, 137, req7); p.TemProxima {
		t.Fatal("página 7 de 137 registros não deveria ter próxima")
	}
}

func TestNovaPageAlemDoFimEhVaziaNaoErro(t *testing.T) {
	// FR-005: posição além do fim devolve página vazia com o total correto.
	req, _ := NovoPageRequest(9999, 20, 20, 100)
	p := NovaPage[string](nil, 137, req)
	if len(p.Itens) != 0 {
		t.Fatalf("esperava página vazia, obteve %d itens", len(p.Itens))
	}
	if p.Itens == nil {
		t.Fatal("itens deveria ser fatia vazia, não nil (serializa como [] e não null)")
	}
	if p.Total != 137 || p.TemProxima {
		t.Fatalf("total=%d temProxima=%v", p.Total, p.TemProxima)
	}
}

func TestOffsetELimit(t *testing.T) {
	req, _ := NovoPageRequest(3, 25, 20, 100)
	if req.Offset() != 50 || req.Limit() != 25 {
		t.Fatalf("offset=%d limit=%d", req.Offset(), req.Limit())
	}
}
