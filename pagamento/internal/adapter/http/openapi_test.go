package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"gopkg.in/yaml.v3"
)

const caminhoContrato = "../../../specs/001-pagamento-assincrono/contracts/openapi.yaml"

func carregarContrato(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(caminhoContrato))
	if err != nil {
		t.Fatalf("contrato não encontrado — a API não pode divergir dele: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func esquema(t *testing.T, doc map[string]any, nome string) map[string]any {
	t.Helper()
	comps := doc["components"].(map[string]any)
	esquemas := comps["schemas"].(map[string]any)
	e, ok := esquemas[nome].(map[string]any)
	if !ok {
		t.Fatalf("esquema %q ausente do contrato", nome)
	}
	return e
}

func chaves(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// A resposta 200 tem exatamente os campos que o contrato declara — nem a menos
// (quebraria o cliente), nem a mais (vazaria detalhe não contratado).
func TestResposta200BateComOContrato(t *testing.T) {
	doc := carregarContrato(t)
	props := esquema(t, doc, "Pagamento")["properties"].(map[string]any)

	reserva := uuid.NewString()
	api := apiCom(repoStub{t: transacaoDe(transacao.Pago, reserva)})
	w := chamar(t, api, reserva, token(t, dona, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", w.Code)
	}

	var corpo map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
		t.Fatal(err)
	}
	if got, want := chaves(corpo), chaves(props); !reflect.DeepEqual(got, want) {
		t.Fatalf("campos divergem do contrato:\n  API: %v\n  contrato: %v", got, want)
	}
}

// Todo estado que a API pode devolver consta do enum do contrato, e vice-versa.
func TestEnumDeStatusBateComODominio(t *testing.T) {
	doc := carregarContrato(t)
	props := esquema(t, doc, "Pagamento")["properties"].(map[string]any)
	statusProp := props["status"].(map[string]any)

	var doContrato []string
	for _, v := range statusProp["enum"].([]any) {
		doContrato = append(doContrato, v.(string))
	}
	sort.Strings(doContrato)

	doDominio := []string{
		string(transacao.Processando), string(transacao.Pago), string(transacao.Recusado),
		string(transacao.Cancelado), string(transacao.PendenteVerificacao),
	}
	sort.Strings(doDominio)

	if !reflect.DeepEqual(doContrato, doDominio) {
		t.Fatalf("enum de status divergente:\n  contrato: %v\n  domínio: %v", doContrato, doDominio)
	}
}

func TestEnumDeFormaBateComODominio(t *testing.T) {
	doc := carregarContrato(t)
	props := esquema(t, doc, "Pagamento")["properties"].(map[string]any)
	formaProp := props["forma_pagamento"].(map[string]any)

	var doContrato []string
	for _, v := range formaProp["enum"].([]any) {
		doContrato = append(doContrato, v.(string))
	}
	sort.Strings(doContrato)

	doDominio := []string{string(transacao.PIX), string(transacao.CartaoCredito)}
	sort.Strings(doDominio)

	if !reflect.DeepEqual(doContrato, doDominio) {
		t.Fatalf("enum de forma divergente:\n  contrato: %v\n  domínio: %v", doContrato, doDominio)
	}
}

// O corpo de erro tem a forma que o contrato declara.
func TestEsquemaDeErroBateComOContrato(t *testing.T) {
	doc := carregarContrato(t)
	props := esquema(t, doc, "Erro")["properties"].(map[string]any)

	api := apiCom(repoStub{t: transacaoDe(transacao.Pago, "x")})
	w := chamar(t, api, "nao-uuid", token(t, dona, nil))

	var corpo map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
		t.Fatal(err)
	}
	if got, want := chaves(corpo), chaves(props); !reflect.DeepEqual(got, want) {
		t.Fatalf("corpo de erro divergente:\n  API: %v\n  contrato: %v", got, want)
	}
}

// Todo caminho declarado no contrato existe de fato na API.
func TestCaminhosDeclaradosExistem(t *testing.T) {
	doc := carregarContrato(t)
	caminhos := doc["paths"].(map[string]any)
	api := apiCom(repoStub{t: transacaoDe(transacao.Pago, "x")})

	for caminho := range caminhos {
		url := "/api/v1" + caminho
		if caminho == "/pagamentos/reserva/{reserva_id}" {
			url = "/api/v1/pagamentos/reserva/" + uuid.NewString()
		}
		w := chamarURL(t, api, url, token(t, dona, nil))
		if w.Code == http.StatusNotFound && w.Body.Len() == 0 {
			t.Fatalf("caminho %q declarado no contrato não existe na API", caminho)
		}
	}
	_ = time.Now
}
