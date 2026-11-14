# Tasks: Cinemas com o filme em cartaz

**Input**: Design documents from `specs/005-cinemas-do-filme/` (plan.md, spec.md, research.md, data-model.md, contracts/rota-filme.md, quickstart.md)

**Tests**: incluídos — o plano define Vitest + Testing Library colocalizados como convenção do front (`src/**/*.test.ts(x)`).

**Formato**: `- [ ] T### [P?] [US?] Descrição com caminho`. Todos os caminhos relativos a `ingressos-golang/front/`.

## Phase 1: Setup

- [X] T001 Confirmar base verde antes de alterar: rodar `npm test`, `npm run lint` e `npm run build` em `ingressos-golang/front/` e anotar falhas pré-existentes

## Phase 2: Foundational (bloqueia todas as histórias)

- [X] T002 [P] Estender `src/dominio/rotas.ts`: variante `{ tipo: 'filme'; id: string }` em `Rota`, `caminhoFilme(id)` (`/filmes/{id}` com `encodeURIComponent`) e ramo em `lerRota` (2 partes, `partes[0] === 'filmes'`; falha de decodificação → `inicio`); cobrir em `src/dominio/rotas.test.ts`
- [X] T003 [P] Em `src/api/catalogo.ts` adicionar `FilmeNaoEncontrado`, `buscarFilme(id)` (`GET /api/v1/filmes/{id}`; 404/400 → `FilmeNaoEncontrado`; valida `id` string) e `buscarSessoesDoFilme(filmeId)` (`buscarTodas<SessaoNaGrade>('/api/v1/sessoes', { filme_id })`); testes em `src/api/catalogo.test.ts`
- [X] T004 [P] Criar `src/dominio/cinemasDoFilme.ts` com `cinemasComOFilme(sessoes, cinemas, agora = new Date())` conforme `data-model.md` (descarta sessões passadas, agrupa por `cinema_id`, mantém só cinemas da lista ativa, ordena UF/cidade/nome via `ordenarPtBr`/`localeCompare('pt-BR')`) e testes em `src/dominio/cinemasDoFilme.test.ts` (sessão futura, só passadas, cinema inativo/ausente, vários cinemas, ordenação)

**Checkpoint**: rota, API e regra de domínio prontas.

## Phase 3: User Story 1 — Abrir um filme e ver os cinemas (P1) 🎯 MVP

**Goal**: card de filme leva a `/filmes/{id}`, que lista os cinemas com o filme em cartaz.

**Independent Test**: na aba Filmes clicar num card; a página mostra o filme e exatamente os cinemas com sessão futura dele (quickstart passos 1, 6; spec US1).

- [X] T005 [P] [US1] Tornar `src/componentes/CartaoFilme.tsx` clicável no padrão de `CartaoCinema` (classe `cartao-clicavel`, `<a className="cartao-link" href={caminhoFilme(id)}>` no título com `interceptarClique` + `navegar`); criar `src/componentes/CartaoFilme.test.tsx` (link correto, clique navega sem recarregar, Ctrl+clique não intercepta) e ajustar `src/componentes/AbaFilmes.test.tsx` se quebrar
- [X] T006 [P] [US1] Criar `src/componentes/CartaoCinemaDoFilme.tsx`: logo/marcador, nome, `cidade / UF`, endereço (sem botão ainda), reusando classes `cartao`/`cartao-cinema`
- [X] T007 [US1] Criar `src/componentes/PaginaFilme.tsx` (props `id`, `carregadores?` injetáveis `carregarFilme`, `carregarSessoes`, `carregarCinemas`): `useRecurso` para filme e para `Promise.all([sessões, cinemas])` → `cinemasComOFilme`; cabeçalho do filme (pôster/marcador, título, gênero·duração, classificação, selo de situação); estados carregando, erro com "Tentar novamente", `FilmeNaoEncontrado` ("Filme não encontrado" + link às filmes), sem cinemas ("Este filme não está em cartaz em nenhum cinema."); lista `<ul className="grade">`; link "← Voltar aos filmes" (`history.back()` se `veioDoApp()`, senão `navegar('/?aba=filmes')`) — conforme `contracts/rota-filme.md`
- [X] T008 [US1] Em `src/App.tsx` renderizar `<PaginaFilme key={rota.id} …/>` quando `rota.tipo === 'filme'`, mantendo `PaginaInicial` montada e oculta (`hidden={rota.tipo !== 'inicio'}` já cobre) e repassando `carregadoresFilme?` injetáveis; verificar que `src/hooks/useAbaNaUrl.ts` já preserva a aba fora da rota de início (sem alteração esperada)
- [X] T009 [US1] Testes em `src/componentes/PaginaFilme.test.tsx`: lista só cinemas com sessão futura do filme, estados (carregando/erro+retry/inexistente/sem cinemas/filme em breve), voltar; e em `src/App.test.tsx` o fluxo card → página do filme → voltar preservando aba e filtros da aba Filmes (FR-012)

**Checkpoint**: MVP funcional — navegação e listagem completas.

## Phase 4: User Story 2 — Filtrar por UF e cidade (P1)

**Goal**: filtros UF/Cidade dependentes, contagem, limpar e estado "sem resultado".

**Independent Test**: quickstart passos 2–3; spec US2.

- [X] T010 [US2] Em `src/componentes/PaginaFilme.tsx` adicionar `<form className="filtros" role="search" aria-label="Filtrar cinemas">` com `CampoSelect` "UF" (`todas="Todas as UFs"`) e "Cidade" (`todas="Todas as cidades"`), opções via `opcoesCinemas(cinemas, filtros.estado)`, troca de UF com `ajustarCidade`, lista via `filtrarCinemas`, botão "Limpar filtros" (desabilitado sem filtros), contagem `aria-live` ("N cinema(s)") e reusar `EstadoLista` (`rotulo="cinemas"`, com `onLimparFiltros`) apenas para o caso "nenhum cinema atende ao filtro"; o caso "sem cinemas em cartaz" permanece na T007; estado de filtros local, inicial `filtrosCinemasVazios`
- [X] T011 [US2] Ampliar `src/componentes/PaginaFilme.test.tsx`: escolher UF restringe lista e cidades; escolher cidade; trocar UF descarta cidade incompatível; limpar filtros; combinação vazia mostra mensagem + limpar; contagem atualizada

## Phase 5: User Story 3 — Comprar ingressos a partir do cinema (P1)

**Goal**: botão "Comprar Ingressos" por item, idêntico ao da página do cinema.

**Independent Test**: quickstart passos 4–5; spec US3.

- [X] T012 [US3] Em `src/componentes/CartaoCinemaDoFilme.tsx` adicionar prop `onComprar` e botão `type="button" className="botao-comprar"` com texto "Comprar Ingressos" (`aria-label` "Comprar Ingressos em {nome do cinema}" mantendo o texto visível); em `PaginaFilme.tsx` passar `onComprar={() => navegar(caminhoCompra(cinema.id, filme.id))}` (mesma ação de `PaginaCinema.tsx`)
- [X] T013 [US3] Testes: em `src/componentes/PaginaFilme.test.tsx` clicar no botão navega para `/cinemas/{cinema}/filmes/{filme}/comprar` do cinema clicado; em `src/App.test.tsx` fluxo deslogado → guarda de login com retorno à compra (reuso de `PaginaCompra`) e logado → direto à compra; "Voltar" na compra, no caso logado sem redirecionamento de login, retorna à página do filme (`history.back`); o caso pós-login/link direto mantém o comportamento atual (volta ao cinema) e não é alterado

## Phase 6: Polish & transversais

- [X] T014 [P] Estilos em `src/index.css` só se faltarem (cabeçalho do filme na página, layout 360 px), reaproveitando tokens/classes (`cartao`, `grade`, `filtros`, `link-voltar`, `botao-comprar`); validar sem rolagem horizontal em 360 px (FR-013)
- [X] T015 Rodar `npm test`, `npm run lint`, `npm run build` e executar o roteiro de `specs/005-cinemas-do-filme/quickstart.md` contra a plataforma local; registrar desvios
- [X] T016 [P] Atualizar `README.md` do front (se listar rotas/páginas) com a rota `/filmes/{id}`

## Dependências e ordem

- Phase 1 → Phase 2 (T002, T003, T004 em paralelo) → US1 → US2 → US3 → Polish.
- US2 e US3 editam `PaginaFilme.tsx` (após T007) e não são paralelizáveis entre si; ambas dependem só da US1 em termos funcionais.
- Dentro da US1: T005 e T006 em paralelo; T007 depende de T002–T004 e T006; T008 de T007; T009 de T005–T008.

## Exemplo de paralelismo

```text
Foundational:  T002 ‖ T003 ‖ T004
US1:           T005 ‖ T006  →  T007 → T008 → T009
Polish:        T014 ‖ T016
```

## Estratégia

1. **MVP** = Foundational + US1 (clicar no card e ver os cinemas). Demonstrável isoladamente.
2. Incrementar com US2 (filtros) e US3 (compra), validando cada checkpoint com o quickstart.
3. Fechar com Polish e verificação completa (testes, lint, build).
