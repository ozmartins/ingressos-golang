// Package health expõe liveness e readiness na porta de administração, fora do
// canal gRPC de negócio.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Verificacao é uma dependência a checar.
type Verificacao struct {
	Nome string
	// Essencial: sua indisponibilidade torna a instância inapta a receber
	// tráfego. O PostgreSQL é essencial; o Redis não (D11).
	Essencial bool
	Checar    func(context.Context) error
}

// Servico agrupa as verificações.
type Servico struct {
	verificacoes []Verificacao
}

// Novo monta o serviço de saúde.
func Novo(verificacoes ...Verificacao) *Servico { return &Servico{verificacoes: verificacoes} }

type resposta struct {
	Status       string            `json:"status"`
	Dependencias map[string]string `json:"dependencias"`
}

// Handler devolve o mux de administração com /health/live e /health/ready.
func (s *Servico) Handler() http.Handler {
	mux := http.NewServeMux()

	// Vivo enquanto o processo responde.
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"vivo"}`))
	})

	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancelar()

		corpo := resposta{Status: "pronto", Dependencias: map[string]string{}}
		codigo := http.StatusOK

		for _, v := range s.verificacoes {
			if err := v.Checar(ctx); err != nil {
				if v.Essencial {
					// Sem esta dependência não há bloqueio correto: a instância
					// não deve receber tráfego (FR-006).
					corpo.Dependencias[v.Nome] = "indisponivel"
					corpo.Status = "indisponivel"
					codigo = http.StatusServiceUnavailable
					continue
				}
				// Degradação: a liberação fica menos pontual, as respostas não
				// mudam. Reprovar aqui tiraria de operação um serviço correto.
				corpo.Dependencias[v.Nome] = "degradado"
				if corpo.Status == "pronto" {
					corpo.Status = "degradado"
				}
				continue
			}
			corpo.Dependencias[v.Nome] = "ok"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(codigo)
		_ = json.NewEncoder(w).Encode(corpo)
	})

	return mux
}
