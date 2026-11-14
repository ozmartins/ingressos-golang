# Tasks: Ajuda do Terminal por Controller

**Input**: Design documents from `/specs/005-help-por-controller/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ajuda.md, quickstart.md

**Tests**: incluídos (constituição, princípio IV); escritos antes da implementação.

Todos os caminhos são relativos a `ingressos-golang/web/`.

## Format: `[ID] [P?] [Story] Description`

---

## Phase 1: Setup

- [X] T001 Confirmar a linha de base: `npm test` e `npm run build` passando antes de qualquer alteração

---

## Phase 2: Foundational

- [X] T002 [P] Teste em `src/terminal/api/tabela.test.ts`: `nomesDeComandos()` devolve `['filmes','cinemas','salas','sessoes','health']`
- [X] T003 Em `src/terminal/api/tabela.ts`, exportar `nomesDeComandos()` (nomes de `CONTROLLERS_CRUD` + `health`), até T002 passar

---

## Phase 3: User Story 1 — Ver só os controllers no help geral (P1) 🎯 MVP

**Goal**: `help` mostra só os nomes dos controllers na seção da API.

**Independent Test**: `help` mostra os 5 nomes e a dica, sem ações/descrições, em até 20 linhas.

- [X] T004 [P] [US1] Testes em `src/terminal/api/formatar.test.ts`: `formatarAjuda()` lista os nomes de `nomesDeComandos()` na seção "Comandos da API do catálogo", não contém `filmes listar` nem descrições, contém a dica `filmes help`, mantém `login`/`logout`, e o total do help geral (`despachar('help')`) tem no máximo 20 linhas
- [X] T005 [P] [US1] Atualizar em `src/terminal/despachar.test.ts` o teste "help junta os comandos locais e os de API" para a nova saída (`help`, `clear`, nomes dos controllers, `health`, `login`, `logout`; sem `sessoes remover <id>`), e adicionar `help filmes` ⇒ mesmo help geral resumido, sem erro (FR-009)
- [X] T006 [US1] Em `src/terminal/api/formatar.ts`, reescrever `formatarAjuda()` para listar os nomes (recuados) + dica e remover da seção a nota "criar e atualizar perguntam os campos…" (vai para a ajuda do controller)

**Checkpoint**: US1 funcional; o help detalhado ainda não está acessível (vem na US2).

---

## Phase 4: User Story 2 — Ver as opções de um controller (P2)

**Goal**: `<controller> help` e `health help` mostram as opções, sem requisição.

**Independent Test**: `salas help` lista as 5 ações de salas e nenhuma de outro controller.

- [X] T007 [P] [US2] Testes em `src/terminal/api/formatar.test.ts`: `formatarAjudaController` para cada controller lista exatamente as ações de `definicoesDe(controller)` (sintaxe + descrição), não menciona outro controller, e inclui a nota de `cancelar`/Esc; para `operacional` mostra `health`
- [X] T008 [P] [US2] Testes em `src/terminal/despachar.test.ts`: `filmes help`, `cinemas help`, `salas help`, `sessoes help`, `health help`, `FILMES   help` e `filmes help extra` retornam a ajuda correta e o `fetch` falso não recebe chamadas (nem sem token) (FR-006, FR-007)
- [X] T009 [US2] Em `src/terminal/api/formatar.ts`, criar `formatarAjudaController(controller)` com o formato de [contracts/ajuda.md](contracts/ajuda.md)
- [X] T010 [US2] Em `src/terminal/despachar.ts`, no ramo de controllers tratar a ação `help` (case-insensitive) com `formatarAjudaController`; no ramo de `health`, tratar `resto` igual a `help` da mesma forma, sem chamar `executarDefinicao`

**Checkpoint**: US1 e US2 funcionam.

---

## Phase 5: User Story 3 — Orientação em entradas parecidas (P3)

**Goal**: controller sem ação/ação desconhecida apontam para `<controller> help`.

**Independent Test**: `filmes` e `filmes xpto` incluem a linha `use: filmes help para ver os detalhes.`.

- [X] T011 [P] [US3] Atualizar testes em `src/terminal/api/formatar.test.ts` (`formatarAcoes`) e `src/terminal/despachar.test.ts` ("controller sem ação…") para incluir a linha de dica
- [X] T012 [US3] Em `src/terminal/api/formatar.ts`, fazer `formatarAcoes` acrescentar `use: <controller> help para ver os detalhes.`

---

## Phase 6: Polish

- [X] T013 [P] Atualizar `README.md` de `ingressos-golang/web/` (menção ao `help` e a `<controller> help`)
- [X] T014 Rodar `npm test` e `npm run build` (saída 0, sem avisos)
- [X] T015 Executar o roteiro de [quickstart.md](quickstart.md) na página e anotar o resultado

---

## Dependencies & Execution Order

- T001 → Foundational (T002→T003) → US1 → US2 → US3 → Polish.
- US2 depende de T009 antes de T010; US1 e US2 editam `formatar.ts`: fazer T006 antes de T009.
- Testes de cada história precedem a implementação.

## Parallel Example

```text
US1: T004 e T005 em paralelo (arquivos de teste distintos)
US2: T007 e T008 em paralelo
```

## Implementation Strategy

- **MVP**: Foundational + US1; mas o help detalhado só fica acessível após US2 — entregar US1 e US2 juntas.
- Depois US3 e Polish. Commits em Conventional Commits, direto na `master`.
