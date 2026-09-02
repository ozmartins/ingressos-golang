// Package openapi serve o contrato publicado do serviço e a interface de
// exploração construída sobre ele.
//
// O contrato não é gerado a partir do código: ele é escrito à mão em
// contracts/openapi.yaml e apenas embutido aqui. A direção importa — o documento
// é o acordo com quem integra, e um acordo derivado da implementação registra o
// que o código faz, não o que ele prometeu fazer.
package openapi

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
)

// Especificacao é o contrato embutido no binário. Exportado para que o teste de
// paridade confronte os paths documentados com as rotas realmente registradas.
//
//go:embed openapi.yaml
var Especificacao []byte

// versaoSwaggerUI fica fixada de propósito. Um `@latest` transformaria a
// atualização de um terceiro em mudança silenciosa da página servida.
const versaoSwaggerUI = "5.17.14"

// HandlerEspecificacao devolve o contrato bruto.
//
// O media type é o registrado pela RFC 9512; navegadores tendem a baixar em vez
// de exibir, o que é o comportamento correto para um artefato de contrato.
func HandlerEspecificacao() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		if _, err := w.Write(Especificacao); err != nil {
			slog.Debug("falha ao escrever a especificação", slog.Any("erro", err))
		}
	}
}

// HandlerUI devolve a página do Swagger UI apontada para caminhoSpec.
//
// A página é montada uma única vez, na composição: o HTML não depende da
// requisição, e recompô-lo por chamada seria trabalho repetido sem ganho algum.
func HandlerUI(caminhoSpec string) http.HandlerFunc {
	pagina := []byte(montarPagina(caminhoSpec))
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(pagina); err != nil {
			slog.Debug("falha ao escrever a página de documentação", slog.Any("erro", err))
		}
	}
}

func montarPagina(caminhoSpec string) string {
	// json.Marshal produz um literal de string JavaScript válido e escapado. O
	// caminho hoje é constante, mas a interpolação crua seria uma armadilha
	// esperando o primeiro caminho vindo de configuração.
	literal, err := json.Marshal(caminhoSpec)
	if err != nil {
		// Marshal de string não falha; o ramo existe só para não engolir o erro.
		literal = []byte(`"/openapi.yaml"`)
	}

	base := "https://cdn.jsdelivr.net/npm/swagger-ui-dist@" + versaoSwaggerUI

	return `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Servico-Pagamento — API</title>
  <link rel="stylesheet" href="` + base + `/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="` + base + `/swagger-ui-bundle.js" crossorigin></script>
  <script src="` + base + `/swagger-ui-standalone-preset.js" crossorigin></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: ` + string(literal) + `,
        dom_id: '#swagger-ui',
        deepLinking: true,
        // Sem isto, o token colado em Authorize se perde a cada recarga e
        // testar a reserva vira repetição manual.
        persistAuthorization: true,
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        layout: 'StandaloneLayout'
      });
    };
  </script>
</body>
</html>
`
}
