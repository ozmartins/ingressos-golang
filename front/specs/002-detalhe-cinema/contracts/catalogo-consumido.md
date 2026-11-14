# Contrato consumido: catálogo (somente leitura)

Fonte da verdade: `catalogo/internal/adapter/http/openapi/openapi.yaml`. `BASE` = `VITE_CATALOGO_URL` ou `/catalogo` (proxy do Vite → `http://localhost:8082`). Sem autenticação.

## GET {BASE}/api/v1/cinemas/{id}
- 200: `Cinema`. 404/400 → "cinema não encontrado". `ativo=false` → também "não encontrado".

## GET {BASE}/api/v1/salas?cinema_id={id}&ativo=true&page&page_size=100
- 200: `PaginaSalas`. Percorrer páginas enquanto `pagina.tem_proxima`.

## GET {BASE}/api/v1/sessoes?cinema_id={id}&page&page_size=100
- 200: `PaginaSessoes` (`SessaoNaGrade[]`), ordenada por início; exclui canceladas e finalizadas.

## GET {BASE}/api/v1/filmes?page&page_size=100
- 200: `PaginaFilmes` (todos os status; o filtro por `EM_CARTAZ` é feito no cliente). Usado para enriquecer os filmes das sessões e para manter apenas os `EM_CARTAZ` (FR-013). Falha aqui é erro da programação (com "Tentar novamente").

## Falhas
Status ≠ 200, rede ou JSON inesperado nas consultas de cinema/salas/sessões → estado `erro` com "Tentar novamente".

## Contrato da URL da página
- `/cinemas/{id}`: detalhe do cinema. O servidor de desenvolvimento/preview deve devolver `index.html` para esse caminho (comportamento padrão do Vite SPA).
- `/?aba=filmes|cinemas`: página inicial (inalterado).
