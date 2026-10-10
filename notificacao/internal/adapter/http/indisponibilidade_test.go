package http

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

func TestFalhaDeInfraNaListagemDa503(t *testing.T) {
	a := montarAmbiente(t)
	a.repo.falha = fmt.Errorf("%w: banco fora do ar", usecase.ErrDependenciaIndisponivel)

	res, corpo := a.get(t, rota, comToken(t, a, usuario1))
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, queria 503 (corpo: %s)", res.StatusCode, corpo)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, queria application/problem+json", ct)
	}
}

func TestFalhaDeInfraNaValidacaoDa503(t *testing.T) {
	a := montarAmbiente(t)
	a.repo.falha = fmt.Errorf("%w: banco fora do ar", usecase.ErrDependenciaIndisponivel)

	cab := map[string]string{"X-API-Key": chaveAPI}
	res, corpo := a.postValidar(t, `{"codigo_qr":"`+assinadorFalso{}.Gerar("ing-1")+`"}`, cab)
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, queria 503 (corpo: %s)", res.StatusCode, corpo)
	}
}
