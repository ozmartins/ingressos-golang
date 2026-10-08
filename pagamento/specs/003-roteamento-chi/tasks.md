# Tasks: Roteamento do Pagamento via chi

**Input**: `specs/003-roteamento-chi/` (spec.md, plan.md, research.md, data-model.md, quickstart.md)
**Tests**: a spec pede equivalência verificável (SC-001/SC-002); por isso há tarefas de teste de caracterização, escritas **antes** da troca para medir o `ServeMux` atual (Constituição III).
**Commits**: direto na `master`, sem branch.

Formato: `- [ ] ID [P?] [Story] descrição com caminho`. Todos os caminhos são relativos a `ingressos-golang/pagamento/`.

## Phase 1: Setup

- [x] T001 Rodar `make test` em `pagamento/` e registrar que a suíte atual passa (linha de base antes de qualquer alteração)
- [x] T002 Adicionar a dependência com `go get github.com/go-chi/chi/v5` e `go mod tidy` em `go.mod`/`go.sum`

## Phase 2: Foundational (caracterização do comportamento atual)

**Bloqueia a troca**: mede o que o `ServeMux` faz hoje, para que a troca seja comparada com fatos.

- [x] T003 Em `internal/adapter/http/handlers_test.go`, escrever teste de caracterização (passando contra o `ServeMux` atual) para caminho inexistente: status, `Content-Type` e corpo
- [x] T004 Em `internal/adapter/http/handlers_test.go`, escrever teste de caracterização para método não permitido (ex.: `DELETE` em `/api/v1/pagamentos/reserva/{id}`): status, cabeçalho `Allow`, corpo
- [x] T005 Em `internal/adapter/http/handlers_test.go`, escrever teste de caracterização para `HEAD` em rota `GET`, `GET /docs`, `GET /docs/` e `GET /docs/qualquer` (status e se a UI é servida), e para barra final divergente nas demais rotas (`/api/v1/health/live/`, `/api/v1/pagamentos/reserva/{id}/`)
- [x] T006 Executar os testes T003–T005 e anotar os valores reais em `specs/003-roteamento-chi/research.md` (seção "Caracterização medida"), confirmando ou corrigindo D2–D4

## Phase 3: User Story 1 — Comportamento HTTP idêntico (P1) 🎯 MVP

**Goal**: rotas, `reserva_id`, autenticação, saúde e documentação servidos pelo chi sem diferença observável.
**Independent Test**: `make test` — suíte existente + caracterização passam.

- [x] T007 [US1] Em `internal/adapter/http/handlers.go`, reescrever `API.Rotas()` com `chi.NewRouter()` registrando as mesmas rotas de `data-model.md` (sem `Use(...)`), e `/docs` e `/docs/` como rotas exatas
- [x] T008 [US1] Em `internal/adapter/http/handlers.go`, trocar `r.PathValue("reserva_id")` por `chi.URLParam(r, "reserva_id")` em `consultar` e `escolherForma`
- [x] T009 [US1] Em `internal/adapter/http/handlers_test.go`, ajustar testes que injetem `SetPathValue` direto num handler para rotear por `Rotas()`; não enfraquecer nenhuma asserção
- [x] T010 [US1] Rodar `make test` e confirmar que a suíte existente passa (FR-002, FR-003, FR-004, FR-006)

## Phase 4: User Story 2 — 404, 405 e HEAD (P2)

**Goal**: respostas para rota inexistente, método não permitido e `/docs/` iguais às medidas em T006.
**Independent Test**: testes de caracterização T003–T005 passam sem alteração.

- [x] T011 [US2] Em `internal/adapter/http/handlers.go`, definir `r.NotFound` e `r.MethodNotAllowed` reproduzindo status, corpo e `Allow` medidos em T006
- [x] T012 [US2] Em `internal/adapter/http/handlers.go`, se T006 mostrou que `HEAD`/`/docs/<subcaminho>` eram atendidos antes, preservar com registro explícito; se a preservação não for natural, parar e levar a diferença ao mantenedor (Constituição IV)
- [x] T013 [US2] Rodar T003–T005 e confirmar que passam inalterados (FR-005)

## Phase 5: User Story 3 — Contrato e fronteira do núcleo (P3)

**Goal**: contrato OpenAPI sem alteração e núcleo proibido de importar chi.
**Independent Test**: `make test` (paridade com contrato) e `make lint`.

- [x] T014 [P] [US3] Em `.golangci.yml`, acrescentar `github.com/go-chi/chi` à regra `nucleo-sem-adaptadores` com `desc: "roteador HTTP é detalhe de adaptador"`
- [x] T015 [P] [US3] Confirmar com `git diff --stat` que `specs/001-pagamento-assincrono/contracts/openapi.yaml` e `internal/adapter/http/openapi/` não foram alterados (FR-007), e que `cmd/pagamento/main.go` também não mudou (FR-009); executar depois de T012
- [x] T016 [US3] Rodar `make lint` e `grep -r go-chi internal/domain internal/usecase` (deve ficar vazio) (FR-008, SC-004)

## Phase 6: Polish

- [x] T017 Rodar `make build && make test` completos e, se Docker disponível, `make test-integration`
- [ ] T018 Executar os passos 4–5 de `specs/003-roteamento-chi/quickstart.md` (fluxo ponta a ponta e comparação de 404/405)
- [ ] T019 Commitar direto na `master` (Conventional Commits, em inglês) com `go.mod`, `go.sum`, `.golangci.yml`, código, testes e `specs/003-roteamento-chi/`

## Dependencies

Setup → Foundational (T003–T006) → US1 → US2 → US3 → Polish. US1 e US2 editam `handlers.go` e são sequenciais; US3 (T014, T015) pode correr em paralelo a partir do fim de T002.

## Implementation Strategy

MVP = Setup + Foundational + US1 (T001–T010): roteador trocado com a suíte verde. US2 fecha as diferenças sutis de 404/405/HEAD; US3 trava a fronteira. Total: 19 tarefas (US1: 4, US2: 3, US3: 3, demais: 9).
