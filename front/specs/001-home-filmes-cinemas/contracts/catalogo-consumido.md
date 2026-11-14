# Contrato consumido: catálogo (somente leitura)

Fonte da verdade: `catalogo/internal/adapter/http/openapi/openapi.yaml`. O front não expõe nenhuma interface própria além da própria página (URL: `/?aba=filmes|cinemas`).

## GET {BASE}/api/v1/filmes
- `BASE` = `VITE_CATALOGO_URL` ou `/catalogo` (proxy do Vite → `http://localhost:8082`).
- Query usada: `page`, `page_size=100`. Sem autenticação.
- 200: `PaginaFilmes` (`itens: Filme[]`, `pagina`). Repetir com `page+1` enquanto `pagina.tem_proxima`.
- 400: entrada inválida → tratado como erro de carregamento.

## GET {BASE}/api/v1/cinemas
- Query usada: `ativo=true`, `page`, `page_size=100`.
- 200: `PaginaCinemas`.

## Tratamento de falhas
Qualquer status ≠ 200, falha de rede ou JSON inesperado → estado `erro` da aba afetada, com ação "Tentar novamente"; a outra aba não é afetada (FR-012).

## Contrato da URL da página
`/?aba=filmes` (padrão) | `/?aba=cinemas`; valor inválido é tratado como `filmes`.
