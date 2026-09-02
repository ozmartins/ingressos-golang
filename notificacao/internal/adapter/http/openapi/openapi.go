package openapi

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
)

//go:embed openapi.yaml
var Especificacao []byte

const versaoSwaggerUI = "5.17.14"

func HandlerEspecificacao() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		if _, err := w.Write(Especificacao); err != nil {
			slog.Debug("falha ao escrever a especificação", slog.Any("erro", err))
		}
	}
}

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
	literal, err := json.Marshal(caminhoSpec)
	if err != nil {
		literal = []byte(`"/openapi.yaml"`)
	}

	base := "https://cdn.jsdelivr.net/npm/swagger-ui-dist@" + versaoSwaggerUI

	return `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Servico-Notificacao — API</title>
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
