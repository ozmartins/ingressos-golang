package http

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/http/openapi"
	"go.yaml.in/yaml/v3"
)

type documento struct {
	Servers []servidor                      `yaml:"servers"`
	Paths   map[string]map[string]yaml.Node `yaml:"paths"`
}

type servidor struct {
	URL string `yaml:"url"`
}

var metodosHTTP = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"patch": true, "head": true, "options": true, "trace": true,
}

func TestContratoEParidadeComAsRotasRegistradas(t *testing.T) {
	var doc documento
	if err := yaml.Unmarshal(openapi.Especificacao, &doc); err != nil {
		t.Fatalf("a especificação embutida não é YAML válido: %v", err)
	}
	if len(doc.Servers) == 0 {
		t.Fatal("a especificação não declara servers; o prefixo das rotas é indeterminado")
	}
	prefixoPadrao := strings.TrimSuffix(doc.Servers[0].URL, "/")

	documentadas := map[string]bool{}
	for caminho, item := range doc.Paths {
		prefixo := prefixoPadrao
		if node, ok := item["servers"]; ok {
			var locais []servidor
			if err := node.Decode(&locais); err != nil {
				t.Fatalf("decodificando servers de %s: %v", caminho, err)
			}
			if len(locais) > 0 {
				prefixo = strings.TrimSuffix(locais[0].URL, "/")
			}
		}
		for chave := range item {
			if metodosHTTP[chave] {
				documentadas[strings.ToUpper(chave)+" "+prefixo+caminho] = true
			}
		}
	}

	registradas := map[string]bool{}
	for _, rota := range Rotas() {
		if rota.Documentada {
			registradas[rota.Metodo+" "+rota.Caminho] = true
		}
	}

	for chave := range registradas {
		if !documentadas[chave] {
			t.Errorf("rota registrada e ausente do contrato: %s", chave)
		}
	}
	for chave := range documentadas {
		if !registradas[chave] {
			t.Errorf("contrato descreve rota que o roteador não registra: %s", chave)
		}
	}

	if t.Failed() {
		t.Logf("registradas: %v", ordenar(registradas))
		t.Logf("documentadas: %v", ordenar(documentadas))
	}
}

func ordenar(m map[string]bool) []string {
	saida := make([]string, 0, len(m))
	for k := range m {
		saida = append(saida, k)
	}
	sort.Strings(saida)
	return saida
}

func TestDocumentacaoEhServidaSemAutenticacao(t *testing.T) {
	router := NovoRouter(Dependencias{
		Saude: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
	})

	casos := []struct {
		caminho      string
		tipoEsperado string
		contem       string
	}{
		{"/docs", "text/html", "/openapi.yaml"},
		{"/docs/", "text/html", "swagger-ui"},
		{"/openapi.yaml", "yaml", "openapi: 3.1.0"},
	}

	for _, caso := range casos {
		t.Run(caso.caminho, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caso.caminho, nil))

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, esperado 200", w.Code)
			}
			if tipo := w.Header().Get("Content-Type"); !strings.Contains(tipo, caso.tipoEsperado) {
				t.Errorf("Content-Type = %q, esperado conter %q", tipo, caso.tipoEsperado)
			}
			if corpo := w.Body.String(); !strings.Contains(corpo, caso.contem) {
				t.Errorf("corpo não contém %q", caso.contem)
			}
		})
	}
}
