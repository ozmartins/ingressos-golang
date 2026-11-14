# Tasks: Página de Terminal Retrô

**Input**: Design documents from `/specs/002-retro-terminal-page/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md
**Tests**: Incluídos para a lógica pura (constituição, princípio IV); a UI é verificada via `quickstart.md`.

**Format**: `- [ ] [ID] [P?] [Story?] Descrição com caminho`. Caminhos relativos a `ingressos-golang/web/`.

## Phase 1: Setup

- [X] T001 Instalar Vitest como dependência de desenvolvimento (`npm install -D vitest`) em `package.json`; confirmar que a instalação não gera conflito de peer com o Vite 8 e que a versão do Node atende `^22.12 || ^24 || >=26`
- [X] T002 Adicionar o script `"test": "vitest run"` em `package.json` e o bloco `test: { environment: 'node' }` em `vite.config.ts` (importando `defineConfig` de `vitest/config`); em seguida rodar `npm run build` para garantir que o `tsc -b` continua passando (o `vite.config.ts` é verificado por `tsconfig.node.json`) — se falhar, corrigir antes de seguir
- [X] T003 [P] Atualizar `index.html`: `<title>Terminal</title>` (mantendo `lang="pt-BR"`)

## Phase 2: Foundational

- [X] T004 Criar o tipo `Linha` (`tipo: 'sistema' | 'entrada' | 'saida' | 'erro'`, `texto: string`), a constante `MAX_LINHAS = 500` e a função pura `adicionarLinhas(atuais, novas)` que descarta as mais antigas além do limite, em `src/terminal/linhas.ts`
- [X] T005 [P] Criar testes de `adicionarLinhas` (acrescenta, respeita o limite de 500 descartando as mais antigas) em `src/terminal/linhas.test.ts`

## Phase 3: User Story 1 - Digitar um comando e ver o resultado (P1) 🎯 MVP

**Goal**: Campo embaixo, saída no restante da tela, comandos embutidos funcionando.
**Independent Test**: `npm run dev`; digitar `help` + Enter mostra o comando e a lista na saída; comando desconhecido mostra erro; saída rola ao fim.

- [X] T006 [P] [US1] Criar testes de `executar` para `help`, `echo` (incluindo `<b>` literal e vazio), `date`, `whoami`, `about`, comando desconhecido (`comando não encontrado: xyz`), entrada vazia e insensibilidade a maiúsculas/espaços extras em `src/terminal/comandos.test.ts`
- [X] T007 [US1] Implementar `executar(entrada, agora)` retornando `{ linhas, limpar }` com os comandos `help`, `echo`, `date`, `whoami`, `about` e o erro de comando desconhecido, conforme `data-model.md`, em `src/terminal/comandos.ts` (depende de T004)
- [X] T008 [US1] Criar o componente `Terminal` em `src/terminal/Terminal.tsx`: estado de linhas iniciado com boas-vindas sugerindo `help`; `<input>` controlado fixo na parte inferior com `autoFocus` e prompt `>`; Enter chama `executar`, acrescenta eco (`tipo: 'entrada'`) + resultado via `adicionarLinhas` e limpa o campo; renderizar linhas como texto puro (sem `dangerouslySetInnerHTML`); rolar a área de saída ao fim a cada nova linha; clique no contêiner devolve o foco ao campo (depende de T004, T007)
- [X] T009 [US1] Renderizar `<Terminal />` em `src/App.tsx` (substitui o `return null`)
- [X] T010 [US1] Criar `src/estilos.css` com o layout mínimo (altura 100vh, coluna flex: saída com `flex: 1` e `overflow-y: auto`, campo embaixo; quebra de linhas longas com `overflow-wrap: anywhere`; sem rolagem horizontal) e importá-lo em `src/main.tsx`
- [X] T011 [US1] Rodar `npm test` e `npm run build`; verificar manualmente os passos 2–6, 9 e 11 do `quickstart.md`

## Phase 4: User Story 2 - Aparência de monitor de fósforo (P2)

**Goal**: Visual de monitor de fósforo.
**Independent Test**: Passos 1 e 10 do `quickstart.md`.

- [X] T012 [US2] Estender `src/estilos.css` com o tema: fundo `#050805`, texto `#33ff66` monoespaçado (`ui-monospace, "Cascadia Mono", Consolas, "Courier New", monospace`), `text-shadow` de brilho, campo sem borda/contorno com `caret-color` transparente e cursor em bloco, cores distintas para `erro` e `entrada`
- [X] T013 [US2] Adicionar em `src/estilos.css` a sobreposição de linhas de varredura (`repeating-linear-gradient`, `pointer-events: none`, `position: fixed`) e o `@keyframes` do cursor piscante
- [X] T014 [US2] Adicionar em `src/estilos.css` o bloco `@media (prefers-reduced-motion: reduce)` que desativa o piscar do cursor e demais animações
- [X] T015 [US2] Verificar contraste ≥ 4,5:1 do texto sobre o fundo e a legibilidade com a varredura ativa; ajustar a opacidade da sobreposição se necessário (passos 1 e 10 do `quickstart.md`)

## Phase 5: User Story 3 - Histórico e comandos de utilidade (P3)

**Goal**: Navegar histórico com ↑/↓ e limpar a tela.
**Independent Test**: Passos 7 e 8 do `quickstart.md`.

- [X] T016 [P] [US3] Criar testes do histórico (adicionar ignora vazios; ↑ para no primeiro; ↓ após o último devolve vazio; novo comando reinicia o ponteiro) em `src/terminal/historico.test.ts`
- [X] T017 [P] [US3] Adicionar caso de teste do comando `clear` (retorna `limpar: true`) em `src/terminal/comandos.test.ts`
- [X] T018 [US3] Implementar as funções puras de histórico (`adicionar`, `anterior`, `proximo`) em `src/terminal/historico.ts` (depende de T016 para validar)
- [X] T019 [US3] Implementar o comando `clear` em `src/terminal/comandos.ts` e listá-lo no `help`
- [X] T020 [US3] Integrar em `src/terminal/Terminal.tsx`: ↑/↓ preenchem o campo via histórico (com `preventDefault`) e `limpar: true` esvazia a saída (depende de T018, T019)
- [X] T021 [US3] Rodar `npm test` e verificar manualmente os passos 7 e 8 do `quickstart.md`

## Phase 6: Polish

- [X] T022 Atualizar `README.md` com o comando `npm test` e a descrição do terminal
- [X] T023 Executar `npm test`, `npm run build` e percorrer toda a tabela de verificação manual do `quickstart.md`; confirmar console sem erros

## Dependencies & Execution Order

- Setup → Foundational → US1 → US2 e US3 (independentes entre si, ambos dependem de US1) → Polish.
- Dentro de US1: T006 (teste) antes/ao lado de T007; T008 depende de T004 e T007; T009 e T010 após T008.
- Paralelos: T003; T005; T006; T016/T017.
- T012–T014 e T010 editam o mesmo arquivo (`src/estilos.css`) e portanto são sequenciais.

## Implementation Strategy

MVP = Fases 1–3 (US1). Depois US2 (visual) e US3 (histórico/clear), por fim Polish. Total: 23 tarefas (Setup 3, Foundational 2, US1 6, US2 4, US3 6, Polish 2).
