# Tasks: Ícone para retornar à página inicial em todas as páginas

**Input**: Design documents de `specs/007-icone-pagina-inicial/` (plan.md, spec.md, research.md, data-model.md, contracts/link-inicio.md, quickstart.md)

**Tests**: incluídos (o plan.md define a estratégia de testes com Vitest + Testing Library).

**Formato**: `- [ ] T### [P?] [US?] Descrição com caminho` — `[P]` = pode rodar em paralelo (arquivos distintos).

## Phase 1: Setup

Nenhum: sem dependências novas nem estrutura nova (projeto Vite/React já configurado).

## Phase 2: Foundational

- [x] T001 Ler `src/hooks/useRota.ts` (`navegar`, `interceptarClique`), `src/componentes/BarraConta.tsx` e `.barra-conta` em `src/index.css` para reaproveitar o padrão de link interno e as variáveis de tema (sem alterações de código)

**Checkpoint**: padrões confirmados; implementação das histórias pode começar.

## Phase 3: User Story 1 — Voltar à página inicial com um clique, de qualquer página (P1) 🎯 MVP

**Goal**: ícone de casa no mesmo lugar em todas as páginas, que leva à página inicial.

**Independent Test**: abrir cada rota (inicial, cinema, filme, compra, pagamento, erro/não encontrado), acionar o ícone e ver a página inicial; "voltar" do navegador retorna à origem; Ctrl+clique abre nova aba.

### Tests (US1)

- [x] T002 [P] [US1] Criar `src/componentes/LinkInicio.test.tsx`: link com nome "Página inicial" e `href="/"`; clique simples chama navegação interna para `/` (sem recarregar) e preserva `history`; Ctrl/Cmd+clique não é interceptado; em `pathname === '/'` o clique não cria nova entrada no histórico; em caminho desconhecido (ex.: `/xyz`) o clique leva a `/`
- [x] T003 [P] [US1] Estender `src/App.test.tsx`: o link "Página inicial" está presente em `/`, `/cinemas/{id}`, `/filmes/{id}`, `/cinemas/{id}/filmes/{id}/comprar`, `/cinemas/{id}/filmes/{id}/pagamento` e em estado de erro/não encontrado; clicar a partir de um cinema exibe a página inicial com seu estado (aba/filtros) preservado; Login/Criar Conta/Log out continuam presentes

### Implementation (US1)

- [x] T004 [US1] Criar `src/componentes/LinkInicio.tsx`: `<a className="link-inicio" href="/" aria-label="Página inicial" title="Página inicial">` com SVG de casa inline (`aria-hidden`, `currentColor`); `onClick` usa `interceptarClique` e `navegar('/')`, ou apenas `preventDefault` se `window.location.pathname === '/'`
- [x] T005 [US1] Criar `src/componentes/CabecalhoSite.tsx`: `<header className="cabecalho-site">` com `LinkInicio` à esquerda e `BarraConta` (mesmas props) à direita
- [x] T006 [US1] Em `src/App.tsx`, trocar `<BarraConta sessao={sessao} onSair={sair} />` por `<CabecalhoSite sessao={sessao} onSair={sair} />` (acima de todas as rotas)
- [x] T007 [US1] Em `src/index.css`, adicionar `.cabecalho-site` (flex, `justify-content: space-between`, `align-items: center`, `flex-wrap: wrap`, o padding hoje em `.barra-conta`) e ajustar `.barra-conta` (sem padding próprio, `margin-left: auto`) para manter a aparência atual dos botões; estilo básico de `.link-inicio` (ícone 24px, cor `--texto`)
- [x] T008 [US1] Rodar `npm test` e `npm run lint` em `ingressos-golang/front` e corrigir regressões (incl. `src/componentes/BarraConta.test.tsx`)

**Checkpoint**: MVP entregue — ícone funcional em todas as páginas.

## Phase 4: User Story 2 — Reconhecer e usar o ícone com facilidade e acessibilidade (P2)

**Goal**: alvo de toque, foco visível, dica e contraste nos dois temas, sem sobrepor a barra de conta.

**Independent Test**: Tab até o ícone mostra foco; leitor de tela anuncia "Página inicial, link"; 360 px de largura e temas claro/escuro legíveis.

### Tests (US2)

- [x] T009 [P] [US2] Em `src/componentes/LinkInicio.test.tsx`, cobrir: alcançável por Tab, Enter navega, `title="Página inicial"` presente, SVG com `aria-hidden="true"` (nome acessível vem só do `aria-label`)

### Implementation (US2)

- [x] T010 [US2] Em `src/index.css`, completar `.link-inicio`: `display: inline-flex`, centralizado, `min-width: 44px; min-height: 44px`, `border-radius: var(--raio)`, `:hover` com `--destaque`, `:focus-visible` com `outline: 3px solid var(--foco)`; usar apenas variáveis existentes (temas claro/escuro)
- [ ] T011 [US2] Validar manualmente em 360 px e nos dois temas (itens 5, 7 e 8 de `specs/007-icone-pagina-inicial/quickstart.md`) e ajustar o leiaute para não sobrepor Login/Criar Conta

**Checkpoint**: US1 + US2 completas.

## Phase 5: Polish & Cross-Cutting

- [ ] T012 Rodar `npm run build` em `ingressos-golang/front` e executar a validação manual completa de `specs/007-icone-pagina-inicial/quickstart.md`
- [x] T013 [P] Se `README.md` do front descrever a navegação/cabeçalho, adicionar uma linha sobre o ícone de início em `README.md`

## Dependências e ordem de execução

- T001 → (T002, T003 em paralelo) → T004 → T005 → T006 → T007 → T008
- US2 (T009–T011) depende de T004/T007 (o componente e o CSS base existirem)
- Polish (T012–T013) depois de US1 e US2

### Paralelismo

- T002 e T003 (arquivos de teste distintos)
- T009 em paralelo com T010 (teste × CSS)
- T013 em paralelo com T012

## Estratégia de implementação

1. **MVP**: Phases 2–3 (T001–T008) — ícone presente e funcional em todas as páginas.
2. **Incremental**: Phase 4 refina acessibilidade/leiaute; Phase 5 valida o fluxo completo.
3. Commits direto na `master`, mensagens em inglês no padrão Conventional Commits.
