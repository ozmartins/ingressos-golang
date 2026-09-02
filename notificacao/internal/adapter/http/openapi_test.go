package http

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"gopkg.in/yaml.v3"
)

const caminhoContrato = "../../../specs/001-emissao-ingressos/contracts/openapi.yaml"

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

func TestTodasAsRotasDoContratoExistem(t *testing.T) {
	doc := carregarContrato(t)
	a := montarAmbiente(t)
	caminhos := doc["paths"].(map[string]any)

	for caminho := range caminhos {
		metodos := caminhos[caminho].(map[string]any)
		for metodo := range metodos {
			t.Run(metodo+" "+caminho, func(t *testing.T) {
				req, err := http.NewRequest(
					map[string]string{"get": http.MethodGet, "post": http.MethodPost}[metodo],
					a.srv.URL+caminho, nil)
				if err != nil {
					t.Fatal(err)
				}
				res, _ := a.enviar(t, req, nil)
				if res.StatusCode == http.StatusNotFound && caminho != "/api/v1/ingressos/validar" {
					t.Errorf("rota do contrato não existe no servidor (404)")
				}
				if res.StatusCode == http.StatusMethodNotAllowed {
					t.Errorf("o servidor não aceita %s nesta rota", metodo)
				}
			})
		}
	}
}

func TestCamposDaListagemBatemComOContrato(t *testing.T) {
	doc := carregarContrato(t)
	comps := doc["components"].(map[string]any)["schemas"].(map[string]any)
	esperados := chavesDe(comps["Ingresso"].(map[string]any)["properties"].(map[string]any))

	a := montarAmbiente(t)
	a.repo.semear(t, "ing-1", usuario1, ingresso.Valido, instanteFixo)
	_, corpo := a.get(t, rota, comToken(t, a, usuario1))

	lista := decodificarLista(t, corpo)
	if len(lista) != 1 {
		t.Fatalf("%d ingressos", len(lista))
	}
	entregues := chavesDe(lista[0])

	if !mesmasChaves(esperados, entregues) {
		t.Errorf("campos divergem do contrato:\n  contrato: %v\n  servidor: %v", esperados, entregues)
	}
	if _, err := time.Parse(time.RFC3339, lista[0]["criado_em"].(string)); err != nil {
		t.Errorf("criado_em não está em RFC 3339: %v", lista[0]["criado_em"])
	}
}

func TestVereditoDaPortariaBateComOContrato(t *testing.T) {
	doc := carregarContrato(t)
	comps := doc["components"].(map[string]any)["schemas"].(map[string]any)
	obrigatoriosRecusa := listaDeStrings(comps["Recusa"].(map[string]any)["required"])

	a := montarAmbiente(t)
	i := a.repo.semear(t, "ing-1", usuario1, ingresso.Valido, instanteFixo)

	_, corpo := a.postValidar(t, `{"codigo_qr":"`+i.CodigoQR+`"}`, chaveOK())
	var ok map[string]any
	decodificarEm(t, corpo, &ok)
	for _, campo := range []string{"valido", "mensagem", "ingresso_id", "utilizado_em"} {
		if _, tem := ok[campo]; !tem {
			t.Errorf("resposta 200 sem o campo %q", campo)
		}
	}

	_, corpoRecusa := a.postValidar(t, `{"codigo_qr":"lixo"}`, chaveOK())
	var recusa map[string]any
	decodificarEm(t, corpoRecusa, &recusa)
	if !mesmasChaves(obrigatoriosRecusa, chavesDe(recusa)) {
		t.Errorf("corpo da recusa divergiu do contrato:\n  contrato: %v\n  servidor: %v",
			obrigatoriosRecusa, chavesDe(recusa))
	}
}

func TestErroDeProtocoloUsaProblemJSON(t *testing.T) {
	a := montarAmbiente(t)
	res, corpo := a.get(t, rota+"?status=INVENTADO", comToken(t, a, usuario1))
	if ct := res.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content-type = %q, queria application/problem+json", ct)
	}
	var p map[string]any
	decodificarEm(t, corpo, &p)
	for _, campo := range []string{"type", "title", "status"} {
		if _, ok := p[campo]; !ok {
			t.Errorf("resposta de problema sem o campo obrigatório %q (RFC 9457)", campo)
		}
	}
}

func chavesDe[T any](m map[string]T) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func listaDeStrings(v any) []string {
	itens, _ := v.([]any)
	out := make([]string, 0, len(itens))
	for _, i := range itens {
		out = append(out, i.(string))
	}
	sort.Strings(out)
	return out
}

func mesmasChaves(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
