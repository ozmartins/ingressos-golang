package http

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

// instanciaDoContexto devolve o identificador de rastreamento da requisição no
// formato de URN, para o campo `instance` do problem+json. É a chave que liga a
// reclamação de quem integra ao rastro correspondente (SC-011).
func instanciaDoContexto(r *http.Request) string {
	if r == nil {
		return ""
	}
	sc := trace.SpanContextFromContext(r.Context())
	if !sc.IsValid() {
		return ""
	}
	return "urn:trace:" + sc.TraceID().String()
}
