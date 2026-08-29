package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
)

type chaveContexto string

const chaveUsuario chaveContexto = "usuario_id"

// UsuarioDoContexto devolve a identidade colocada pelo middleware.
func UsuarioDoContexto(ctx context.Context) string {
	v, _ := ctx.Value(chaveUsuario).(string)
	return v
}

// VerificadorDeCredencial é a porta mínima que o middleware precisa.
type VerificadorDeCredencial interface {
	Verificar(ctx context.Context, token string) (identidade.Identidade, error)
}

// Autenticacao exige credencial válida e coloca a identidade no contexto.
//
// A recusa acontece antes do handler: uma requisição sem credencial nunca chega
// ao caso de uso e, portanto, nunca alcança o Servico-Estoque.
func Autenticacao(v VerificadorDeCredencial, aoRecusar func(http.ResponseWriter, *http.Request, string)) func(http.Handler) http.Handler {
	return func(prox http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := tokenDoCabecalho(r)
			if !ok {
				aoRecusar(w, r, "Credencial ausente ou malformada.")
				return
			}
			id, err := v.Verificar(r.Context(), token)
			if err != nil {
				// O motivo específico fica no log; ao cliente, apenas a recusa.
				slog.WarnContext(r.Context(), "credencial recusada", slog.Any("erro", err))
				aoRecusar(w, r, "Credencial inválida ou expirada.")
				return
			}
			ctx := context.WithValue(r.Context(), chaveUsuario, id.UsuarioID)
			prox.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func tokenDoCabecalho(r *http.Request) (string, bool) {
	cabecalho := r.Header.Get("Authorization")
	if cabecalho == "" {
		return "", false
	}
	partes := strings.SplitN(cabecalho, " ", 2)
	if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") || strings.TrimSpace(partes[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(partes[1]), true
}
