# Tasks: Roteamento do Estoque via chi

**Input**: [spec.md](./spec.md), [plan.md](./plan.md), [research.md](./research.md) (decisões D1–D10), [quickstart.md](./quickstart.md)

**Tests**: obrigatórios (constituição, princípio VIII — interface exposta). Abordagem: caracterização primeiro — a tabela de comportamento é escrita e passa contra o `ServeMux` atual **antes** da troca, e depois deve passar com o chi sem alterar asserções (exceto as linhas D7, ver T014).

**Trabalho direto na `master`**, sem branch de feature. Todos os caminhos são relativos a `ingressos-golang/estoque/`.

## Phase 1: Setup

- [X] T001 Adicionar `github.com/go-chi/chi/v5 v5.3.2` a `go.mod`/`go.sum` (offline: `GOFLAGS=-mod=mod GOPROXY=off go get github.com/go-chi/chi/v5@v5.3.2`; a versão está no cache local)

## Phase 2: Foundational (testes de caracterização, rodando ainda sobre o ServeMux)

**⚠️ Bloqueia as fases seguintes**: sem a rede de segurança verde antes da troca, não há como provar equivalência.

- [X] T002 [P] Escrever `internal/adapter/http/roteador_test.go`: tabela (método, caminho) → (status, `Allow`, `Content-Type`, corpo) via `API.Rotas()` com fakes de `handlers_test.go`, cobrindo as linhas da tabela D2: 405 em rota só-`POST`/só-`GET` (`Allow: POST` / `GET, HEAD`, corpo `Method Not Allowed\n`), `HEAD` em rota `GET` → 200, `OPTIONS` → 405, `GET /nada` → 404 (`404 page not found\n`), `/docs`, `/docs/`, `/docs/x` → 200, `sessao_id` com `%2F` chegando decodificado (`x/y`) ao caso de uso
- [X] T003 [P] Escrever `internal/platform/health/health_test.go`: `/health/live` → 200 `{"status":"vivo"}`; `/health/ready` → 200 `pronto`, 503 com dependência essencial falhando, 200 `degradado` com não essencial falhando; 404 e 405 (+`Allow`, `HEAD`) como na tabela D2
- [X] T004 Rodar `go test -race ./internal/adapter/http/... ./internal/platform/health/...` e confirmar que T002 e T003 passam **contra o ServeMux atual** (qualquer falha aqui é erro do teste, não do código); registrar o resultado

**Checkpoint**: comportamento atual congelado em teste.

## Phase 3: User Story 1 — Comportamento HTTP idêntico (Priority: P1) 🎯 MVP

**Goal**: API REST e saúde roteadas pelo chi, com os caminhos felizes idênticos.

**Independent Test**: suíte de `internal/adapter/http` (incluindo `handlers_test.go` inalterado) e `health_test.go` verdes sobre o chi nos casos de sucesso/401/parâmetros.

- [X] T005 [US1] Criar `internal/adapter/http/roteador.go` com `NovoRoteador() *chi.Mux` (por ora `chi.NewRouter()` simples; as compensações entram na US2) — D9
- [X] T006 [US1] Migrar `API.Rotas()` em `internal/adapter/http/handlers.go` para `NovoRoteador()` com `Post(".../bloqueios")`, `Get(".../poltronas")`, `Get("/openapi.yaml")`, `Get("/docs")`; manter `r.PathValue("sessao_id")` (chi v5.3.2 preenche — D1); retornar `http.Handler`
- [X] T007 [US1] Adicionar helper privado `sessaoID(r *http.Request) string` em `internal/adapter/http/handlers.go` (`url.PathUnescape` de `r.PathValue("sessao_id")`; em erro devolve o valor cru — D6) e usá-lo em `bloquear` e `consultarMapa`
- [X] T008 [US1] Migrar `Servico.Handler()` em `internal/platform/health/health.go` para `adaptadorhttp.NovoRoteador()` (`Get("/health/live")`, `Get("/health/ready")`), sem alterar os handlers
- [X] T009 [US1] Rodar `go test -race ./internal/adapter/http/... ./internal/platform/health/...`: `handlers_test.go` inalterado e os casos de sucesso/401/parâmetro codificado de T002/T003 verdes (as linhas de 404/405/HEAD//docs/x podem falhar até a US2)

**Checkpoint**: caminho feliz equivalente; MVP da troca.

## Phase 4: User Story 2 — 404, 405, HEAD e subcaminhos (Priority: P2)

**Goal**: respostas para caminhos inexistentes e métodos não permitidos idênticas às de hoje.

**Independent Test**: tabela de caracterização (T002/T003) 100% verde sobre o chi, salvo as linhas D7.

- [X] T010 [US2] Em `NovoRoteador()` (`internal/adapter/http/roteador.go`) aplicar `middleware.GetHead` do chi (D4)
- [X] T011 [US2] Em `NovoRoteador()` registrar `MethodNotAllowed` com handler que responde 405, `Content-Type: text/plain; charset=utf-8`, corpo `Method Not Allowed\n` e `Allow` calculado sondando o próprio roteador com `Match` para `GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS` em ordem alfabética, incluindo `HEAD` quando `GET` casa (D3)
- [X] T012 [US2] Em `Rotas()` (`handlers.go`) trocar o registro de documentação para `Get("/docs")` + `Get("/docs/*")` (D5)
- [X] T013 [US2] Rodar a tabela de T002/T003: todas as linhas verdes, exceto as da D7
- [X] T014 [US2] Adicionar em `roteador_test.go` um teste explícito que **documenta** a diferença D7 (`GET /a//x/c` e caminhos com `..` → 404 em vez de 307), com comentário apontando para `research.md` D7; diferença aceita pelo mantenedor em 2026-10-07 (FR-005, princípio X) — **não** implementar redirecionamento

**Checkpoint**: nenhuma diferença observável além da D7.

## Phase 5: User Story 3 — Fronteira do núcleo e contrato (Priority: P3)

**Goal**: o chi não vaza para domínio/casos de uso e o contrato segue intocado.

**Independent Test**: `make test` e `make lint` verdes.

- [X] T015 [P] [US3] Adicionar `"github.com/go-chi"` à lista `proibidos` em `test/arquitetura_test.go`
- [X] T016 [P] [US3] Adicionar `github.com/go-chi` com `desc` ao `deny` da regra `nucleo-nao-importa-adaptador` em `.golangci.yml`
- [X] T017 [US3] Rodar `make test` e `make lint` (lint rodou com `~/go/bin/golangci-lint` 2.13.2: 60 achados, idênticos aos do HEAD sem a mudança — nenhum novo, depguard sem achados) e confirmar: verdes; `internal/adapter/http/openapi/sincronia_test.go` passa sem alterar `specs/001-estoque-bloqueio-poltronas/contracts/openapi.yaml`; `go list -deps ./internal/domain/... ./internal/usecase/... | grep go-chi` vazio

## Phase 6: Polish

- [X] T018 Rodar `make test-integration` (inclui `test/integration/largada_test.go` sobre `/health/ready`) e registrar o resultado (rodou: ok, 23,8 s); se Docker indisponível, dizer explicitamente que não rodou
- [X] T019 [P] Procurar menções ao `ServeMux`/roteador padrão em `README.md` e `specs/*/` do estoque (`grep -rn "ServeMux" .`) e **reportar** ao mantenedor as que ficaram desatualizadas, sem editar specs anteriores (princípio X)
- [X] T020 Executar o roteiro de `quickstart.md` (passos 1–6; o 5 é opcional) e confirmar `go mod tidy` sem diff residual

## Dependencies & Execution Order

- Setup (T001) → Foundational (T002‖T003 → T004) → US1 (T005 → T006/T007 → T008 → T009) → US2 (T010, T011, T012 → T013 → T014) → US3 (T015‖T016 → T017) → Polish.
- T006, T007 e T012 editam `handlers.go` e T005/T010/T011 editam `roteador.go`: sequenciais dentro de cada arquivo.
- US3 é independente de US2 em código (T015/T016 podem ir logo após T001), mas T017 só fecha depois de tudo.

## Parallel Opportunities

- T002 ‖ T003 (arquivos distintos); T015 ‖ T016; T019 ‖ T018.

## Implementation Strategy

1. **MVP = Foundational + US1**: rede de segurança + troca do caminho feliz.
2. US2 fecha as divergências medidas; US3 tranca a fronteira do núcleo.
3. Commitar direto na `master` ao final (push só se pedido); um commit coeso, ou um por fase se o mantenedor preferir.
