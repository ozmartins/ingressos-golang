# Research: Roteamento HTTP via chi

Referência de comportamento: `http.ServeMux` com padrões `"MÉTODO /caminho"` (Go 1.22+), tal como em `internal/adapter/http/handlers.go`.

## D1 — Versão e módulo

- **Decisão**: `github.com/go-chi/chi/v5` v5.3.2.
- **Rationale**: última versão disponível localmente (cache de módulos), módulo v5 estável, sem dependências transitivas.
- **Alternativas**: manter o `ServeMux` (rejeitado: contraria o pedido); gorilla/mux, echo (não pedidos).

## D2 — HEAD

- **Fato**: padrões `GET` do `ServeMux` também atendem `HEAD`; o chi não (HEAD → 405).
- **Decisão**: `r.Use(middleware.GetHead)` (pacote `chi/middleware`, mesmo módulo) para rotear HEAD ao tratador GET.
- **Alternativa**: registrar `Head` por rota (verboso, propenso a esquecimento).

## D3 — 404 e 405

- **Fato**: 404 equivalente (`404 page not found`, `text/plain`). No 405 o `ServeMux` envia `Allow: GET, HEAD` (ou `POST`) e corpo `Method Not Allowed`; o chi envia 405 sem `Allow`.
- **Decisão**: `r.MethodNotAllowed(...)` próprio que descobre os métodos aceitos para o caminho via `chi.RouteContext` + `Mux.Match` (API pública) sobre `GET, HEAD, POST`, define `Allow` na ordem do `ServeMux` e responde 405 com o mesmo corpo. 404 usa o padrão do chi.
- **Alternativa**: aceitar 405 sem `Allow` (mais simples, mas regressão de contrato HTTP; fica como opção do mantenedor).

## D4 — `/docs` e `/docs/`

- **Fato**: `"GET /docs/"` no `ServeMux` é padrão de subárvore: serve `/docs/qualquer-coisa`. No chi, `/docs/` é caminho exato.
- **Decisão**: registrar `GET /docs` e `GET /docs/*` para preservar o comportamento atual (inclusive subcaminhos).

## D5 — Barra final e demais rotas

- `ServeMux` não normaliza barra final nas rotas exatas; o chi também não (sem `StripSlashes`). Nenhum middleware de normalização será adicionado.

## D6 — Autenticação

- Feita dentro dos tratadores (`a.Auth.Identificar`, `a.Chave.Autorizar`); permanece assim. Não há grupos/middlewares de auth no chi, logo rotas públicas e protegidas não mudam.

## D7 — Parâmetros de rota

- Nenhuma rota usa `PathValue`/parâmetros; sem impacto.

## D8 — Testes

- Teste de caracterização escrito **antes** da troca (rodado contra `ServeMux`, depois contra chi): 404, 405+`Allow`, HEAD, `/docs/sub`, rotas públicas sem credencial.
