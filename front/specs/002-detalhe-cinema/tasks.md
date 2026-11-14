# Tasks: Página de detalhes do cinema (salas, sessões e filmes)

**Input**: Design documents de `specs/002-detalhe-cinema/` (plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md)

**Tests**: incluídos, seguindo a estratégia da feature 001 (Vitest + Testing Library; lógica pura em `dominio/`, componentes colocalizados). Escreva cada teste antes da implementação correspondente e confirme que falha.

**Formato**: `- [ ] T### [P?] [US?] Descrição com caminho`. Todos os caminhos são relativos a `ingressos-golang/front/`. `[P]` = pode rodar em paralelo (arquivos distintos, sem dependência pendente).

## Phase 1: Setup

Sem dependências novas nem mudança de configuração (proxy `/catalogo`, Vitest e jsdom já existem da feature 001; o Vite já devolve `index.html` para `/cinemas/{id}`).

- [X] T001 Confirmar a linha de base antes de alterar: rodar `npm test`, `npm run lint` e `npm run build` em `front/` e registrar que passam (nenhum arquivo alterado) — _feito: 58 testes verdes_

---

## Phase 2: Foundational (bloqueia todas as histórias)

- [X] T002 [P] Adicionar tipos `TipoTela`, `Idioma`, `Sala` e `SessaoNaGrade` conforme `data-model.md` em `src/api/tipos.ts`
- [X] T003 [P] Teste de `caminhoCinema(id)` e `lerRota(pathname)` (`/cinemas/{id}` → `{ tipo: 'cinema', id }`; `/`, `/?aba=…` e qualquer outro caminho → `{ tipo: 'inicio' }`; barra final tolerada) em `src/dominio/rotas.test.ts`
- [X] T004 Implementar `caminhoCinema` e `lerRota` em `src/dominio/rotas.ts`
- [X] T005 [P] Teste do hook de rota em `src/hooks/useRota.test.tsx`: lê `location.pathname`, `navegar(caminho)` faz `pushState` e atualiza, `popstate` atualiza a rota
- [X] T006 Implementar `useRota()` retornando `[rota, navegar]` em `src/hooks/useRota.ts` (mesma técnica de `src/hooks/useAbaNaUrl.ts`)
- [X] T007 [P] Teste do cliente em `src/api/catalogo.test.ts` (acrescentar casos): `buscarCinema(id)` devolve o cinema, 404/400 → erro tipado `CinemaNaoEncontrado`, outros status → erro comum; `buscarSalas(cinemaId)` envia `cinema_id` e `ativo=true` e percorre páginas; `buscarSessoes(cinemaId)` envia `cinema_id` e percorre páginas; `buscarTodosFilmes()` não descarta `FORA_DE_CARTAZ`
- [X] T008 Implementar `buscarCinema`, `buscarSalas`, `buscarSessoes`, `buscarTodosFilmes` e a classe `CinemaNaoEncontrado` em `src/api/catalogo.ts` (reutilizar `buscarTodas`; `buscarCinema` usa `GET /api/v1/cinemas/{id}`)
- [X] T009 [P] Teste de `formatarPreco('42.00')` → `R$ 42,00`, `formatarHora`, `formatarDia` (ex.: `sáb., 15/11`) e `formatarDataHora` em `pt-BR` em `src/dominio/formatar.test.ts` (acrescentar casos)
- [X] T010 Implementar `formatarPreco`, `formatarHora`, `formatarDia`, `formatarDataHora` com `Intl` em `src/dominio/formatar.ts`
- [X] T011 [P] Teste do hook `useRecurso` em `src/hooks/useRecurso.test.tsx`: `carregando → pronto`, `erro` (guarda o erro lançado) e `recarregar()`; recarrega quando a chave (ex.: id do cinema) muda
- [X] T012 Implementar `useRecurso<T>(carregar, chave)` retornando `{ estado, dado, erro, recarregar }` em `src/hooks/useRecurso.ts` (baseado em `src/hooks/useLista.ts`, mas para um valor único)
- [X] T013 [P] Estilos base da página de detalhe (cabeçalho do cinema, seções, lista de sessões/horários como chips, link de voltar, foco visível, ≥ 360 px sem rolagem horizontal) em `src/index.css`

**Checkpoint**: rota, cliente, formatação e hook de carga prontos; as histórias podem começar.

---

## Phase 3: User Story 1 - Abrir um cinema e ver sua programação (P1) 🎯 MVP

**Goal**: clicar no card leva a `/cinemas/{id}`, que mostra os dados do cinema e as sessões vinculadas, cada uma com filme e sala; "voltar" retorna à aba de cinemas com filtros preservados.

**Independent Test**: quickstart passos 1, 2, 5, 6, 7, 8 e 9.

### Testes (US1)

- [X] T014 [P] [US1] Teste de `montarSessoes(sessoes, filmes)` em `src/dominio/programacao.test.ts`: ordena por início (desempate por número da sala), junta cada sessão ao filme pelo `filme_id`, **descarta sessões de filmes que não estão `EM_CARTAZ` ou que não constam da listagem (FR-013)**, converte `preco_base` em número
- [X] T015 [P] [US1] Teste em `src/componentes/CartaoCinema.test.tsx` (acrescentar): o cartão contém um link para `/cinemas/{id}` cujo nome acessível é o nome do cinema; clique simples navega sem recarregar (chama `navegar`); clique com Ctrl/Cmd/botão do meio não é interceptado
- [X] T016 [P] [US1] Teste de `PaginaCinema` em `src/componentes/PaginaCinema.test.tsx` (carregadores injetados): mostra nome, cidade/UF, endereço e logo; lista as sessões com título, gênero, duração, classificação, sala, horário, idioma e preço; não mostra sessões de outro cinema; estados carregando, sem sessões, cinema não encontrado (com link à lista) e erro com "Tentar novamente"
- [X] T017 [P] [US1] Teste de `App` em `src/App.test.tsx`: `/` mostra a página inicial; `/cinemas/{id}` mostra `PaginaCinema`; ao voltar para a lista a `PaginaInicial` continua com os filtros e a aba anteriores (continua montada/oculta); abrir `/cinemas/{id}` diretamente e usar "Voltar" leva a `/?aba=cinemas`

### Implementação (US1)

- [X] T018 [US1] Implementar `montarSessoes` em `src/dominio/programacao.ts` (tipo `SessaoExibida` do `data-model.md`)
- [X] T019 [US1] Tornar `src/componentes/CartaoCinema.tsx` clicável: `<a href={caminhoCinema(id)}>` no nome do cinema com área clicável cobrindo o cartão (pseudo-elemento) e `onClick` que, sem modificadores, chama `navegar`; ajustar estilos de foco/hover em `src/index.css`
- [X] T020 [P] [US1] Criar `src/componentes/CabecalhoCinema.tsx` (nome, cidade/UF, endereço, logo ou marcador padrão reutilizando o estilo do cartão)
- [X] T021 [P] [US1] _(substituída: o item de sessão foi implementado direto como `ProgramacaoFilme.tsx`, já agrupado — evita retrabalho com T036)_ Criar `src/componentes/SessaoItem.tsx` (filme: imagem/marcador, título, gênero, duração, classificação; sala: "Sala N · tipo de tela"; data e hora, idioma, preço) — lista simples cronológica nesta história
- [X] T022 [US1] Criar `src/componentes/PaginaCinema.tsx`: carrega cinema, sessões e filmes em paralelo com `useRecurso` (falha em qualquer das três consultas → erro com "Tentar novamente"; `CinemaNaoEncontrado` ou `ativo=false` → "Cinema não encontrado" com link à lista); renderiza cabeçalho, link "Voltar", estados (carregando, erro, sem sessões) e a lista de `SessaoItem`
- [X] T023 [US1] Atualizar `src/App.tsx`: usar `useRota`; montar `PaginaInicial` na primeira visita à rota `inicio` e mantê-la montada com `hidden` quando a rota é de cinema; renderizar `PaginaCinema` para `/cinemas/{id}` (carregadores injetáveis para testes)
- [X] T024 [US1] Implementar o "Voltar" em `PaginaCinema`: `history.back()` se a navegação veio da lista dentro do app, senão `navegar('/?aba=cinemas')`; garantir que `useAbaNaUrl` leia `?aba=cinemas` ao reaparecer a lista (`src/hooks/useAbaNaUrl.ts`, ajustar se necessário)
- [X] T025 [US1] Rodar `npm test` e o roteiro de quickstart (passos 1, 2, 5–9); corrigir divergências

**Checkpoint**: MVP — card → página do cinema com sessões, filme e sala.

---

## Phase 4: User Story 2 - Ver as salas do cinema (P2)

**Goal**: listar todas as salas ativas do cinema, inclusive as sem sessões.

**Independent Test**: quickstart passo 3.

### Testes (US2)

- [X] T026 [P] [US2] Teste de `salasComSessoes(salas, sessoes)` em `src/dominio/programacao.test.ts` (acrescentar): todas as salas ativas retornam, ordenadas por número, com a contagem de sessões; sala sem sessões tem contagem 0
- [X] T027 [P] [US2] Teste de `ListaSalas` em `src/componentes/ListaSalas.test.tsx`: mostra número, tipo de tela e capacidade; sala sem sessões exibe "Sem sessões programadas"; lista vazia exibe "Este cinema ainda não tem salas cadastradas."
- [X] T028 [US2] Acrescentar a `src/componentes/PaginaCinema.test.tsx` um caso: a seção de salas aparece com salas sem sessões

### Implementação (US2)

- [X] T029 [US2] Implementar `salasComSessoes` em `src/dominio/programacao.ts`
- [X] T030 [P] [US2] Criar `src/componentes/ListaSalas.tsx`
- [X] T031 [US2] Integrar em `src/componentes/PaginaCinema.tsx`: carregar `buscarSalas` em paralelo (falha de salas mostra erro da seção sem derrubar a programação) e renderizar `ListaSalas` acima da programação

**Checkpoint**: US1 + US2 funcionam de forma independente.

---

## Phase 5: User Story 3 - Organizar a programação por filme e por dia (P3)

**Goal**: sessões agrupadas por filme, em ordem cronológica, com filtro por dia.

**Independent Test**: quickstart passo 4.

### Testes (US3)

- [X] T032 [P] [US3] Teste de `agruparPorFilme(sessoes)` e `diasDisponiveis(sessoes)` e `filtrarPorDia(sessoes, dia)` em `src/dominio/programacao.test.ts` (acrescentar): cada filme aparece uma vez com sessões ordenadas; filmes ordenados pela primeira sessão (desempate por título); dias distintos no fuso local, em ordem; filtro por dia remove filmes sem sessões naquele dia; sessão às 22h locais pertence ao dia local, não ao dia UTC
- [X] T033 [P] [US3] Teste de `ProgramacaoFilme` em `src/componentes/ProgramacaoFilme.test.tsx`: um bloco por filme com seus horários (hora, sala, tipo de tela, idioma, preço)
- [X] T034 [US3] Acrescentar a `src/componentes/PaginaCinema.test.tsx`: escolher um dia mostra só as sessões dele; "Todos os dias" restaura; dia sem resultado mostra mensagem com ação de limpar

### Implementação (US3)

- [X] T035 [US3] Implementar `agruparPorFilme`, `diasDisponiveis` e `filtrarPorDia` em `src/dominio/programacao.ts`
- [X] T036 [P] [US3] _(feito junto com T021)_ Criar `src/componentes/ProgramacaoFilme.tsx` (cabeçalho do filme + lista de horários), substituindo o uso de `SessaoItem` na programação (remover `src/componentes/SessaoItem.tsx` se ficar sem uso)
- [X] T037 [P] [US3] Criar o seletor de dia (select "Todos os dias" + dias disponíveis, rótulos `formatarDia`) reutilizando `src/componentes/CampoSelect.tsx`
- [X] T038 [US3] Integrar em `src/componentes/PaginaCinema.tsx`: estado local do dia, programação agrupada por filme e mensagem de "nenhuma sessão neste dia" com botão de limpar

**Checkpoint**: todas as histórias entregues.

---

## Phase 6: Polish & cross-cutting

- [X] T038b [US1] Cobrir FR-013 em `src/componentes/PaginaCinema.test.tsx`: sessão de filme `BREVE`/`FORA_DE_CARTAZ` não aparece, o filme não é listado e a sala só com essas sessões mostra "Sem sessões programadas"
- [X] T038c [US3] Adicionar o botão "Comprar Ingressos" (FR-014) em `src/componentes/ProgramacaoFilme.tsx` com teste em `ProgramacaoFilme.test.tsx` e estilo `.botao-comprar` em `src/index.css`
- [X] T039 [P] Revisar acessibilidade da página (um `h1` por página, `h2` por seção, nome acessível do link do cartão, `aria-live` nos estados de carga, navegação completa por teclado) em `src/componentes/PaginaCinema.tsx` e filhos
- [X] T040 [P] Revisar responsividade (360 px sem rolagem horizontal; chips de horário quebram linha) em `src/index.css`
- [X] T041 Rodar `npm test`, `npm run lint` e `npm run build`; corrigir falhas
- [ ] T042 Executar o roteiro completo de `specs/002-detalhe-cinema/quickstart.md` contra a plataforma no ar e anotar divergências
- [X] T043 Atualizar `README.md` do `front/` com a nova rota `/cinemas/{id}` (e a nota de SPA fallback para hospedagem estática, se aplicável)

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 (bloqueia tudo) → US1 (MVP) → US2 e US3 (independentes entre si, ambos tocam `PaginaCinema.tsx` e `programacao.ts`, então executar em sequência ou coordenar) → Polish.
- Dentro de cada história: testes primeiro (falham) → funções puras → componentes → integração na página.
- US2 e US3 dependem apenas da US1 já existir (`PaginaCinema`); não dependem uma da outra.

## Parallel Examples

- Foundational: T002, T003, T005, T007, T009, T011, T013 em paralelo (arquivos distintos); cada implementação segue seu teste.
- US1: T014–T017 (testes) em paralelo; depois T020 e T021 em paralelo.
- US2: T026 e T027 em paralelo; T030 em paralelo com T029.
- US3: T032 e T033 em paralelo; T036 e T037 em paralelo.

## Implementation Strategy

1. **MVP**: Phases 1–3 (US1) — já responde ao pedido ("clicar no cinema → ver sala+sessão+filme vinculados").
2. Incremento 2: US2 (salas, inclusive sem sessões).
3. Incremento 3: US3 (agrupamento por filme e filtro de dia).
4. Polish e validação pelo quickstart.

Total: 43 tarefas (Setup 1, Foundational 12, US1 12, US2 6, US3 7, Polish 5).
