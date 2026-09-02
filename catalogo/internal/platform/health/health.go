package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Verificador interface {
	Ping(ctx context.Context) error
}

type resposta struct {
	Status       string            `json:"status"`
	Dependencias map[string]string `json:"dependencias,omitempty"`
}

func Handler(banco Verificador) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		res := resposta{Status: "ok", Dependencias: map[string]string{"banco": "ok"}}
		codigo := http.StatusOK

		if err := banco.Ping(ctx); err != nil {
			slog.WarnContext(ctx, "verificação de saúde falhou", slog.Any("erro", err))
			res.Status = "indisponivel"
			res.Dependencias["banco"] = "indisponivel"
			codigo = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(codigo)
		_ = json.NewEncoder(w).Encode(res)
	}
}
