# Tasks: Comandos do Terminal para a API Servico-Catalogo

**Input**: Design documents from `/specs/003-catalogo-api-commands/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/comandos-terminal.md, quickstart.md
**Tests**: Incluídos para a lógica pura (constituição, princípio IV), com `fetch` falso injetado; a UI e a integração real são verificadas via `quickstart.md`.

**Format**: `- [ ] [ID] [P?] [Story?] Descrição com caminho`. Caminhos relativos a `ingressos-golang/web/`.

## Phase 1: Setup

- [X] T001 Adicionar em `vite.config.ts` o bloco `server.proxy`: `/catalogo` → `http://localhost:8082` e `/identidade` → `http://localhost:8081`, ambos com `changeOrigin: true` e `rewrite` removendo o prefixo (necessário porque o catálogo não envia cabeçalhos CORS); repetir o mesmo bloco em `preview.proxy`

## Phase 2: Foundational (bloqueia todas as histórias)

- [X] T002 [P] Criar a tabela declarativa dos 21 endpoints (`controller`, `acao`, `metodo`, `caminho`, `raiz`, `protegido`, `filtros`, `usaId`, `usaCorpo`, `uso`, `descricao`) conforme `contracts/comandos-terminal.md`, e a função `buscarDefinicao(controller, acao)` (insensível a maiúsculas), em `src/terminal/api/tabela.ts`
- [X] T003 [P] Criar testes da tabela: exatamente 21 definições, pares `(controller, acao)` únicos, pares `(metodo, caminho)` únicos, `health` é o único com `raiz`, escritas são `protegido`, leituras não, em `src/terminal/api/tabela.test.ts`
- [X] T004 [P] Criar testes de `lerArgumentos` (filtros `chave=valor` aceitos/recusados conforme `filtros` da definição, id obrigatório, id ausente → erro de uso, argumentos extras) em `src/terminal/api/argumentos.test.ts`
- [X] T005 Implementar `lerArgumentos(definicao, resto)` retornando `{ id?, query }` ou erro de uso com a sintaxe da definição, em `src/terminal/api/argumentos.ts` (depende de T002)
- [X] T006 [P] Criar testes de `requisitar` com `fetch` falso: monta URL (`/catalogo/api/v1/...` e `/catalogo/health` para `raiz`), query string, cabeçalho `Authorization: Bearer` quando houver token, corpo JSON com `Content-Type`, resposta JSON e `204` sem corpo, `Location`, erro de rede e timeout, em `src/terminal/api/cliente.test.ts`
- [X] T007 Implementar `requisitar(contexto, requisicao)` com `fetch` injetável, `AbortController` (timeout de 10 s), parse tolerante do corpo (`unknown`) e erros tipados `rede` e `timeout`, em `src/terminal/api/cliente.ts` (depende de T002)
- [X] T008 [P] Criar testes de `formatarResposta` e `formatarErro`: texto amigável por recurso (filme, cinema, sala, sessão, health), resumo de paginação, erro com `<status> <título>`, detalhe e `campo: mensagem`, mensagens para 400/401/404/409/503, texto puro (`<b>` preservado), em `src/terminal/api/formatar.test.ts`
- [X] T009 Implementar `formatarResposta` (200 em texto amigável por recurso, sem JSON bruto) e `formatarErro` (problema padronizado e falhas de rede/timeout) retornando `Linha[]`, em `src/terminal/api/formatar.ts`

## Phase 3: User Story 1 - Consultar o catálogo pelo terminal (P1) 🎯 MVP

**Goal**: Os 10 comandos de leitura mais `health` funcionam pelo terminal.
**Independent Test**: `npm run dev` com a plataforma no ar; `filmes listar`, `filmes obter <id>`, `cinemas listar`, `salas obter <id>`, `sessoes listar`, `health` (passos 2–7 e 17 do `quickstart.md`).

- [X] T010 [P] [US1] Criar testes de `despachar` para leituras com `fetch` falso: `filmes listar status=EM_CARTAZ page=2 page_size=5` gera a query correta, `cinemas obter <id>`, `salas listar cinema_id=<uuid>`, `sessoes listar data=2026-11-20`, `health` na raiz, 404 vira erro, rede indisponível vira mensagem, e comandos locais (`echo`, `date`) continuam funcionando, em `src/terminal/despachar.test.ts`
- [X] T011 [US1] Estender o tipo `Resultado` em `src/terminal/comandos.ts` com `sessao?: Sessao` (importando o tipo de `api/sessao.ts`; criar ali apenas a interface `Sessao { token: string | null }` por ora) e exportar `AJUDA_LOCAL` (a lista usada pelo `help` local); criar `despachar(entrada, contexto): Promise<Resultado>` que roteia `<controller> <ação>`/`health` pela tabela (T002, T005, T007, T009) e delega o restante a `executar()`; definir o tipo `Contexto` (`agora`, `fetchFn`, `base`, `sessao`) em `src/terminal/despachar.ts`; `despachar` nunca altera o contexto no lugar
- [X] T012 [US1] Integrar em `src/terminal/Terminal.tsx`: chamar `await despachar(...)` com `fetch` real e bases `/catalogo` e `/identidade`, passando a `Sessao` do estado (inicialmente sem token); estado `ocupado` que ignora Enter durante a chamada e mostra um indicador de espera no prompt; resultados anexados na ordem dos comandos; quando `resultado.sessao` vier definido, substituí-la no estado (depende de T011)
- [ ] T013 [US1] Rodar `npm test` e `npm run build`; verificar manualmente os passos 2–7 e 17 do `quickstart.md` contra os serviços reais

## Phase 4: User Story 2 - Autenticar para operações de escrita (P2)

**Goal**: `login`, `logout` e bloqueio de comandos protegidos sem token; segredo nunca exposto.
**Independent Test**: Passos 8, 9 e 16 do `quickstart.md`.

- [X] T014 [P] [US2] Criar testes de sessão com `fetch` falso: `login` envia `grant_type=client_credentials` form-urlencoded ao endpoint do realm `cinema` e guarda o token; credencial inválida (400/401) não guarda token e não ecoa o segredo; `logout` descarta; `mascarar('login id segredo')` → `login id ****`; entradas que não são `login` não mudam, em `src/terminal/api/sessao.test.ts`
- [X] T015 [US2] Implementar `login`, `logout` e `mascarar` em `src/terminal/api/sessao.ts` (a interface `Sessao` já existe desde a T011; token só em memória); `login` e `logout` devolvem a nova `Sessao` em vez de alterar a recebida
- [X] T016 [US2] Estender `src/terminal/despachar.ts`: rotear `login`/`logout`, devolvendo a nova sessão em `Resultado.sessao`; para definições `protegido` sem token devolver a orientação de usar `login` sem enviar requisição; enviar o token nas protegidas; em resposta 401 devolver `sessao: { token: null }` e orientar nova autenticação (depende de T015)
- [X] T017 [US2] Adicionar em `src/terminal/despachar.test.ts` casos: escrita sem token → orientação e nenhuma chamada ao `fetch`; com token envia `Authorization`; 401 devolve `sessao.token === null`; `login` válido devolve `sessao` com token e inválido não devolve `sessao`; `logout` devolve `sessao.token === null`; comandos que não mudam a sessão não trazem `sessao`
- [X] T018 [US2] Em `src/terminal/Terminal.tsx`: aplicar `mascarar` ao eco da linha e ao que entra no histórico (`adicionar`); nunca logar o token ou o segredo no console (a atualização do estado da `Sessao` por `resultado.sessao` já vem da T012)
- [ ] T019 [US2] Verificar manualmente os passos 8, 9 e 16 do `quickstart.md`, incluindo que o segredo não aparece no eco nem ao navegar o histórico com ↑

## Phase 5: User Story 3 - Cadastrar, atualizar e remover registros (P3)

**Goal**: Os 10 comandos de escrita (`criar`, `atualizar`, `remover`) para os 4 controllers.
**Independent Test**: Passos 10–15 do `quickstart.md`.

- [X] T020 [P] [US3] Criar testes de `lerCorpo` (JSON válido após o id/ação, corpo com espaços e aspas sem escape, corpo ausente → erro de uso, JSON inválido → erro de sintaxe) em `src/terminal/api/argumentos.test.ts`
- [X] T021 [US3] Implementar `lerCorpo(resto)` e integrá-lo a `lerArgumentos` para definições com `usaCorpo`, em `src/terminal/api/argumentos.ts`
- [X] T022 [P] [US3] Adicionar testes de formatação em `src/terminal/api/formatar.test.ts`: `201` mostra o JSON e `Location`, `204` mostra `Removido com sucesso`, erro com lista `errors[]`
- [X] T023 [US3] Estender `formatarResposta` com o tratamento de `201` (corpo + `Location`) e `204` em `src/terminal/api/formatar.ts`
- [X] T024 [US3] Adicionar em `src/terminal/despachar.test.ts` casos de escrita: `filmes criar {...}`, `salas atualizar <id> {...}`, `sessoes remover <id>`, JSON inválido sem chamada ao `fetch`, 400 com erros por campo, 409 de conflito
- [ ] T025 [US3] Verificar manualmente os passos 10–15 do `quickstart.md` contra os serviços reais (criar → obter → atualizar → remover um filme)

## Phase 6: User Story 4 - Descobrir os comandos de API (P4)

**Goal**: `help` e mensagens de uso baseados na tabela.
**Independent Test**: Passo 1 e 18 do `quickstart.md`.

- [X] T026 [P] [US4] Criar testes de ajuda em `src/terminal/api/formatar.test.ts`: `formatarAjuda` lista os 21 comandos de API agrupados por controller mais `login` e `logout`; `formatarUso(definicao)` mostra a sintaxe; `formatarAcoes(controller)` lista as ações válidas
- [X] T027 [US4] Implementar `formatarAjuda`, `formatarUso` e `formatarAcoes` a partir da tabela em `src/terminal/api/formatar.ts`
- [X] T028 [US4] Em `src/terminal/despachar.ts`: interceptar `help` antes de delegar a `executar()` e montá-lo unindo `AJUDA_LOCAL` (exportada por `comandos.ts`) à ajuda de API (`formatarAjuda`); `comandos.ts` não passa a conhecer os comandos de API; `<controller>` sem ação ou com ação desconhecida mostra as ações válidas; argumentos faltando mostram o uso; adicionar os casos em `src/terminal/despachar.test.ts`
- [ ] T029 [US4] Verificar manualmente os passos 1 e 18 do `quickstart.md`

## Phase 7: Polish

- [X] T030 Atualizar `README.md` com os comandos de API, o proxy do Vite em desenvolvimento (e a ressalva de que produção exigirá proxy equivalente) e o pré-requisito de a plataforma estar no ar
- [ ] T031 Executar `npm test`, `npm run build` e percorrer toda a tabela de verificação do `quickstart.md`; confirmar console do navegador sem erros e sem token/segredo em nenhum log

## Dependencies & Execution Order

- Setup (T001) → Foundational (T002–T009) → US1 → US2 → US3 → US4 → Polish.
- US2 depende de US1 (`despachar` e `Terminal` já integrados). US3 depende de US2 (token para escrita). US4 depende da tabela e de `despachar` (US1), podendo ser feita antes de US2/US3 se necessário.
- Dentro da fundação: T005 depende de T002; T007 depende de T002; T009 pode usar os tipos de T007.
- Arquivos compartilhados que exigem sequência: `despachar.ts` (T011, T016, T028), `Terminal.tsx` (T012, T018), `argumentos.ts` (T005, T021), `formatar.ts` (T009, T023, T027) e `despachar.test.ts` (T010, T017, T024, T028).
- Paralelos: T002/T003/T004/T006/T008 (arquivos distintos); T014; T020/T022; T026.

## Implementation Strategy

MVP = Fases 1–3 (leitura + `health`, 11 dos 21 endpoints, sem credenciais). Depois autenticação (US2), escrita (US3, completa os 21) e ajuda (US4), por fim Polish. Total: 31 tarefas (Setup 1, Foundational 8, US1 4, US2 6, US3 6, US4 4, Polish 2).
