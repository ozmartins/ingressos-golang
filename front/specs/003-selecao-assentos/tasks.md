# Tasks: Página de seleção de assentos ("Comprar Ingressos")

**Input**: Design documents de `specs/003-selecao-assentos/` (plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md)

**Tests**: incluídos — o front já adota testes colocalizados (Vitest + Testing Library) para domínio e componentes.

**Formato**: `- [X] T### [P?] [US?] Descrição com caminho`. Caminhos relativos a `ingressos-golang/front/`.

## Phase 1: Setup

- [X] T001 Adicionar proxy `/estoque` → `http://localhost:8085` (reescrevendo o prefixo) em `proxyServicos` de `vite.config.ts` (vale para `server` e `preview`)
- [X] T002 [P] Adicionar os tipos `TipoPoltrona`, `StatusPoltrona`, `Poltrona`, `MapaPoltronas`, `RespostaBloqueio` e `Problem` em `src/api/tipos.ts` (ver data-model.md)

## Phase 2: Foundational (bloqueia todas as histórias)

- [X] T003 Estender `src/dominio/rotas.ts`: nova `Rota` `{ tipo: 'compra'; cinemaId; filmeId }`, `caminhoCompra(cinemaId, filmeId, sessaoId?)` e `lerRota` para `/cinemas/{c}/filmes/{f}/comprar` (decodificação segura; inválido → `inicio`)
- [X] T004 [P] Testes de `caminhoCompra`/`lerRota` (codificação, caminhos malformados, rotas existentes intactas) em `src/dominio/rotas.test.ts`
- [X] T005 [P] Criar `src/dominio/sessoesDoFilme.ts`: `sessoesDoFilme(sessoes, filmeId, agora)` (somente futuras, ordenadas) e `sessaoPadrao(lista, sessaoIdDaUrl?)` (id ausente, inexistente, passado ou de outro filme → primeira sessão futura) + testes em `src/dominio/sessoesDoFilme.test.ts`
- [X] T006 [P] Criar `src/dominio/pedido.ts`: `precoEmCentavos("42.00")`, `LIMITE_ASSENTOS = 10`, `alternarAssento(rotulos, rotulo, livres)` (respeita limite e só livres), `calcularPedido(precoCentavos, rotulos, taxaCentavos = 0)` + testes em `src/dominio/pedido.test.ts` (inclui SC-005: combinações de seleção/desmarcação sem float)
- [X] T007 [P] Estender `formatarPreco` em `src/dominio/formatar.ts` com `formatarCentavos(centavos)` ("R$ 84,00") + teste em `src/dominio/formatar.test.ts`
- [X] T008 Criar `src/api/estoque.ts`: base `VITE_ESTOQUE_URL ?? '/estoque'`; `buscarMapa(sessaoId, token)` e `bloquearPoltronas(sessaoId, rotulos, token)` com erros tipados (`NaoAutenticado` 401, `SessaoIndisponivel` 404/422, `ConflitoPoltronas` 409, `LimiteExcedido` 400 com `metadata.limite`, `ErroEstoque` demais); decide por status/último segmento de `type` (contracts/estoque-consumido.md) + testes com `fetch` mocado em `src/api/estoque.test.ts`
- [X] T009 Em `src/dominio/keycloak.ts`, validar que `retorno` seja caminho interno (começa com `/` e não `//`) em `urlAutenticacao`/`concluirAutenticacao`, caindo em `/` caso contrário; teste em `src/dominio/keycloak.test.ts`

**Checkpoint**: rota, domínio puro, API do estoque e proxy prontos.

## Phase 3: User Story 1 — Abrir a página de compra logado (P1) 🎯 MVP

**Objetivo**: botão abre a página com cabeçalho da sessão, mapa (somente leitura) e resumo vazio.
**Teste independente**: logado, clicar "Comprar Ingressos" e ver cinema/sala/dia/hora, mapa e resumo do filme correto; trocar de sessão; voltar ao cinema.

- [X] T010 [P] [US1] Criar `src/dominio/mapaAssentos.ts`: `agruparPorFileira(poltronas)` (fileiras ordenadas, assentos por número), `situacaoDoAssento(poltrona, selecionados)` (livre/selecionado/ocupado/indisponível), `rotuloAcessivel(poltrona, situacao)` + testes em `src/dominio/mapaAssentos.test.ts`
- [X] T011 [P] [US1] Criar `src/componentes/CabecalhoSessao.tsx`: indicador de etapa "Selecione seus assentos", cinema, sala, dia e horário, e seletor das sessões do filme (`onTrocar(sessaoId)`) + teste em `src/componentes/CabecalhoSessao.test.tsx`
- [X] T012 [P] [US1] Criar `src/componentes/MapaAssentos.tsx` (somente exibição nesta história): fileiras com letra nos dois lados (A mais próxima da tela, embaixo), barra "TELA", legenda (livre, selecionado, ocupado, indisponível, acessível, namoradeira), botões `disabled` quando ocupados, contêiner com rolagem horizontal própria + teste em `src/componentes/MapaAssentos.test.tsx`
- [X] T013 [P] [US1] Criar `src/componentes/ResumoPedido.tsx`: pôster (ou marcador 🎬), título, classificação, formato/idioma, cinema, sala, dia/horário, itens, taxa, total (0 / R$ 0,00) + teste em `src/componentes/ResumoPedido.test.tsx`
- [X] T014 [US1] Criar `src/componentes/PaginaCompra.tsx` com carregadores injetáveis (padrão `CarregadoresCinema`): carrega em paralelo cinema/filmes/sessões (`src/api/catalogo.ts`) e o mapa (`src/api/estoque.ts`); escolhe a sessão padrão, reflete a escolha em `?sessao=` (replaceState), refaz só o mapa ao trocar; estados carregando/erro (tentar novamente)/não encontrado/sem sessões/sessão encerrada; 401 ao carregar o mapa → refazer login com retorno à mesma URL; mapa vazio ou 404/422 → "seleção indisponível para esta sessão" (mapa desabilitado); botão voltar ao cinema (usa `veioDoApp`) + teste em `src/componentes/PaginaCompra.test.tsx`
- [X] T015 [US1] Ligar o botão: `onComprar` de `src/componentes/PaginaCinema.tsx` chama `navegar(caminhoCompra(cinema.id, filme.id))`; atualizar o comentário em `src/componentes/ProgramacaoFilme.tsx` e os testes `src/componentes/PaginaCinema.test.tsx`
- [X] T016 [US1] Em `src/App.tsx`, renderizar `PaginaCompra` na rota `compra` (mantendo `PaginaInicial` montada/oculta) passando o token da sessão
- [X] T017 [P] [US1] Estilos em `src/index.css` para cabeçalho da sessão, mapa, assentos, tela, legenda, resumo (grid lateral ≥ 900 px; empilhado abaixo), usando apenas os tokens `:root` existentes (claro e escuro)

**Checkpoint**: MVP — logado, o botão leva a uma página funcional (leitura).

## Phase 4: User Story 2 — Login antes de comprar (P1)

**Objetivo**: deslogado vai ao login e volta à página de compra.
**Teste independente**: deslogado, clicar no botão (ou abrir a URL direta), concluir login e chegar à compra do filme.

- [X] T018 [US2] Criar `src/hooks/useGuardaLogin.ts` (usado por `PaginaCompra`): se `sessao === null`, dispara uma única vez `urlAutenticacao('login', pathname+search)` + `location.assign`; mostra "Redirecionando para o login…" (`role="status"`) e não monta a página nem consulta o estoque
- [X] T019 [US2] Integrar `useGuardaLogin` em `src/componentes/PaginaCompra.tsx` (clique e acesso direto; token expirado conta como deslogado via `lerSessao`)
- [X] T020 [P] [US2] Testes em `src/componentes/PaginaCompra.test.tsx` (ou `src/hooks/useGuardaLogin.test.tsx`): deslogado → redireciona com retorno correto e sem chamar o estoque; logado → não redireciona; uma única chamada em re-render; retorno do Keycloak leva à rota de compra (reuso de `useSessao`)

## Phase 5: User Story 3 — Escolher assentos e resumo (P2)

**Objetivo**: seleção/desmarcação com resumo, limite e números.
**Teste independente**: selecionar/desmarcar, tentar ocupado e o 11º, alternar números, usar teclado.

- [X] T021 [US3] Tornar `MapaAssentos.tsx` interativo: `onAlternar(rotulo)`, `aria-pressed`, `aria-label` via `rotuloAcessivel`, foco visível, alternância "Exibir números dos assentos" (switch acessível) + testes em `src/componentes/MapaAssentos.test.tsx`
- [X] T022 [US3] Em `PaginaCompra.tsx`, manter `selecionados` (zera ao trocar de sessão), usar `alternarAssento`/`calcularPedido`, avisar ao atingir o limite (`role="alert"`) e passar itens/assentos/taxa/total ao `ResumoPedido`
- [X] T023 [P] [US3] Completar `ResumoPedido.tsx` com lista de assentos escolhidos, "Itens", "Total taxa", "Total" via `formatarCentavos` + testes (0, 1, 2 itens; limite) em `src/componentes/ResumoPedido.test.tsx`

## Phase 6: User Story 4 — Confirmar e reservar (P3)

**Objetivo**: "Continuar" cria a reserva temporária e trata conflito/erros.
**Teste independente**: reservar com sucesso; conflito 409; token expirado; falha de rede.

- [X] T024 [US4] Em `ResumoPedido.tsx`, ação "Continuar" (desabilitada sem assentos ou durante a chamada) com `onContinuar`
- [X] T025 [US4] Em `PaginaCompra.tsx`, implementar o fluxo de reserva (`bloquearPoltronas`): 201 → painel de confirmação com assentos e prazo (`expira_em` formatado, sem reenvio); 409 → aviso, recarregar mapa, seleção ∩ livres; 400 limite → aviso; 404/422 → "sessão não aceita reserva"; 401 → guardar seleção em `sessionStorage` (chave por sessão) e refazer login com retorno à mesma página, restaurando-a ao voltar; rede/5xx → erro com tentar novamente
- [X] T026 [P] [US4] Testes em `src/componentes/PaginaCompra.test.tsx` para cada desfecho (201, 409, 400, 401 com preservação, 5xx, duplo clique envia uma só requisição)

## Phase 7: Polish & transversais

- [X] T027 [P] Revisão de acessibilidade/responsividade (360 px sem rolagem horizontal da página; alvos de toque; foco; contraste nos temas claro/escuro) em `src/index.css`
- [X] T028 [P] Garantir que não há cores fora dos tokens (grep por `#` e `rgb` nos estilos novos em `src/index.css`) — SC-007
- [X] T029 Executar `npm run lint`, `npm test` e `npm run build` em `ingressos-golang/front/` e corrigir falhas
- [ ] T030 Rodar os cenários de `specs/003-selecao-assentos/quickstart.md` contra a plataforma no ar (docker compose) e registrar o resultado — **parcial**: proxy `/estoque` (401 sem token), `/catalogo` (200) e rota SPA verificados por curl; fluxo completo com login Keycloak no navegador fica para validação manual
- [X] T031 [P] Documentar a rota e o proxy `/estoque` em `README.md` do front

## Dependências

- Setup (T001–T002) → Foundational (T003–T009) → US1 → US2 → US3 → US4 → Polish.
- US2 depende de US1 (a página existir) mas é testável isoladamente com carregadores mocados; US3 depende de US1 (mapa/resumo); US4 depende de US3 (seleção).
- T014 depende de T008, T010–T013; T016 depende de T014; T019 depende de T016/T018; T025 depende de T008, T022, T024.

## Paralelismo

- Foundational: T004, T005, T006, T007 em paralelo (arquivos distintos), após T003.
- US1: T010, T011, T012, T013, T017 em paralelo; T014 só após eles.
- US3: T023 em paralelo a T021.

## Estratégia

- **MVP**: Phases 1–3 (+ US2 para não expor a página a anônimos: entregar US1+US2 juntos, ambos P1).
- Depois US3 (interação) e US4 (reserva) em incrementos; cada fase termina com `npm test` verde e commit direto na `master`.
