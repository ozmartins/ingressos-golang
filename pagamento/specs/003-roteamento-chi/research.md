# Research: Roteamento do Pagamento via chi

## D1 — chi v5 sem middlewares

**Decision**: `chi.NewRouter()` e registro direto das rotas, sem `Use(...)`.
**Rationale**: o `ServeMux` atual não tem camada transversal (`main.go` entrega `api.Rotas()` direto ao `http.Server`). Middlewares do chi (`Logger`, `Recoverer`, `RequestID`) mudariam comportamento observável; a Constituição I proíbe complexidade não pedida.
**Alternatives**: usar `middleware.Recoverer` por "boa prática" — rejeitado, escopo novo.

## D2 — 404 e 405

**Decision**: caracterizar por teste o que o `ServeMux` devolve hoje (404: `404 page not found`; 405: `Method Not Allowed` com cabeçalho `Allow`, ambos `text/plain`) e reproduzir com `r.NotFound` / `r.MethodNotAllowed`. O chi, por padrão, devolve 405 sem corpo e só preenche `Allow` em versões recentes; o `Allow` precisa ser calculado a partir do contexto de rotas se não vier pronto.
**Rationale**: FR-005 exige respostas iguais ou decisão do mantenedor.
**Alternatives**: aceitar o padrão do chi — só com aprovação do mantenedor.

## D3 — `/docs` e `/docs/`

**Decision**: registrar `GET /docs` e `GET /docs/` como rotas exatas, ambas para `openapi.HandlerUI`.
**Rationale**: no `ServeMux`, `GET /docs/` casa também `/docs/qualquer`; no chi casa só `/docs/`. A caracterização mostra se há consumidor de subcaminhos; se `HandlerUI` ignora o caminho, `/docs/x` hoje devolve a UI e passará a 404 — diferença a verificar e, se existir, preservar com `r.Handle("/docs/*", ...)` ou levar ao mantenedor.

## D4 — `HEAD`

**Decision**: medir. O `ServeMux` responde `HEAD` em padrões `GET`; o chi responde 405. Se confirmado, preservar com tratamento explícito (ex.: `Head` apontando aos mesmos handlers nas rotas GET) ou decisão do mantenedor.

## D5 — Fronteira do núcleo

**Decision**: acrescentar `github.com/go-chi/chi` à regra `nucleo-sem-adaptadores` do `.golangci.yml`, como já é feito com `net/http`, gorm e pgx.

## D6 — Parâmetro de caminho

**Decision**: `chi.URLParam(r, "reserva_id")` substitui `r.PathValue("reserva_id")`. Os handlers continuam `func(http.ResponseWriter, *http.Request)`, então os testes que chamam `api.Rotas().ServeHTTP` não mudam. Atenção: testes que invoquem um handler diretamente com `SetPathValue` precisam passar a rotear pelo `Rotas()`.

## Caracterização medida (T006)

Medido no `http.ServeMux` antes da troca (testes em `internal/adapter/http/roteamento_test.go`):

- 404: `text/plain; charset=utf-8`, corpo `404 page not found\n`; barra final divergente nas rotas de API também dá 404.
- 405: `text/plain; charset=utf-8`, corpo `Method Not Allowed\n`, `Allow` em ordem alfabética e incluindo `HEAD` onde há `GET` (ex.: `GET, HEAD, POST`).
- `HEAD` em rotas `GET`: 200 (o `ServeMux` atende).
- `/docs`, `/docs/` e `/docs/qualquer`: todos 200 com a UI.

Resultado no chi v5.3.2: o padrão difere (405 sem corpo, `Allow` repetido, sem `HEAD`, `/docs/` exato). Tudo foi preservado com `HEAD` registrado junto de cada `GET`, `/docs/*`, `NotFound` = `http.NotFound` e `MethodNotAllowed` próprio que calcula `Allow` via `Match`. Nenhuma diferença remanescente; nada a levar ao mantenedor.
