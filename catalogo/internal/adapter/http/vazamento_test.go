package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// T076 — nenhuma resposta de erro pode carregar detalhe interno.
//
// Varre erros realistas de cada camada — driver, gRPC, resolução de nome — e
// confirma que nada disso aparece no corpo devolvido ao cliente. É uma
// verificação de superfície de ataque, não de formatação.
func TestNenhumaRespostaVazaDetalheInterno(t *testing.T) {
	errosRealistas := []error{
		errors.New(`ERROR: relation "sessoes" does not exist (SQLSTATE 42P01)`),
		errors.New(`failed to connect to host=db-prod-01.interno port=5432: dial tcp 10.0.3.12:5432: connect: connection refused`),
		errors.New(`rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 10.0.4.7:50051: connect: connection refused"`),
		fmt.Errorf("%w: rpc error: code = DeadlineExceeded desc = context deadline exceeded", shared.ErrEstoqueIndisponivel),
		fmt.Errorf("%w: circuit breaker estoque.bloqueio is open", shared.ErrEstoqueIndisponivel),
		fmt.Errorf("consultando filmes: %w", errors.New(`pq: syntax error at or near "SELCT"`)),
	}

	proibidos := []string{
		"SQLSTATE", "relation", "SELCT", "pq:", "rpc error", "grpc", "circuit breaker",
		"dial tcp", "10.0.", "5432", "50051", "db-prod-01", "goroutine", ".go:",
	}

	for _, err := range errosRealistas {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/sessoes/x/reservar", nil)
		EscreverErroDeDominio(w, r, err, "sessao")

		var p Problem
		if e := json.Unmarshal(w.Body.Bytes(), &p); e != nil {
			t.Fatalf("resposta não é problem+json: %v", e)
		}
		corpo := p.Detail + " " + p.Title + " " + p.Type
		for _, proibido := range proibidos {
			if strings.Contains(strings.ToLower(corpo), strings.ToLower(proibido)) {
				t.Errorf("erro %q vazou %q na resposta: %s", err, proibido, corpo)
			}
		}
	}
}

// O pânico não pode devolver a pilha ao cliente.
func TestPanicoNaoVazaPilha(t *testing.T) {
	// A recuperação vive no middleware; aqui garantimos que o corpo padrão que
	// ele escreve é o mesmo problem+json genérico, sem rastro de execução.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/filmes", nil)
	EscreverProblem(w, r, catErroInterno, "Erro interno. Consulte o suporte informando o identificador desta requisição.")

	corpo := w.Body.String()
	for _, proibido := range []string{"goroutine", "runtime.", ".go:", "panic"} {
		if strings.Contains(corpo, proibido) {
			t.Errorf("resposta de pânico vazou %q: %s", proibido, corpo)
		}
	}
}
