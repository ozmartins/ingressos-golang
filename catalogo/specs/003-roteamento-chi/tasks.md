# Tasks: Roteamento do Catálogo via chi

**Input**: `specs/003-roteamento-chi/` (plan.md, spec.md, research.md, quickstart.md)

**Tests**: incluídos — o princípio VIII da constituição exige teste para a API exposta, e a
equivalência com o ServeMux só é verificável por teste (research D7).

**Formato**: `- [ ] T### [P?] [US?] Descrição com caminho`. `[P]` = arquivos distintos, sem
dependência de tarefa incompleta. Todos os caminhos são relativos a `ingressos-golang/catalogo/`.

## Phase 1: Setup

- [X] T001 Rodar `make test` e `go test ./test/contract/...` na `master` limpa e anotar o resultado como linha de base (se algo já falha, parar e avisar o mantenedor antes de mexer).
- [X] T002 Adicionar `github.com/go-chi/chi/v5 v5.3.2` a `go.mod`/`go.sum` (`GOFLAGS=-mod=mod GOPROXY=off go get github.com/go-chi/chi/v5@v5.3.2`; o módulo já está no cache local).
- [X] T003 [P] Acrescentar `github.com/go-chi` à regra `nucleo-sem-adaptadores` do depguard em `.golangci.yml`, com `desc` no padrão das demais ("roteador HTTP é detalhe de adaptador").

---

## Phase 2: User Story 1 - Comportamento HTTP idêntico após a troca (Priority: P1) 🎯 MVP

**Goal**: mesmas rotas, parâmetros, proteção, middlewares e rótulo `rota` com o chi.

**Independent Test**: `go test -race ./internal/adapter/http/... ./test/contract/...` verde, sem asserção enfraquecida; log "requisição atendida" com `rota="GET /api/v1/filmes/{id}"`.

### Tests for User Story 1

- [X] T004 [US1] Criar `internal/adapter/http/router_test.go` com um teste que percorre `Rotas()` e, para cada rota documentada, monta `NovoRouter` com handlers que registram a chamada e verifica: (a) o handler da rota é atingido; (b) `r.PathValue("id")` chega com o valor enviado; (c) rota `Protegida` sem credencial devolve a recusa de não autenticado, e rota pública não exige credencial. (Escrever antes da troca: deve passar com o ServeMux e continuar passando com o chi.)
- [X] T005 [US1] Em `internal/adapter/http/router_test.go`, acrescentar teste que captura o `slog` e prova que `middleware.Log` registra `rota="GET /api/v1/filmes/{id}"` (com método, sem o identificador) para uma requisição a `/api/v1/filmes/<uuid>` — é o caso do achado D3/D4.1 e deve falhar se o roteador for colocado por fora dos middlewares ou o `Pattern` perder o método.

### Implementation for User Story 1

- [X] T006 [US1] Reescrever `NovoRouter` em `internal/adapter/http/router.go`: `r := chi.NewRouter()`; `r.Use(middleware.Telemetria, middleware.Recuperacao, middleware.Log(d.Metricas), chimw.GetHead)` (importar `github.com/go-chi/chi/v5/middleware` como `chimw`); laço sobre `rotas` com `r.Method(rota.Metodo, rota.Caminho, h)`, onde `h` (já protegido quando `Protegida`) é embrulhado por função que define `req.Pattern = rota.Metodo + " " + rota.Caminho` antes de chamar o handler (research D3, D4.1, D4.2). Não alterar `rotas`, `Rota`, `Rotas()`, `Dependencias`, `handlers.go` nem `middleware/`; retornar `r` como `http.Handler`.
- [X] T007 [US1] Rodar `go test -race ./internal/adapter/http/... ./test/contract/...`; T004 e T005 e toda a suíte existente devem passar sem edição de asserções.

**Checkpoint**: US1 entregue — o catálogo já roda sobre o chi com paridade de rotas, parâmetros, proteção e rótulos.

---

## Phase 3: User Story 2 - Rotas inexistentes e métodos não permitidos (Priority: P2)

**Goal**: 404, 405 e HEAD iguais aos do ServeMux (decisão do mantenedor: preservar).

**Independent Test**: os testes de borda de T008 passam; os `curl` do passo 3 do quickstart dão o mesmo que na versão anterior.

### Tests for User Story 2

- [X] T008 [US2] Em `internal/adapter/http/router_test.go`, testes de borda: (a) caminho inexistente → 404 `404 page not found`; (b) `POST` em `/api/v1/filmes/{id}` → 405 com `Allow: DELETE, GET, HEAD, PUT` (ordem alfabética, HEAD incluído porque GET casa) e corpo `Method Not Allowed\n`; (c) `HEAD /api/v1/filmes` → 200 sem corpo; (d) `/docs` e `/docs/` → 200 sem autenticação; (e) pânico num handler vira o `problem+json` de erro interno também quando a requisição cai em 404/405 (middlewares envolvem tudo).
- [X] T009 [US2] Em `internal/adapter/http/router_test.go`, teste de caracterização de D5: `GET /api/v1/filmes/x%2Fy` e id não-UUID devolvem a mesma categoria de erro de validação que um id inválido simples; `//` e `/../` apenas documentam o resultado do chi (404) com comentário apontando para research D5. Se o resultado de `%2F` diferir do esperado, parar e levar ao mantenedor (princípio X).

### Implementation for User Story 2

- [X] T010 [US2] Em `internal/adapter/http/router.go`, registrar `r.MethodNotAllowed(...)` com handler próprio: para cada `m` em GET, POST, PUT, DELETE chamar `rctx.Routes.Match(chi.NewRouteContext(), m, r.URL.Path)` (`rctx := chi.RouteContext(r.Context())`), acrescentar `HEAD` se GET casar, ordenar, gravar `Allow` com `strings.Join(..., ", ")` e responder `http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)` (research D4.3). Não registrar `NotFound` (o padrão do chi já é `http.NotFound`).
- [X] T011 [US2] Rodar `go test -race ./internal/adapter/http/...`; T008 e T009 verdes.

**Checkpoint**: US1 + US2 — bordas idênticas às de hoje, exceto as residuais de D5.

---

## Phase 4: User Story 3 - Contrato verificado e fronteira do núcleo mantida (Priority: P3)

**Goal**: paridade com o OpenAPI e depguard continuam valendo; documentação alinhada ao código.

**Independent Test**: `make lint` e o teste de paridade (`docs_test.go`) passam; nenhum pacote do núcleo importa o chi.

- [X] T012 [US3] Atualizar a spec 001 para refletir o chi, como foi feito para o GORM na 002 (plan.md, "Divergências" item 2; **sujeito a confirmação do mantenedor**): `specs/001-catalogo-sessoes-reserva/research.md` (linhas 15–19: decisão passa a chi v5, com o motivo e o ServeMux como alternativa substituída), `plan.md` (linhas 15 e 102) e `tasks.md` (T020).
- [X] T013 [P] [US3] Conferir `README.md` (e `ers-catalogo.md`, só leitura): trocar menção ao roteador/ServeMux, se existir; o trecho da linha ~221 sobre paridade de rotas continua valendo.
- [X] T014 [US3] Provar que a regra do depguard funciona: importar temporariamente `github.com/go-chi/chi/v5` num arquivo de `internal/usecase/`, rodar `make lint`, confirmar a falha com a `desc` de T003, e **reverter** a importação.
- [X] T015 [US3] Rodar `make lint` limpo e `go test -race ./internal/adapter/http/ -run Contrato` (paridade com o OpenAPI) sem alteração do contrato versionado. (`make lint`: 19 apontamentos, todos preexistentes e fora dos arquivos desta feature; o depguard passa)

---

## Phase 5: Polish & Cross-Cutting

- [X] T016 `make test` completo e `make test-integration` (o `reservar_test.go` monta `NovoRouter`); SC-001.
- [ ] T017 Executar o quickstart (passos 3 e 4) com `docker compose up --build catalogo`, conferir 405/HEAD/404/`/docs/` e o rótulo `rota` no log; SC-003.
- [ ] T018 Levar ao mantenedor as divergências residuais de research D5 (limpeza de caminho `//`/`..` → 307 vs 404; `%2F` decodificado vs bruto), com o resultado de T009, e registrar a decisão em `research.md` — sem corrigir nem aceitar por conta própria.
- [ ] T019 Commitar direto na `master` (sem branch de feature), mensagem em inglês no padrão Conventional Commits; push só se pedido.

---

## Dependencies & Execution Order

- Phase 1: T001 → T002; T003 é [P] (arquivo distinto).
- US1 (T004–T007) depende de T002. T004/T005 antes de T006; T006 antes de T007.
- US2 (T008–T011) depende de US1 (mesmo `router.go`/`router_test.go`): T008/T009 → T010 → T011.
- US3 (T012–T015) depende de T003; T012 e T013 independem do código e podem ocorrer a qualquer momento após a Phase 1; T014 → T015.
- Polish depende de tudo.

### Parallel opportunities

- T003 junto com T002; T012 e T013 junto com US1/US2 (só documentação).
- T004 e T005 editam o mesmo arquivo — sequenciais.

## Implementation Strategy

MVP = Phase 1 + US1: já entrega o catálogo no chi com paridade de rotas, parâmetros, proteção e rótulos. US2 fecha as bordas (405/HEAD) e US3 as garantias de contrato e de fronteira. Parar após cada checkpoint e rodar a suíte.
