# Tasks: Processamento Assíncrono de Pagamentos (Servico-Pagamento)

**Input**: Design documents from `specs/001-pagamento-assincrono/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: obrigatórios pelo princípio II da constituição do workspace para o núcleo
de domínio e para **toda** operação exposta — incluindo as expostas por evento, com
caminho de sucesso e cada categoria de erro declarada em `contracts/erros.md`.
Adaptadores, fiação e código de composição não estão sujeitos à obrigação; onde há
teste deles aqui, é porque a garantia só é demonstrável com infraestrutura real.

**Organization**: agrupadas por história de usuário, para que cada uma seja
implementável, testável e entregável de forma independente.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos distintos, sem dependência pendente)
- **[Story]**: US1..US4, conforme `spec.md`
- Todo caminho de arquivo é relativo a `pagamento/`

---

## Phase 1: Setup

**Purpose**: esqueleto do módulo e ferramental. Nada de domínio ainda.

- [X] T001 Criar o módulo Go e a árvore de diretórios de `plan.md` §Source Code em `go.mod` (module `github.com/oseias/ingressos-golang/pagamento`, Go 1.25) e nas pastas `cmd/`, `internal/{domain,usecase,adapter,platform}`, `migrations/`, `test/integration/`, `scripts/`
- [X] T002 [P] Configurar linter e alvos de build em `.golangci.yml` e `Makefile` (`build`, `test`, `test-integration`, `lint`, `migrate-up`, `migrate-down`, `run`)
- [X] T003 [P] Subir infraestrutura local em `docker-compose.yml` (PostgreSQL 16 e RabbitMQ com painel), `Dockerfile`, `.dockerignore` e `.gitignore`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: o que toda história precisa. Nenhuma história começa antes deste bloco.

**⚠️ CRÍTICO**: T004 a T014 bloqueiam as Fases 3 a 6.

- [X] T004 Implementar leitura e validação da configuração em `internal/platform/config/config.go`, com todas as chaves de `research.md` D11 (banco, AMQP, prefetch, limite de entregas, prazo do adquirente, JWKS, emissor, público, OTLP, nível de log) e recusa de subir com variável ausente ou malformada
- [X] T005 [P] Configurar observabilidade em `internal/platform/observability/` — `log/slog` em JSON e inicialização do OpenTelemetry (métricas e rastreamento por OTLP)
- [X] T006 [P] Escrever a migração `migrations/000001_criar_transacoes.up.sql` e `.down.sql` com o DDL de `data-model.md` §1, incluindo as cinco restrições `CHECK` e a unicidade de `reserva_id`
- [X] T007 Implementar o núcleo de domínio em `internal/domain/transacao/transacao.go` — entidade, os cinco estados, as transições permitidas a partir de `PROCESSANDO`, o vocabulário fechado de motivos (`data-model.md` §3) e a regra de expiração contra relógio injetado
- [X] T008 [P] Testar o domínio em `internal/domain/transacao/transacao_test.go` — toda transição permitida, toda transição proibida a partir de estado terminal, expiração no limite e além dele, e a invariante de que `PENDENTE_VERIFICACAO` nunca é anunciável (princípio II)
- [X] T009 Declarar as portas em `internal/usecase/ports.go` — `Repositorio`, `Adquirente` (com os três desfechos distinguíveis de `research.md` D7), `Publicador`, `Relogio` e `GeradorID`
- [X] T010 Implementar o pool `pgx` e o esqueleto do repositório em `internal/adapter/postgres/transacoes.go`
- [X] T011 Declarar a topologia AMQP em `internal/adapter/amqp/topologia.go` conforme `contracts/eventos.md` §5 — exchange, DLX, fila quórum com `x-delivery-limit` e fila morta —, de forma idempotente e recusando subir se não puder ser garantida
- [X] T012 [P] Implementar o adquirente simulado em `internal/adapter/adquirente/simulado/simulado.go`, com regras por forma de pagamento e faixa de valor cobrindo aprovação, recusa, demora e indisponibilidade (`research.md` D7)
- [X] T013 [P] Implementar as sondas em `internal/platform/health/health.go` — vivacidade e prontidão (banco e canal de eventos), conforme FR-025
- [X] T014 Montar a composição em `cmd/pagamento/main.go` — configuração → adaptadores → casos de uso → servidor HTTP e consumidor, com desligamento ordenado

**Checkpoint**: base pronta; as histórias podem começar.

---

## Phase 3: User Story 1 — Cobrar automaticamente uma reserva anunciada (P1) 🎯 MVP

**Goal**: um `reserva.criada` válido vira cobrança e o desfecho é anunciado no barramento.

**Independent Test**: publicar um `reserva.criada` e verificar que a transação é
registrada em `PROCESSANDO` antes de qualquer cobrança, que a cobrança é tentada uma
vez, que a transação chega a estado final e que exatamente um anúncio é publicado.

- [X] T015 [US1] Implementar `CriarSeAusente` em `internal/adapter/postgres/transacoes.go` com `INSERT ... ON CONFLICT (reserva_id) DO NOTHING RETURNING *`, devolvendo se criou e a linha vigente (`data-model.md` §4)
- [X] T016 [US1] Implementar `Finalizar` em `internal/adapter/postgres/transacoes.go`, condicionando a escrita a `status = 'PROCESSANDO'` na cláusula `WHERE` e gravando `codigo_transacao_gateway`, `motivo_falha`, `pago_em` e `atualizado_em = now()` (FR-023)
- [X] T017 [US1] Implementar `MarcarAnunciado` em `internal/adapter/postgres/transacoes.go`, permitido apenas a partir de estado terminal anunciável
- [X] T018 [P] [US1] Implementar o publicador em `internal/adapter/amqp/publicador.go` com *publisher confirms*, envelope (`evento`, `versao`, `ocorrido_em`), `message_id` = `transacao_id` e reinjeção do `traceparent`, montando os payloads de `contracts/eventos.md` §2 e §3
- [X] T019 [US1] Implementar o caso de uso em `internal/usecase/processar_pagamento.go` — validar o anúncio (FR-003, FR-004), conferir a expiração (FR-005), registrar em `PROCESSANDO`, cobrar pela porta, finalizar e então publicar e marcar, nessa ordem invariável
- [X] T020 [US1] Implementar o consumidor em `internal/adapter/amqp/consumidor.go` — ack somente após publicação e marcação; `nack` sem reenfileirar para anúncio inválido; `nack` com reenfileiramento para falha transitória (`contracts/eventos.md` §4)
- [X] T021 [P] [US1] Testar o caso de uso em `internal/usecase/processar_pagamento_test.go` com portas falsas — aprovado, recusado, reserva expirada e cada forma de anúncio inválido (campo ausente, valor não positivo, forma desconhecida), verificando a ordem gravar → publicar → marcar
- [X] T022 [P] [US1] Testar a conformidade dos fatos publicados em `internal/adapter/amqp/publicador_test.go` — campos, tipos e formato de instante batendo com `contracts/eventos.md` §2 e §3
- [X] T023 [P] [US1] Implementar o publicador manual em `cmd/publicar/main.go`, gerando `reserva.criada` no formato que este serviço exige, com valor, forma e prazo parametrizáveis (necessário porque o estoque ainda não publica os dois campos — `research.md` D1)
- [X] T024 [US1] Testar ponta a ponta em `test/integration/cobranca_test.go` com Testcontainers (PostgreSQL e RabbitMQ reais) — `reserva.criada` publicado leva a `PAGO` e a um `pagamento.sucesso` com o payload do contrato

**Checkpoint**: o serviço cobra e anuncia. É a entrega mínima com valor.

---

## Phase 4: User Story 2 — Nunca cobrar a mesma reserva duas vezes (P1)

**Goal**: entregas repetidas ou simultâneas produzem exatamente uma cobrança, e nenhum
resultado se perde por queda entre gravar e publicar.

**Independent Test**: publicar o mesmo `reserva.criada` muitas vezes, em sequência e em
paralelo, e verificar uma única transação, uma única cobrança e um único anúncio.

**Nota de dependência**: a instrução `ON CONFLICT` já entra em T015, porque é assim que
se insere nesta tabela. Esta fase é dona do que fazer **quando o conflito acontece**.

- [X] T025 [US2] Implementar o ramo de conflito em `internal/usecase/processar_pagamento.go` — estado terminal com anúncio feito confirma sem efeito; estado terminal com anúncio pendente republica a partir do que está gravado, sem tocar no adquirente (FR-014); `PROCESSANDO` devolve à fila
- [X] T026 [P] [US2] Testar o ramo de conflito em `internal/usecase/processar_pagamento_test.go` — reentrega em cada um dos cinco estados, verificando em quais há republicação e em quais não há, e que o adquirente nunca é chamado de novo
- [X] T027 [US2] Testar a corrida em `test/integration/idempotencia_test.go` — vinte entregas simultâneas da mesma reserva resultam em uma transação, uma chamada ao adquirente e um anúncio (SC-002)
- [X] T028 [US2] Testar a recuperação em `test/integration/anuncio_pendente_test.go` — interromper entre `Finalizar` e publicar, e verificar que a reentrega publica o resultado gravado sem nova cobrança e marca `resultado_anunciado` (SC-003)

**Checkpoint**: a garantia de dinheiro está provada com infraestrutura real.

---

## Phase 5: User Story 3 — Consultar o andamento do pagamento (P2)

**Goal**: a pessoa dona da reserva acompanha o desfecho; mais ninguém enxerga nada.

**Independent Test**: levar uma reserva a cada estado possível e verificar que a
consulta devolve o estado corrente, e que terceiro e reserva inexistente recebem
respostas indistinguíveis.

- [X] T029 [P] [US3] Implementar a validação stateless do token em `internal/adapter/http/auth.go` — JWKS, assinatura, emissor, público e validade, extraindo a claim `sub` (`research.md` D8)
- [X] T030 [P] [US3] Implementar `BuscarPorReserva` em `internal/adapter/postgres/transacoes.go`, com ausência distinguível de erro
- [X] T031 [US3] Implementar o caso de uso em `internal/usecase/consultar_pagamento.go`, com a guarda de dono devolvendo o mesmo resultado de ausência quando o `sub` não bate com `usuario_id` (FR-017)
- [X] T032 [US3] Implementar as rotas em `internal/adapter/http/handlers.go` e `rotas.go` — `GET /api/v1/pagamentos/reserva/{reserva_id}` e as duas sondas de saúde, com os `codigo` de erro de `contracts/erros.md` e sem expor `motivo_falha` nem `codigo_transacao_gateway`
- [X] T033 [P] [US3] Escrever o gerador de token de teste em `scripts/token-teste.sh`, assinando com a chave de teste do roteiro de `quickstart.md` §3
- [X] T034 [US3] Testar o contrato HTTP em `internal/adapter/http/handlers_test.go` sobre `httptest` — 200 para a dona em cada estado, 400 para UUID malformado, 401 sem token e com token inválido, 404 para terceiro **em cada um dos cinco estados** e para reserva inexistente, **comparando os dois corpos de 404 byte a byte** (FR-017, SC-007, `contracts/erros.md`)
- [X] T035 [P] [US3] Verificar em `internal/adapter/http/openapi_test.go` que os campos e os `enum` das respostas batem com `contracts/openapi.yaml`

**Checkpoint**: a jornada da pessoa está completa.

---

## Phase 6: User Story 4 — Absorver picos e sobreviver a falhas (P2)

**Goal**: nenhuma intenção perdida sob rajada; o teto de cobranças simultâneas respeitado;
ausência de resposta do adquirente tratada como o silêncio deliberado que a spec exige.

**Independent Test**: publicar uma rajada muito acima da vazão de cobrança e verificar que
nada se perde e que a concorrência nunca ultrapassa o teto; e que intenções que falham
por infraestrutura voltam a ser processadas.

- [X] T036 [US4] Aplicar `basic.qos` com o teto configurável e um conjunto de rotinas de mesmo tamanho em `internal/adapter/amqp/consumidor.go` (FR-019, `research.md` D6)
- [X] T037 [US4] Implementar o desfecho indeterminado em `internal/usecase/processar_pagamento.go` e no consumidor — prazo do adquirente estourado leva a `PENDENTE_VERIFICACAO`, sem publicação e sem nova cobrança, com `nack` sem reenfileirar (FR-022, `research.md` D4)
- [X] T038 [P] [US4] Testar o desfecho indeterminado em `internal/usecase/processar_pagamento_test.go` — nenhum anúncio é emitido, o estado é `PENDENTE_VERIFICACAO` e `resultado_anunciado` continua falso
- [X] T039 [US4] Testar a quarentena em `test/integration/quarentena_test.go` — anúncio inválido cai na fila morta sem criar transação; intenção que falha até o limite de entregas é encaminhada pelo broker; e toda transação que termina em `PENDENTE_VERIFICACAO` tem, ao mesmo tempo, a mensagem correspondente na fila morta e nenhum anúncio publicado (SC-006, SC-009)
- [X] T040 [US4] Testar a falha transitória em `test/integration/resiliencia_test.go` — banco indisponível durante o consumo devolve a mensagem à fila, nenhum resultado é anunciado, e o processamento se completa quando o banco volta
- [X] T041 [US4] Testar a rajada em `test/integration/vazao_test.go` — mil intenções em um minuto são todas processadas, o número de cobranças simultâneas nunca ultrapassa o teto, e consultas disparadas concorrentemente durante o pico respondem abaixo de um segundo em 95% dos casos (SC-004, SC-005)

**Checkpoint**: todos os requisitos funcionais estão cobertos.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T042 [P] Instrumentar o fluxo em `internal/adapter/amqp/consumidor.go` e `internal/platform/observability/` — `reserva_id` e `transacao_id` em todo registro, extração do `traceparent` dos cabeçalhos e span de consumo filho, mais as métricas de cobranças em andamento e de desfecho por categoria (FR-024)
- [X] T043 [P] Documentar o serviço em `README.md` — como subir, as variáveis de ambiente de T004 e a dependência de integração aberta com o estoque
- [X] T044 [P] Acrescentar ao `Makefile` os alvos que o roteiro usa (`publicar-reserva`, `publicar-rajada`, `espiar-evento`, `run-falha-apos-gravar`)
- [X] T045 Executar o roteiro de `quickstart.md` de ponta a ponta e registrar no relatório o que rodou e o que não rodou, conforme o princípio III
- [X] T046 Rodar `make lint`, `make test` e `make test-integration` e deixar os três verdes

---

## Dependencies

```text
Setup (T001-T003)
   └─> Foundational (T004-T014)   ⚠ bloqueia tudo
          ├─> US1 (T015-T024)  P1  🎯 MVP
          │      └─> US2 (T025-T028)  P1   depende do fluxo de US1
          ├─> US3 (T029-T035)  P2   independente de US1/US2/US4
          └─> US4 (T036-T041)  P2   depende do consumidor de US1 (T020)
                 └─> Polish (T042-T046)
```

**Ordem de histórias**: US1 → US2 → US3 → US4. US3 é a única verdadeiramente
independente: só precisa da Fase 2 e de linhas na tabela, que os testes podem
inserir direto. US2 e US4 dependem do fluxo de consumo que US1 constrói.

## Parallel Opportunities

| Bloco | Tarefas paralelizáveis |
|---|---|
| Setup | T002, T003 |
| Foundational | T005, T006, T008, T012, T013 (arquivos distintos) |
| US1 | T018, T021, T022, T023 |
| US2 | T026 (isolada; T027 e T028 disputam a infraestrutura de teste) |
| US3 | T029, T030, T033, T035 |
| US4 | T038 |
| Polish | T042, T043, T044 |

Os testes de integração (T024, T027, T028, T039, T040, T041) sobem contêineres e
**não** devem rodar em paralelo entre si sem isolamento de portas.

## Implementation Strategy

**MVP**: Fases 1 a 3 (T001–T024). Ao fim delas o serviço cobra uma reserva anunciada
e publica o desfecho — que é a razão de existir do serviço. Falta ainda a garantia de
cobrança única, então **não é entregável a usuário real**: T027 é o que separa o
demonstrável do confiável.

**Incremento seguinte**: Fase 4. É a menor fatia que torna o serviço apto a lidar com
dinheiro de verdade.

**Depois**: US3 e US4 podem ser feitas em qualquer ordem, ou em paralelo por pessoas
diferentes — não compartilham arquivo algum.

**Restrição de conclusão** (constituição, Fluxo de Desenvolvimento): a feature não pode
ser dada por concluída com T008, T021, T022, T026, T034, T035 ou T038 pendentes — são
os testes de domínio e de interface exposta exigidos pelo princípio II. E o estado de
cada tarefa MUST ser aferido no código, não nesta marcação: caixa marcada cujo efeito
não existe é divergência, e divergência é pergunta (princípio IV).

**Pendência conhecida que nenhuma tarefa fecha**: o `Servico-Estoque` ainda não publica
`valor_total` nem `forma_pagamento` em `reserva.criada` (`research.md` D1). T023 contorna
isso para validar o serviço; a integração real depende de trabalho no estoque, fora do
escopo desta feature.
