package http

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

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
