# Tasks: Página inicial com abas de Filmes e Cinemas

**Input**: Design documents de `specs/001-home-filmes-cinemas/` (plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md)

**Tests**: incluídos, conforme a estratégia do plano (Vitest para lógica pura e componentes). Escreva cada teste antes da implementação correspondente e confirme que falha.

**Formato**: `- [ ] T### [P?] [US?] Descrição com caminho`. Todos os caminhos são relativos a `ingressos-golang/front/`. `[P]` = pode rodar em paralelo (arquivos distintos, sem dependência pendente).

## Phase 1: Setup

- [X] T001 Adicionar devDependencies `vitest`, `jsdom`, `@testing-library/react`, `@testing-library/user-event`, `@testing-library/jest-dom` e o script `"test": "vitest run"` em `package.json` (rodar `npm install`)
- [X] T002 Configurar em `vite.config.ts` o proxy `/catalogo` → `http://localhost:8082` (com `rewrite` removendo o prefixo, em `server` e `preview`, como em `../admin/vite.config.ts`) e o bloco `test` (`environment: 'jsdom'`, `setupFiles: ['./src/testes/setup.ts']`, importando `defineConfig` de `vitest/config`)
- [X] T003 [P] Criar `src/testes/setup.ts` importando `@testing-library/jest-dom/vitest` e limpeza automática entre testes
- [X] T004 [P] Declarar `VITE_CATALOGO_URL?: string` em `src/vite-env.d.ts` (criar o arquivo com `/// <reference types="vite/client" />` se não existir) e confirmar `lang="pt-BR"` e título "Cinema" em `index.html`

---

## Phase 2: Foundational (bloqueia todas as histórias)

- [X] T005 [P] Criar tipos `Filme`, `Cinema`, `StatusFilme`, `PaginaApi<T>` conforme `data-model.md` em `src/api/tipos.ts`
- [X] T006 [P] Teste de `normalizar()` (acentos, caixa, espaços nas pontas, string vazia) em `src/dominio/texto.test.ts`
- [X] T007 Implementar `normalizar(texto)` (NFD sem diacríticos, `toLowerCase`, `trim`) e `ordenarPtBr(lista)` em `src/dominio/texto.ts`
- [X] T008 [P] Teste do cliente do catálogo em `src/api/catalogo.test.ts`: percorre páginas enquanto `tem_proxima`, envia `page_size=100`, `ativo=true` nos cinemas, lança erro em status ≠ 200/rede/JSON inválido (usar `fetch` simulado)
- [X] T009 Implementar `buscarFilmes()` e `buscarCinemas()` em `src/api/catalogo.ts` (base `import.meta.env.VITE_CATALOGO_URL ?? '/catalogo'`, caminho `/api/v1/...`, concatena páginas; `buscarFilmes` descarta `FORA_DE_CARTAZ`)
- [X] T010 [P] Teste do hook em `src/hooks/useLista.test.tsx`: estados `carregando → pronto`, `erro` e `recarregar()`
- [X] T011 Implementar `useLista(carregar)` retornando `{ estado, itens, erro, recarregar }` em `src/hooks/useLista.ts`
- [X] T012 [P] Criar `EstadoLista` (carregando, vazio, sem resultado com botão "Limpar filtros", erro com botão "Tentar novamente") em `src/componentes/EstadoLista.tsx`, com teste em `src/componentes/EstadoLista.test.tsx`
- [X] T013 [P] Criar estilos base (variáveis CSS, tipografia, grade responsiva de cartões, foco visível, ≥ 360 px sem rolagem horizontal) em `src/index.css`

**Checkpoint**: base pronta; as histórias podem começar.

---

## Phase 3: User Story 1 - Explorar e filtrar filmes (P1) 🎯 MVP

**Goal**: página inicial abre com lista de filmes e filtros de título, gênero, classificação e situação.

**Independent Test**: abrir a página, ver filmes sem interação, aplicar cada filtro e a combinação, limpar filtros (cenários 1–3 do quickstart).

### Testes (US1)

- [X] T014 [P] [US1] Teste de `filtrarFilmes` e `opcoesFilmes` (texto sem acento/caixa, gênero, classificação, situação, combinação E, opções ordenadas e derivadas dos dados) em `src/dominio/filtrosFilmes.test.ts`
- [X] T015 [P] [US1] Teste de `AbaFilmes` (renderiza cartões, filtra ao digitar, contagem "N filmes", "Limpar filtros", sem resultado, erro com nova tentativa) em `src/componentes/AbaFilmes.test.tsx`

### Implementação (US1)

- [X] T016 [US1] Implementar `FiltrosFilmes`, `filtrosFilmesVazios`, `filtrarFilmes` e `opcoesFilmes` em `src/dominio/filtrosFilmes.ts`
- [X] T017 [P] [US1] Criar `CartaoFilme` (título, gênero, duração "2h 05min", classificação, situação, imagem com substituto neutro quando ausente; omite sinopse ausente) em `src/componentes/CartaoFilme.tsx`
- [X] T018 [US1] Criar `AbaFilmes` (campo de busca, selects de gênero/classificação/situação com `<label>`, botão limpar, contagem em `aria-live="polite"`, usa `useLista(buscarFilmes)` e `EstadoLista`) em `src/componentes/AbaFilmes.tsx`
- [X] T019 [US1] Criar `PaginaInicial` exibindo a `AbaFilmes` sob um cabeçalho e um `Abas` mínimo (tablist com a aba "Filmes" ativa) em `src/componentes/PaginaInicial.tsx` e `src/componentes/Abas.tsx`; trocar o corpo de `src/App.tsx` por `<PaginaInicial />`

**Checkpoint**: US1 funcional e testável sozinha (MVP).

---

## Phase 4: User Story 2 - Explorar e filtrar cinemas (P2)

**Goal**: segunda aba lista cinemas ativos com filtros de nome, estado e cidade (dependente do estado).

**Independent Test**: selecionar "Cinemas", filtrar por nome, estado e cidade (cenário 4 do quickstart).

### Testes (US2)

- [X] T020 [P] [US2] Teste de `filtrarCinemas` e `opcoesCinemas` (nome sem acento/caixa, estado, cidade, cidades limitadas ao estado, cidade zerada ao trocar de estado) em `src/dominio/filtrosCinemas.test.ts`
- [X] T021 [P] [US2] Teste de `AbaCinemas` (cartões, filtros, contagem, limpar, sem resultado) em `src/componentes/AbaCinemas.test.tsx`

### Implementação (US2)

- [X] T022 [US2] Implementar `FiltrosCinemas`, `filtrosCinemasVazios`, `filtrarCinemas`, `opcoesCinemas` e `ajustarCidade` em `src/dominio/filtrosCinemas.ts`
- [X] T023 [P] [US2] Criar `CartaoCinema` (nome, cidade/UF, endereço) em `src/componentes/CartaoCinema.tsx`
- [X] T024 [US2] Criar `AbaCinemas` (busca, selects de estado e cidade, limpar, contagem, `useLista(buscarCinemas)`, `EstadoLista`) em `src/componentes/AbaCinemas.tsx`
- [X] T025 [US2] Estender `Abas` e `PaginaInicial` com a aba "Cinemas" (ordem Filmes, Cinemas; padrão Filmes) em `src/componentes/Abas.tsx` e `src/componentes/PaginaInicial.tsx`

**Checkpoint**: US1 e US2 funcionam de forma independente.

---

## Phase 5: User Story 3 - Alternar abas preservando contexto (P3)

**Goal**: filtros preservados ao trocar de aba, aba refletida na URL, navegação por teclado.

**Independent Test**: filtrar filmes → cinemas → voltar mantém filtros; recarregar em `?aba=cinemas` abre em Cinemas; teclado alterna abas (cenários 6 e 8 do quickstart).

### Testes (US3)

- [X] T026 [P] [US3] Teste de `useAbaNaUrl` (lê `?aba=`, valor inválido/ausente → `filmes`, escreve na URL, reage a `popstate`) em `src/hooks/useAbaNaUrl.test.tsx`
- [X] T027 [P] [US3] Teste de `PaginaInicial` (filtros de filmes mantidos após ir a cinemas e voltar; filtrar cinemas não altera filmes; setas/Home/End movem a seleção; `aria-selected` correto) em `src/componentes/PaginaInicial.test.tsx`

### Implementação (US3)

- [X] T028 [US3] Implementar `useAbaNaUrl()` com `history.pushState` e `popstate` em `src/hooks/useAbaNaUrl.ts`
- [X] T029 [US3] Manter as duas abas montadas na `PaginaInicial` (painel inativo com `hidden`) para preservar estado e dados, usando `useAbaNaUrl`, em `src/componentes/PaginaInicial.tsx`
- [X] T030 [US3] Completar `Abas` com padrão WAI-ARIA (`role=tablist/tab/tabpanel`, `aria-controls`, `tabIndex` móvel, setas, Home/End) em `src/componentes/Abas.tsx`

**Checkpoint**: todas as histórias entregues.

---

## Phase 6: Polish & Cross-Cutting

- [X] T031 [P] Revisar responsividade em 360 px e foco/contraste em `src/index.css`
- [X] T032 [P] Atualizar `README.md` do `front` (como rodar, `VITE_CATALOGO_URL`, proxy do catálogo, testes) em português
- [X] T033 Rodar `npm test`, `npm run lint` e `npm run build` e corrigir falhas
- [ ] T034 Executar manualmente os cenários 1–9 de `specs/001-home-filmes-cinemas/quickstart.md` contra o catálogo real e registrar divergências

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 → histórias. Phase 2 bloqueia tudo.
- US1 (P1) não depende de outras histórias. US2 reutiliza a base e estende `Abas`/`PaginaInicial` criados em US1 (T025 depende de T019). US3 depende de US1 e US2 (T029/T030 alteram `PaginaInicial`/`Abas`).
- Dentro de cada história: testes → lógica pura → componentes → integração em `PaginaInicial`.
- Polish depois das histórias desejadas.

### Parallel opportunities

- Setup: T003 e T004 em paralelo após T001/T002.
- Foundational: T005, T006, T008, T010, T012, T013 em paralelo (arquivos distintos).
- US1: T014, T015, T017 em paralelo. US2: T020, T021, T023 em paralelo; US1 e US2 podem avançar em paralelo até T025. US3: T026, T027 em paralelo.

```text
# Exemplo (US1): testes e cartão juntos
T014 filtrosFilmes.test.ts | T015 AbaFilmes.test.tsx | T017 CartaoFilme.tsx
```

## Implementation Strategy

1. **MVP**: Phases 1–3 (US1). Validar com o cenário 1–3 do quickstart.
2. Incremento 2: US2 (cinemas). Incremento 3: US3 (contexto, URL, teclado).
3. Polish e validação manual completa (T031–T034).
