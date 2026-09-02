package http

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

func chaveOK() map[string]string { return map[string]string{"X-API-Key": chaveAPI} }

func TestValidarIngressoValido(t *testing.T) {
	a := montarAmbiente(t)
	i := a.repo.semear(t, "ing-1", usuario1, ingresso.Valido, instanteFixo)

	res, corpo := a.postValidar(t, `{"codigo_qr":"`+i.CodigoQR+`"}`, chaveOK())
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, queria 200 (corpo: %s)", res.StatusCode, corpo)
	}
	var v map[string]any
	decodificarEm(t, corpo, &v)
	if v["valido"] != true {
		t.Errorf("valido = %v, queria true", v["valido"])
	}
	if v["ingresso_id"] != "ing-1" {
		t.Errorf("ingresso_id = %v", v["ingresso_id"])
	}
	if v["utilizado_em"] == nil || v["utilizado_em"] == "" {
		t.Error("resposta autorizada sem utilizado_em")
	}
}

func TestValidarIngressoJaUtilizado(t *testing.T) {
	a := montarAmbiente(t)
	i := a.repo.semear(t, "ing-1", usuario1, ingresso.Utilizado, instanteFixo)

	res, corpo := a.postValidar(t, `{"codigo_qr":"`+i.CodigoQR+`"}`, chaveOK())
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, queria 409 (corpo: %s)", res.StatusCode, corpo)
	}
	var v map[string]any
	decodificarEm(t, corpo, &v)
	if v["valido"] != false {
		t.Errorf("valido = %v, queria false", v["valido"])
	}
	if v["mensagem"] != "Ingresso já utilizado anteriormente." {
		t.Errorf("mensagem = %v", v["mensagem"])
	}
}

func TestValidarIngressoCancelado(t *testing.T) {
	a := montarAmbiente(t)
	i := a.repo.semear(t, "ing-1", usuario1, ingresso.Cancelado, instanteFixo)

	res, corpo := a.postValidar(t, `{"codigo_qr":"`+i.CodigoQR+`"}`, chaveOK())
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, queria 409 (corpo: %s)", res.StatusCode, corpo)
	}
}

func TestRecusasDe404SaoIdenticasByteAByte(t *testing.T) {
	a := montarAmbiente(t)
	a.repo.semear(t, "ing-1", usuario1, ingresso.Valido, instanteFixo)

	casos := map[string]string{
		"malformado":          "lixo-sem-forma",
		"assinatura invalida": "CIN1.ing-1.forjada",
		"inexistente":         "CIN1.ing-nao-existe.assinatura",
		"prefixo errado":      "XXX9.ing-1.assinatura",
		"partes a mais":       "CIN1.ing-1.assinatura.extra",
		"vazio":               "",
	}
	var refCorpo []byte
	refStatus := 0
	for nome, codigo := range casos {
		res, corpo := a.postValidar(t, `{"codigo_qr":"`+codigo+`"}`, chaveOK())
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d, queria 404 (corpo: %s)", nome, res.StatusCode, corpo)
			continue
		}
		if refStatus == 0 {
			refStatus, refCorpo = res.StatusCode, corpo
			continue
		}
		if !bytes.Equal(corpo, refCorpo) {
			t.Errorf("%s: corpo difere das demais recusas — vaza qual é o caso.\n  este: %s\n  ref : %s",
				nome, corpo, refCorpo)
		}
	}
}

func TestValidarRecusaCredencial(t *testing.T) {
	a := montarAmbiente(t)
	i := a.repo.semear(t, "ing-1", usuario1, ingresso.Valido, instanteFixo)
	pedido := `{"codigo_qr":"` + i.CodigoQR + `"}`

	casos := map[string]map[string]string{
		"sem chave":           {},
		"chave errada":        {"X-API-Key": "errada"},
		"chave vazia":         {"X-API-Key": ""},
		"token no lugar dela": {"Authorization": "Bearer " + a.token(t, usuario1)},
	}
	var refCorpo []byte
	for nome, cab := range casos {
		res, corpo := a.postValidar(t, pedido, cab)
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, queria 401 (corpo: %s)", nome, res.StatusCode, corpo)
			continue
		}
		if refCorpo == nil {
			refCorpo = corpo
		} else if !bytes.Equal(corpo, refCorpo) {
			t.Errorf("%s: corpo de 401 difere dos demais", nome)
		}
	}

	depois, err := a.repo.BuscarPorID(t.Context(), "ing-1")
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if depois.Status != ingresso.Valido {
		t.Errorf("status = %q; requisição sem credencial não pode dar baixa (FR-012)", depois.Status)
	}
}

func TestValidarCorpoInvalido(t *testing.T) {
	a := montarAmbiente(t)
	casos := map[string]string{
		"sem codigo_qr": `{}`,
		"tipo errado":   `{"codigo_qr":123}`,
		"json quebrado": `{isto nao e json`,
		"corpo vazio":   ``,
	}
	for nome, corpo := range casos {
		t.Run(nome, func(t *testing.T) {
			res, b := a.postValidar(t, corpo, chaveOK())
			if res.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("status = %d, queria 422 (corpo: %s)", res.StatusCode, b)
			}
			if ct := res.Header.Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("content-type = %q, queria application/problem+json", ct)
			}
		})
	}
}

func TestValidacoesSimultaneasAutorizamUmaSo(t *testing.T) {
	a := montarAmbiente(t)
	i := a.repo.semear(t, "ing-1", usuario1, ingresso.Valido, instanteFixo)
	pedido := `{"codigo_qr":"` + i.CodigoQR + `"}`

	const n = 12
	status := make(chan int, n)
	pronto := make(chan struct{})
	for k := 0; k < n; k++ {
		go func() {
			<-pronto
			res, _ := a.postValidar(t, pedido, chaveOK())
			status <- res.StatusCode
		}()
	}
	close(pronto)

	autorizadas, conflitos := 0, 0
	for k := 0; k < n; k++ {
		switch <-status {
		case http.StatusOK:
			autorizadas++
		case http.StatusConflict:
			conflitos++
		}
	}
	if autorizadas != 1 {
		t.Errorf("%d autorizações, queria exatamente 1 (FR-011)", autorizadas)
	}
	if conflitos != n-1 {
		t.Errorf("%d conflitos, queria %d", conflitos, n-1)
	}
}

func TestTokenExpiradoNaoEAceito(t *testing.T) {
	a := montarAmbiente(t)
	expirado := a.token(t, usuario1, func(c jwt.MapClaims) {
		c["exp"] = time.Now().Add(-time.Hour).Unix()
	})
	res, _ := a.get(t, "/api/v1/ingressos/meus-ingressos",
		map[string]string{"Authorization": "Bearer " + expirado})
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, queria 401", res.StatusCode)
	}
}
