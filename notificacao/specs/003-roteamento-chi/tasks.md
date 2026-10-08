# Tasks: Roteamento HTTP da Notificação via chi

**Input**: `/specs/003-roteamento-chi/` (plan.md, spec.md, research.md, quickstart.md)

**Tests**: obrigatórios (constituição, princípio II). Caminhos relativos a `ingressos-golang/notificacao/`.

## Phase 1: Setup

- [x] T001 Adicionar `github.com/go-chi/chi/v5` v5.3.2 em `go.mod`/`go.sum` (`go get github.com/go-chi/chi/v5@v5.3.2`; só fica indireto até o uso, rodar `go mod tidy` ao final de T005)

## Phase 2: Foundational — rede de segurança (antes de trocar o roteador)

- [x] T002 Criar `internal/adapter/http/roteamento_test.go` com testes de caracterização contra `api.Rotas()` (usando `montarAmbiente` de `ambiente_test.go`): 404 em caminho inexistente (status + corpo `404 page not found`); 405 em `GET /api/v1/ingressos/validar` e `POST /health/live` com cabeçalho `Allow`; `HEAD /health/live` → 200 sem corpo; `GET /docs`, `/docs/` e `/docs/qualquer` → 200 HTML; `/health/*`, `/openapi.yaml`, `/docs` sem credencial → 200; `meus-ingressos` sem token → 401 `application/problem+json`, `validar` sem chave → 401
- [x] T003 Rodar `go test -race ./internal/adapter/http/...` contra o `ServeMux` atual e ajustar APENAS as expectativas de T002 até passarem (documenta o comportamento de referência: valores exatos de `Allow` e corpos)

**Checkpoint**: referência capturada e verde com o roteador antigo.

## Phase 3: User Story 1 — Contrato HTTP idêntico (P1) 🎯 MVP

**Goal**: `Rotas()` usa chi, contrato inalterado.
**Independent Test**: suíte HTTP existente + T002 passam sem mudar asserções.

- [x] T004 [US1] Em `internal/adapter/http/handlers.go`, reescrever `Rotas()` com `chi.NewRouter()`: `r.Use(middleware.GetHead)`; `r.Get` para `/api/v1/ingressos/meus-ingressos`, `/health/live`, `/health/ready`, `/openapi.yaml`, `/docs`, `/docs/*`; `r.Post` para `/api/v1/ingressos/validar`; retorno continua `http.Handler`; remover o uso de `http.NewServeMux`
- [x] T005 [US1] `go mod tidy`, depois `go build ./...` e `go test -race ./internal/adapter/http/...`; confirmar que testes existentes (`listagem_test.go`, `validar_test.go`, `openapi_test.go`, `ambiente_test.go`) passam sem alteração de asserções

**Checkpoint**: US1 entregue; só 404/405 podem divergir (tratado em US2).

## Phase 4: User Story 2 — 404/405 preservados (P2)

**Goal**: 404 e 405 (com `Allow`) equivalentes ao comportamento anterior.
**Independent Test**: casos de 404/405 de T002 passam.

- [x] T006 [US2] Em `internal/adapter/http/handlers.go`, registrar `r.MethodNotAllowed(...)` que calcula `Allow` consultando o roteador (`chi.RouteContext` + `Mux.Match` para GET, HEAD, POST) na mesma ordem observada em T003, e responde 405 com o corpo de referência; manter o 404 padrão do chi (ajustar só se T002 divergir)
- [x] T007 [US2] Rodar `go test -race ./...` e corrigir divergências dos testes de 404/405/HEAD/`/docs/*` de T002 sem alterar suas expectativas

## Phase 5: Polish

- [x] T008 [P] `make lint` e `go vet ./...` em `notificacao/`; corrigir achados — `make lint` 0 issues (golangci-lint em `~/go/bin`) e `go vet` ok
- [x] T009 [P] Rodar `make test` completo; se Docker disponível, `make test-integration` (`test/integration/ambiente_test.go` usa `Rotas()`) — `make test` ok; `make test-integration` ok (140 s, execução real com `-count=1`)
- [x] T010 [P] Atualizar menção ao roteador em `README.md` e `erp-notificacao.md` se citarem `ServeMux`/`net/http` como roteador (checar com `grep -n "ServeMux\|roteador" README.md erp-notificacao.md`)
  (verificado: nenhuma menção a ServeMux/roteador)
- [x] T011 Validar `specs/003-roteamento-chi/quickstart.md` (passos automatizados; manuais se a plataforma estiver no ar)

## Dependências

T001 → T002 → T003 → T004 → T005 → T006 → T007 → Polish (T008–T010 em paralelo; T011 por último). US2 depende de US1 (mesmo arquivo `handlers.go`).

## Estratégia

MVP = US1 (T001–T005). US2 fecha as diferenças de borda do roteador. Commit direto na `master`, sem branch.
