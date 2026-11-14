# Tasks: Aplicação React Vazia

**Input**: Design documents from `/specs/001-empty-react-app/`
**Prerequisites**: plan.md, spec.md, research.md, quickstart.md (sem contracts/ nem data-model aplicável)
**Tests**: Não solicitados; verificação via `tsc` + `vite build` e `quickstart.md`.

**Format**: `- [ ] [ID] [P?] [Story?] Descrição com caminho`. Todos os caminhos são relativos a `ingressos-golang/web/`.

## Phase 1: Setup

- [X] T001 Criar `package.json` (name `web`, private, type `module`, scripts `dev`=`vite`, `build`=`tsc -b && vite build`, `preview`=`vite preview`) em `package.json`
- [X] T002 Instalar dependências com npm: `react`, `react-dom` e, como dev, `vite`, `@vitejs/plugin-react`, `typescript`, `@types/react`, `@types/react-dom` — gera `package-lock.json` e `node_modules/` (já ignorado em `.gitignore`)

## Phase 2: Foundational

- [X] T003 [P] Criar `tsconfig.json` (strict, jsx `react-jsx`, target ES2022, moduleResolution `bundler`, noEmit, include `src`) em `tsconfig.json`
- [X] T004 [P] Criar `tsconfig.node.json` com `composite: true`, `emitDeclarationOnly: true`, `include: ["vite.config.ts"]`, e referenciá-lo em `tsconfig.json` (`references`), necessário para o `tsc -b` do script `build`
- [X] T005 [P] Criar `vite.config.ts` com o plugin `@vitejs/plugin-react` em `vite.config.ts`

## Phase 3: User Story 1 - Iniciar a aplicação em modo de desenvolvimento (P1) 🎯 MVP

**Goal**: App React vazia que instala e roda localmente.
**Independent Test**: `npm install && npm run dev`, abrir `http://localhost:5173`: página vazia, console sem erros.

- [X] T006 [P] [US1] Criar `index.html` com `<div id="root"></div>`, título `Ingressos` e script `/src/main.tsx` em `index.html`
- [X] T007 [P] [US1] Criar `src/vite-env.d.ts` com `/// <reference types="vite/client" />`
- [X] T008 [P] [US1] Criar componente raiz vazio (`export default function App() { return null }`) em `src/App.tsx`
- [X] T009 [US1] Criar `src/main.tsx` montando `<App />` em `#root` com `StrictMode` (depende de T008)
- [X] T010 [US1] Executar `npm run dev` e confirmar que a página carrega sem erros no console (cenário 2 de `quickstart.md`)

## Phase 4: User Story 2 - Gerar versão de produção (P2)

**Goal**: Build de produção sem erros.
**Independent Test**: `npm run build` retorna código 0 e gera `dist/`.

- [X] T011 [US2] Executar `npm run build` e corrigir eventuais erros de tipo/configuração em `tsconfig.json`/`vite.config.ts`
- [X] T012 [US2] Executar `npm run preview` e confirmar que `dist/` serve a página vazia (cenário 4 de `quickstart.md`)

## Phase 5: Polish

- [X] T014 Criar `README.md` curto em `README.md` documentando os comandos `npm install`, `npm run dev`, `npm run build` e `npm run preview` (atende FR-002)
- [X] T013 Confirmar via `git status` que `.claude/`, `.specify/`, `.gitignore`, `.dockerignore` permanecem intactos e que `node_modules/` e `dist/` estão ignorados (cenário 5 de `quickstart.md`)

## Dependencies & Execution Order

- Setup (T001→T002) → Foundational (T003–T005 em paralelo) → US1 → US2 → Polish.
- US2 depende de US1 (precisa do código-fonte da app para construir).
- Paralelos: T003/T004/T005; T006/T007/T008.

## Implementation Strategy

MVP = Phase 1–3 (US1). Depois validar build (US2) e fechar com Polish. Total: 14 tarefas (US1: 5, US2: 2, Setup 2, Foundational 3, Polish 2).
