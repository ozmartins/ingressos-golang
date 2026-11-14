# Tasks: Comandos de Escrita do Terminal Guiados por Campos

**Input**: Design documents from `/specs/004-campos-comandos-escrita/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/formularios.md, quickstart.md

**Tests**: incluídos — a constituição (princípio IV) exige testes automatizados quando há comportamento de negócio. Testes de cada história são escritos antes da implementação e devem falhar primeiro.

**Organization**: tarefas agrupadas por história de usuário. Todos os caminhos são relativos a `ingressos-golang/web/`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência de tarefa incompleta)
- **[Story]**: história de usuário a que a tarefa pertence (US1–US4)

---

## Phase 1: Setup

- [X] T001 Confirmar a linha de base: rodar `npm test` e `npm run build` em `ingressos-golang/web/` e registrar que ambos passam antes de qualquer alteração

---

## Phase 2: Foundational (bloqueia todas as histórias)

**Purpose**: tipos de campo, conversão/validação, máquina de estados e roteamento — a base comum de todos os formulários.

- [X] T002 [P] Escrever testes de `converter()` para os tipos `texto` (obrigatório/opcional, tamanho máximo), `inteiro` (mínimo, não numérico), `logico` (`s/n`, `sim/não`, `true/false`), `opcao` (case-insensitive, valor canônico), `uuid`, `decimal` (até 2 casas, devolvido como string) e `datahora` (`AAAA-MM-DD HH:MM` interpretado no fuso local do navegador, e ISO-8601 → ISO com fuso; testar com fuso fixo) em `src/terminal/api/formularios.test.ts`
- [X] T003 Criar em `src/terminal/api/formularios.ts` os tipos `CampoDef` (união discriminada por `tipo`), `Formulario`, e `converter(campo, texto)` devolvendo `{ valor } | { erro }` (branco em opcional ⇒ `{ valor: undefined }`), até T002 passar
- [X] T004 [P] Escrever testes de `montarCorpo(campos, respostas)` (omite opcionais em branco, preserva tipos, `estado` em maiúsculas) em `src/terminal/api/formularios.test.ts`
- [X] T005 Implementar `montarCorpo()` em `src/terminal/api/formularios.ts`
- [X] T006 [P] Escrever testes da máquina de estados em `src/terminal/dialogo.test.ts`: pergunta inicial exibe nome, obrigatoriedade, opções e padrão (FR-003); resposta inválida repete a mesma pergunta sem avançar (FR-006); opcional em branco avança; ao fim exibe resumo e pede `s/n` (FR-010); `s` devolve `enviar` com o corpo; `n` encerra sem enviar
- [X] T007 Criar `src/terminal/dialogo.ts` com `iniciar(definicao, formulario, id?)` e `responder(dialogo, texto)` retornando `{ dialogo | null, linhas, enviar? }`, conforme [data-model.md](data-model.md), até T006 passar
- [X] T008 [P] Atualizar `src/terminal/api/argumentos.test.ts`: `criar` sem argumentos é válido; `criar <qualquer texto>` e `atualizar <id> <extra>` retornam erro orientando o novo uso (FR-011); `atualizar` sem id continua exigindo identificador
- [X] T009 Em `src/terminal/api/argumentos.ts`, remover `lerCorpo` e a leitura de JSON; implementar as regras de T008 (ajustar o tipo `Argumentos`, removendo `corpo`)
- [X] T010 [P] Em `src/terminal/api/tabela.ts`, trocar `uso` de `criar <json>` e `atualizar <id> <json>` por `criar` e `atualizar <id>`; atualizar `src/terminal/api/tabela.test.ts` e os exemplos/mensagens de uso em `src/terminal/api/formatar.ts` e `src/terminal/api/formatar.test.ts` para não exibirem JSON (FR-013)
- [X] T011 Em `src/terminal/despachar.ts`: extrair o envio (requisição, tratamento de 401, formatação) de `executarDefinicao` para uma função reutilizável `enviarDefinicao(def, id, corpo, ctx)`; adicionar `dialogo?` ao `Resultado` (em `src/terminal/comandos.ts`) e `continuarDialogo(dialogo, entrada, ctx)` que delega a `responder` e, ao receber `enviar`, chama `enviarDefinicao`
- [X] T012 Em `src/terminal/Terminal.tsx`: guardar `dialogo` em `useState`; com diálogo ativo, encaminhar a linha digitada a `continuarDialogo` em vez de `despachar`, ecoar a linha na saída e **não** chamá-la em `adicionar` do histórico (FR-012), decidindo isso por uma função pura exportada de `src/terminal/dialogo.ts` (ex.: `registrarNoHistorico(dialogo)`), coberta por teste em `src/terminal/dialogo.test.ts`; mostrar um prompt distinto (ex.: `?`) durante o diálogo

**Checkpoint**: a infraestrutura existe, mas nenhum formulário está cadastrado ainda; `npm test` e `npm run build` passam.

---

## Phase 3: User Story 1 — Criar um registro respondendo aos campos (P1) 🎯 MVP

**Goal**: `filmes criar`, `cinemas criar` e `sessoes criar` guiam o preenchimento campo a campo e enviam o JSON montado.

**Independent Test**: autenticado, executar `cinemas criar`, responder, confirmar e ver o cinema criado; conferir com `cinemas obter <id>`.

### Tests for User Story 1

- [X] T013 [P] [US1] Teste em `src/terminal/despachar.test.ts`: `filmes criar` sem login responde com orientação de `login` e **não** abre diálogo (FR-008)
- [X] T014 [P] [US1] Teste em `src/terminal/despachar.test.ts`: fluxo completo de `cinemas criar` com `fetch` falso — respostas → resumo → `s` → `POST /cinemas` com o corpo esperado (`estado` em maiúsculas, `ativo` omitido quando em branco) e saída do registro criado
- [X] T015 [P] [US1] Teste em `src/terminal/api/formularios.test.ts`: os formulários `filmes.criar`, `cinemas.criar` e `sessoes.criar` listam exatamente os campos de [contracts/formularios.md](contracts/formularios.md), com obrigatoriedade, opções e padrões; `preco_base` é enviado como string

### Implementation for User Story 1

- [X] T016 [US1] Em `src/terminal/api/formularios.ts`, declarar em `FORMULARIOS` os formulários `filmes.criar`, `cinemas.criar` e `sessoes.criar`
- [X] T017 [US1] Em `src/terminal/despachar.ts`, em `executarDefinicao`: para definições com `usaCorpo`, verificar autenticação **antes** de abrir o diálogo, buscar o formulário, chamar `iniciar` e devolver `Resultado` com `dialogo` e a primeira pergunta; se não houver formulário para a definição (caso de `salas criar` até a T025), devolver erro "comando ainda não disponível" sem abrir diálogo

**Checkpoint**: US1 funcional e testável sozinha para filmes, cinemas e sessões.

---

## Phase 4: User Story 2 — Atualizar um registro respondendo aos campos (P2)

**Goal**: `<controller> atualizar <id>` guia o preenchimento dos campos aceitos na atualização.

**Independent Test**: autenticado, `filmes atualizar <id>`, responder, confirmar e ver o registro atualizado.

### Tests for User Story 2

- [X] T018 [P] [US2] Teste em `src/terminal/despachar.test.ts`: `filmes atualizar <id>` percorre o formulário e envia `PUT /filmes/<id>` com o corpo montado; `filmes atualizar` sem id mostra a mensagem de uso
- [X] T019 [P] [US2] Teste em `src/terminal/api/formularios.test.ts`: `salas.atualizar` pede apenas `numero`, `tipo_tela` e `ativo`; `sessoes.atualizar` pede `filme_id`, `data_hora_inicio`, `idioma`, `preco_base`, `status` e **não** `sala_id`; `filmes.atualizar`/`cinemas.atualizar` reutilizam os campos da criação

### Implementation for User Story 2

- [X] T020 [US2] Em `src/terminal/api/formularios.ts`, declarar `filmes.atualizar`, `cinemas.atualizar`, `sessoes.atualizar` e `salas.atualizar` (reaproveitando as listas de campos da criação quando idênticas)
- [X] T021 [US2] Em `src/terminal/despachar.ts`, garantir que o `id` da linha chega ao diálogo e ao `enviarDefinicao` (substituição de `{id}` no caminho) para as ações `atualizar`

**Checkpoint**: US1 e US2 funcionam; `salas criar` ainda depende da US3.

---

## Phase 5: User Story 3 — Informar a planta de fileiras da sala (P3)

**Goal**: `salas criar` pede cinema, número, tipo de tela, fileiras (uma ou mais) e `ativo`.

**Independent Test**: autenticado, `salas criar` com duas fileiras (`A`/10 e `B`/12) cria sala com capacidade 22.

### Tests for User Story 3

- [X] T022 [P] [US3] Teste em `src/terminal/dialogo.test.ts`: sub-diálogo de fileiras (letra → assentos → tipo com padrão `NORMAL` → "adicionar outra? (s/n)"); `s` repete, `n` avança; `n` sem nenhuma fileira é recusado com aviso (FR-007)
- [X] T023 [P] [US3] Teste em `src/terminal/dialogo.test.ts`: letra inválida (vazia, minúscula é normalizada para maiúscula, dígito, mais de 5 letras) e assentos < 1 repetem a pergunta
- [X] T024 [P] [US3] Teste em `src/terminal/despachar.test.ts`: `salas criar` completo envia `POST /salas` com `fileiras: [{ fileira, assentos, tipo }]` no formato do contrato

### Implementation for User Story 3

- [X] T025 [US3] Em `src/terminal/api/formularios.ts`, adicionar o tipo de campo `fileiras` e o formulário `salas.criar`
- [X] T026 [US3] Em `src/terminal/dialogo.ts`, implementar as etapas `fileira-letra`, `fileira-assentos`, `fileira-tipo` e `fileira-mais`, acumulando `fileiras` e incluindo-as no resumo e no corpo

**Checkpoint**: os 8 comandos de escrita funcionam pelo diálogo.

---

## Phase 6: User Story 4 — Cancelar e corrigir durante o preenchimento (P4)

**Goal**: cancelar a qualquer momento sem enviar nada; instruções visíveis.

**Independent Test**: iniciar `filmes criar`, responder alguns campos, digitar `cancelar` e confirmar que nenhuma requisição saiu.

*(O resumo com confirmação `s/n` já foi entregue em T006/T007; esta fase cobre o cancelamento e as dicas.)*

### Tests for User Story 4

- [X] T027 [P] [US4] Teste em `src/terminal/dialogo.test.ts`: `cancelar` em qualquer etapa (campo, fileiras, confirmação) encerra o diálogo, devolve mensagem de cancelamento e nenhum `enviar`
- [X] T028 [P] [US4] Teste em `src/terminal/despachar.test.ts`: após `cancelar`, o `fetch` falso não recebeu nenhuma chamada

### Implementation for User Story 4

- [X] T029 [US4] Em `src/terminal/dialogo.ts`, tratar `cancelar` (case-insensitive) em qualquer etapa e incluir, na primeira pergunta de cada diálogo, a dica `(digite "cancelar" para abortar)`
- [X] T030 [US4] Em `src/terminal/Terminal.tsx`, tratar a tecla Esc com diálogo ativo como `cancelar`

**Checkpoint**: todas as histórias concluídas.

---

## Phase 7: Polish & Cross-Cutting

- [X] T031 [P] Atualizar `help` em `src/terminal/api/formatar.ts` (e teste) para descrever o fluxo guiado e a palavra `cancelar`; conferir que não há mais exemplos com JSON (FR-013)
- [X] T032 [P] Atualizar `README.md` de `ingressos-golang/web/` se ele documentar os comandos com `<json>`
- [X] T033 Rodar `npm test` (as suítes existentes de leitura, `remover`, `health`, `login` e `logout` são a verificação de regressão do FR-014) e `npm run build` (saída 0, sem avisos) e abrir o app sem erros no console (constituição IV)
- [ ] T034 Executar o roteiro de [quickstart.md](quickstart.md) contra o catálogo real (passos 1–11) e anotar o resultado

---

## Dependencies & Execution Order

- **Setup (T001)** → **Foundational (T002–T012)** → histórias → **Polish**.
- Atenção: T009–T011 deixam o `despachar.ts` sem compilar até T011/T012; fazer T009, T010, T011 e T012 antes de rodar `npm run build`.
- Dentro de Foundational: T002→T003, T004→T005, T006→T007 (em `formularios.ts`/`dialogo.ts`, T003 antes de T005 pois é o mesmo arquivo); T008→T009; T010 independente; T011 depende de T007 e T009; T012 depende de T011.
- **US1 (P1)** depende só de Foundational. **US2 (P2)** depende de US1 (reaproveita T017). **US3 (P3)** depende de Foundational e do `salas.criar` não existir antes dela; pode rodar em paralelo com US2 após US1. **US4 (P4)** depende de T007/T012.
- Testes de cada história precedem sua implementação.

## Parallel Example

```text
Foundational: T002, T004, T006, T008, T010 em paralelo (arquivos de teste distintos / tabela)
US1:          T013, T014, T015 em paralelo
US3 + US2:    após US1, as duas fases podem ser conduzidas em paralelo (arquivos de teste compartilhados exigem merge cuidadoso)
```

## Implementation Strategy

- **MVP**: Foundational + US1 (filmes, cinemas e sessões criados por diálogo). Validar com o quickstart (passos 1, 2, 4).
- **Incremental**: US2 (atualizar) → US3 (salas com fileiras) → US4 (cancelar/Esc) → Polish.
- Commits em Conventional Commits, direto na `master`, após cada fase verde.
