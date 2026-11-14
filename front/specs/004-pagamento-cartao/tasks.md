# Tasks: Página de pagamento via cartão (simulada)

**Input**: `specs/004-pagamento-cartao/` (plan.md, spec.md, research.md, data-model.md, contracts/rota-pagamento.md, quickstart.md)

**Tests**: incluídos, seguindo a convenção do front (testes colocalizados com Vitest + Testing Library).

**Formato**: `- [ ] ID [P?] [US?] Descrição com caminho` — `[P]` = paralelizável (arquivos diferentes, sem dependência pendente). Caminhos relativos a `ingressos-golang/front/`.

## Phase 1: Setup

- [X] T001 Confirmar a base verde antes de começar: rodar `npm test`, `npm run lint` e `npm run build` em `ingressos-golang/front/` e anotar falhas pré-existentes

---

## Phase 2: Foundational (bloqueia todas as histórias)

- [X] T002 Adicionar a rota `{ tipo: 'pagamento'; cinemaId; filmeId }` em `src/dominio/rotas.ts`: `caminhoPagamento(cinemaId, filmeId)` → `/cinemas/{id}/filmes/{id}/pagamento` e reconhecimento em `lerRota` (5 segmentos, último `pagamento`)
- [X] T003 [P] Testes de `caminhoPagamento`/`lerRota` (ida e volta, ids com caracteres especiais, caminho inválido → `inicio`) em `src/dominio/rotas.test.ts`
- [X] T004 [P] Criar `src/dominio/reservaPagamento.ts`: tipo `ReservaPagamento` (ver data-model.md), `salvarReserva`, `lerReserva(cinemaId, filmeId)` (valida JSON, campos, `rotulos` não vazio, ids da rota; tolera `sessionStorage` indisponível), `limparReserva`, `segundosRestantes(expiraEm, agora)`, chave `pagamento:reserva`
- [X] T005 [P] Testes de `reservaPagamento` (válida, malformada, ids divergentes, rotulos vazio, expirada, limpar, contagem) em `src/dominio/reservaPagamento.test.ts`

**Checkpoint**: rota e instantâneo da reserva prontos.

---

## Phase 3: User Story 1 — Ir ao pagamento ao continuar (P1) 🎯 MVP

**Objetivo**: "Continuar" com reserva criada abre a página de pagamento com o resumo e o prazo.

**Teste independente**: logado, selecionar assentos, "Continuar" → abre `/pagamento` com resumo correto; conflito/erro permanece na seleção.

- [X] T006 [P] [US1] Teste em `src/componentes/PaginaCompra.test.tsx`: reserva bem-sucedida grava o instantâneo e navega para `caminhoPagamento`; conflito (409) e erro NÃO navegam nem gravam
- [X] T007 [US1] Em `src/componentes/PaginaCompra.tsx` (`continuar`, no `.then` de `c.reservar`): montar `ReservaPagamento` (filme, cinema, sala, início, rótulos, preço/taxa/total em centavos via `pedido`, `expira_em`), chamar `salvarReserva` e `navegar(caminhoPagamento(cinemaId, filmeId))`; remover o painel local "Assentos reservados" que fica obsoleto (e ajustar testes existentes dele)
- [X] T008 [P] [US1] Criar `src/componentes/ResumoReserva.tsx`: pôster (marcador padrão se ausente), título, cinema, sala, dia/horário, assentos, quantidade, taxa e total em reais (reuso de `formatar.ts`) e contagem regressiva `role="timer"` em mm:ss; teste em `src/componentes/ResumoReserva.test.tsx`
- [X] T009 [US1] Criar `src/componentes/PaginaPagamento.tsx` (esqueleto): `useGuardaLogin`, leitura do instantâneo, estados "sem reserva" (com links a `caminhoCompra` e início) e "reserva válida" mostrando `ResumoReserva`; link "Voltar" para `caminhoCompra(cinemaId, filmeId, sessaoId)`; props injetáveis (`irParaLogin`, relógio) para teste
- [X] T010 [US1] Em `src/App.tsx`, renderizar `PaginaPagamento` quando `rota.tipo === 'pagamento'` (key por cinema/filme, passando `sessao`)
- [X] T011 [P] [US1] Testes de `PaginaPagamento` em `src/componentes/PaginaPagamento.test.tsx`: abre com resumo; sem instantâneo → "Nenhuma reserva em andamento"; voltar leva à seleção; sem login → redireciona ao login
- [X] T012 [P] [US1] Teste de roteamento em `src/App.test.tsx`: a rota `/pagamento` renderiza a página de pagamento

**Checkpoint**: o fluxo Continuar → pagamento já é demonstrável (sem formulário ainda).

---

## Phase 4: User Story 2 — Preencher o cartão e pagar (P1)

**Objetivo**: formulário com máscara/validação e pagamento simulado (aprovado/recusado).

**Teste independente**: dados válidos → aprovado; vazios/inválidos → erros por campo; final `0000` → recusado.

- [X] T013 [P] [US2] Criar `src/dominio/cartao.ts`: `formatarNumero` (grupos de 4, só dígitos, máx. 19), `formatarValidade` (MM/AA), `validarCartao(dados, agora)` → erros por campo (`numero`, `nome`, `validade`, `cvv`) conforme data-model.md, `cartaoRecusado(numero)` (termina em `0000`)
- [X] T014 [P] [US2] Testes de `cartao` (máscaras, número livre, validade vencida/fim do mês/mês inválido, CVV 3–4, nome, número colado com espaços/hífens, recusa `4111111000030000`) em `src/dominio/cartao.test.ts`
- [X] T015 [P] [US2] Criar `src/componentes/FormularioCartao.tsx`: 4 campos com `<label>`, `inputMode="numeric"`, `autoComplete` (`cc-number`, `cc-name`, `cc-exp`, `cc-csc`), máscara ao digitar, erro por campo via `aria-invalid`/`aria-describedby`, botão "Pagar" (desabilitado em processamento), aviso visível "Simulação — não informe dados reais de cartão"; teste em `src/componentes/FormularioCartao.test.tsx`
- [X] T016 [US2] Em `src/componentes/PaginaPagamento.tsx`: máquina de estados `ocioso | processando | aprovado | recusado`; ao pagar validar (T013), mostrar "Processando…", aguardar atraso injetável (~2 s, `setTimeout`) e resolver aprovado ou recusado (`cartaoRecusado`); ignorar duplo envio; dados do cartão apenas em estado React (sem storage, rede ou `console`)
- [X] T017 [US2] Em `src/componentes/PaginaPagamento.tsx`: tela "Pagamento aprovado" (`role="status"`, resumo, link para início, `limparReserva`) e "Pagamento recusado" (`role="alert"`, botão "Tentar outro cartão" que volta a `ocioso` mantendo o resumo)
- [X] T018 [US2] Testes em `src/componentes/PaginaPagamento.test.tsx` (fake timers): em branco → erros e nada processado; inválidos por campo; válido → processando → aprovado e instantâneo removido; final `0000` → recusado → tentar outro; duplo clique processa uma vez; nenhuma chamada `fetch` e nenhum dado de cartão em `sessionStorage`/`localStorage`
- [X] T019 [P] [US2] Estilos do formulário, do resumo e dos estados em `src/index.css` usando apenas tokens existentes (`--superficie`, `--destaque`, `--borda`…); layout sem rolagem horizontal em 360 px

**Checkpoint**: pagamento simulado completo.

---

## Phase 5: User Story 3 — Acesso indevido e reserva expirada (P2)

**Objetivo**: tratar login, ausência de reserva e prazo vencido.

**Teste independente**: abrir `/pagamento` deslogado, sem reserva e com prazo vencido.

- [X] T020 [US3] Em `src/componentes/PaginaPagamento.tsx`: contagem regressiva (tick de 1 s; limpar no desmontar) que ao zerar muda para "Reserva expirada" (formulário desabilitado, link "Escolher assentos"); revalidar a expiração ao enviar e ao concluir o processamento (expirada → bloqueia/aviso, sem aprovar)
- [X] T021 [US3] Testes em `src/componentes/PaginaPagamento.test.tsx` (fake timers): contagem em mm:ss decresce; ao zerar mostra "Reserva expirada" e "Pagar" fica indisponível; expirar durante o processamento impede a aprovação; deslogado com retorno ao endereço do pagamento; recarga (nova montagem) com instantâneo válido mantém a página

**Checkpoint**: todas as histórias independentes e verificadas.

---

## Phase 6: Polish

- [X] T022 [P] Revisão de acessibilidade em `PaginaPagamento`/`FormularioCartao`: ordem de foco, foco no primeiro campo com erro ao enviar, anúncios de resultado e do timer, operável só por teclado (SC-005)
- [X] T023 [P] Conferir o CSS novo contra a paleta (nenhuma cor fora dos tokens, SC-006) e o layout em 360 px
- [X] T024 Rodar `npm test`, `npm run lint` e `npm run build` em `ingressos-golang/front/` e corrigir o que falhar
- [X] T026 [US3] Isolar a reserva por usuário: `idDoUsuario(token)` em `src/dominio/keycloak.ts`, campo `usuarioId` em `src/dominio/reservaPagamento.ts` (gravado em `PaginaCompra.tsx`, conferido em `lerReserva`/`PaginaPagamento.tsx`) e testes correspondentes (resolve o achado F2 do analyze)
- [X] T025 Percorrer os cenários de `specs/004-pagamento-cartao/quickstart.md` manualmente (incluindo Network/Storage sem dados de cartão) e commitar na `master` (Conventional Commits, em inglês)

---

## Dependências

- Phase 2 (T002–T005) → bloqueia todas as histórias.
- US1 → base de US2 e US3 (ambas editam `PaginaPagamento.tsx`); US2 → antes de US3 (T020 usa os estados de T016).
- Dentro de US1: T007 depende de T002/T004; T009 depende de T004/T008; T010 depende de T009.
- T013–T015 e T019 independem entre si e podem rodar antes de T016.

## Paralelismo

- Foundational: T003, T004, T005 (T003 após T002).
- US1: T006, T008, T011, T012 (testes/componente em arquivos distintos).
- US2: T013, T014, T015, T019 simultâneas; T016–T018 sequenciais (mesmo arquivo).
- Polish: T022, T023.

## Estratégia

- **MVP**: Phases 1–3 (US1): "Continuar" abre a página de pagamento com o resumo.
- Incremental: US2 entrega o formulário e o pagamento simulado (objetivo principal do pedido); US3 endurece expiração/acesso; depois Polish.
