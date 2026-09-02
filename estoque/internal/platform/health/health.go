package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Verificacao struct {
	Nome      string
	Essencial bool
	Checar    func(context.Context) error
}

type Servico struct {
	verificacoes []Verificacao
}

func Novo(verificacoes ...Verificacao) *Servico { return &Servico{verificacoes: verificacoes} }

type resposta struct {
	Status       string            `json:"status"`
	Dependencias map[string]string `json:"dependencias"`
}

func (s *Servico) Handler() http.Handler {
	mux := http.NewServeMux()

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
					corpo.Dependencias[v.Nome] = "indisponivel"
					corpo.Status = "indisponivel"
					codigo = http.StatusServiceUnavailable
					continue
				}
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
