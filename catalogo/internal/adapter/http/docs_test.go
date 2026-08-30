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

// documento é a fatia do OpenAPI que interessa à paridade: quais caminhos
// existem, sob qual servidor, e com quais métodos.
type documento struct {
	Servers []servidor                      `yaml:"servers"`
	Paths   map[string]map[string]yaml.Node `yaml:"paths"`
}

type servidor struct {
	URL string `yaml:"url"`
}

// metodosHTTP são as chaves de um path item que denotam operação. As demais
// (servers, parameters, summary...) descrevem o caminho, não uma operação.
var metodosHTTP = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"patch": true, "head": true, "options": true, "trace": true,
}

// A documentação só vale enquanto descreve o serviço que está no ar. Este teste
// é o que impede que ela envelheça em silêncio: rota nova sem contrato, ou
// contrato descrevendo rota que não existe mais, quebra o build.
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
		// Um path item pode sobrescrever o servidor — é como /health fica fora
		// do prefixo versionado.
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
	// Só o mínimo para o roteador montar. O Verificador fica nil de propósito:
	// se alguma destas rotas passasse pelo middleware de autenticação, ela
	// entraria em pânico em vez de responder — que é exatamente o sinal desejado.
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
