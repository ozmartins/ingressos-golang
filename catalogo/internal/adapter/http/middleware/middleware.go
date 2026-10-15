package middleware

import (
	"net/http"
)

func Recuperacao(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"type":"https://cinema.example/errors/erro-interno","title":"Erro interno","status":500,"detail":"Erro interno. Consulte o suporte informando o identificador desta requisição."}`))
			}
		}()

		prox.ServeHTTP(w, r)
	})
}

func Encadear(h http.Handler, camadas ...func(http.Handler) http.Handler) http.Handler {
	for i := len(camadas) - 1; i >= 0; i-- {
		h = camadas[i](h)
	}

	return h
}
