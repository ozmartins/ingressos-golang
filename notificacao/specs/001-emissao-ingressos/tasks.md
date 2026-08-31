---
description: "Task list for Emissão e Validação de Ingressos Digitais (Servico-Notificacao)"
---

# Tasks: Emissão e Validação de Ingressos Digitais (Servico-Notificacao)

**Input**: Design documents from `specs/001-emissao-ingressos/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: **OBRIGATÓRIOS**, não opcionais. O princípio II da constituição do workspace
exige teste automatizado do núcleo de domínio (invariantes, transições, condições de
erro) e de **toda operação de interface exposta** — síncrona ou por evento — cobrindo o
caminho de sucesso e cada categoria de erro declarada em
[contracts/erros.md](./contracts/erros.md). Testes de adaptador, infraestrutura e fiação
de composição seguem opcionais, por custo-benefício.

> **Nota sobre o template**: `notificacao/.specify/templates/tasks-template.md` ainda diz
> "Tests are OPTIONAL". Essa cópia local ficou para trás do template do workspace, que
> foi emendado quando a constituição v1.0.0 foi ratificada. Pela cláusula de Governance
> ("em conflito entre a constituição e um documento, a constituição vence e o documento
> MUST ser corrigido"), este arquivo segue a constituição. A correção do template está
> reportada ao mantenedor como mudança dedicada, não como efeito colateral desta feature.

**Organization**: tarefas agrupadas por user story, para que cada uma seja implementável
e testável de forma independente.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivo diferente, sem dependência pendente)
- **[Story]**: a qual user story a tarefa pertence (US1–US4)
- Todo caminho de arquivo é relativo a `notificacao/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: esqueleto do módulo e ferramental, alinhado aos serviços já existentes no workspace.

- [X] T001 Inicializar o módulo Go em `go.mod` com `module github.com/oseias/ingressos-golang/notificacao` e `go 1.27.0`, e a árvore de pastas de `plan.md` §Source Code
- [X] T002 [P] Configurar o linter em `.golangci.yml`, espelhando `../pagamento/.golangci.yml`
- [X] T003 [P] Criar `Makefile` com os alvos `build`, `test`, `test-integration`, `lint`, `migrate-up`, `migrate-down`, `run`, `publicar-pagamento`, `publicar-cru`, seguindo `../pagamento/Makefile`
- [X] T004 [P] Criar `docker-compose.yml` com PostgreSQL 16 e RabbitMQ (com plugin de management), portas alinhadas ao `quickstart.md`
- [X] T005 [P] Criar `Dockerfile` multi-estágio e `.dockerignore`, espelhando `../pagamento/`
- [X] T006 [P] Criar `.gitignore` ignorando `bin/` e artefatos locais

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: o que **toda** user story precisa: domínio do ingresso, código de acesso, esquema, topologia, servidor e credenciais.

**⚠️ CRÍTICO**: nenhuma user story pode começar antes desta fase terminar.

- [X] T007 Implementar a leitura e validação de ambiente em `internal/platform/config/config.go`, reunindo **todas** as chaves problemáticas em um único erro (research D11). Obrigatórias, sem padrão: `DATABASE_URL`, `AMQP_URL`, `JWKS_URL`, `JWT_ISSUER`, `JWT_AUDIENCE`, `INGRESSO_QR_SEGREDO`, `PORTARIA_API_KEY`. Com padrão: `PORTA_HTTP` (8080), `AMQP_EXCHANGE` (`cinema.eventos`), `AMQP_EXCHANGE_DLX` (`cinema.eventos.dlx`), `AMQP_FILA_PAGAMENTO_SUCESSO` (`notificacao.pagamento-sucesso`), `AMQP_PREFETCH` (10), `AMQP_LIMITE_ENTREGAS` (3), `NOTIFICADOR_MODO` (`enviar`), `NIVEL_LOG` (`info`) — as três numéricas ou enumeradas recusam valor inválido em vez de cair no padrão
- [X] T008 [P] Escrever o teste de configuração em `internal/platform/config/config_test.go`: ausência de `INGRESSO_QR_SEGREDO` impede subir, e o erro lista todas as chaves faltantes de uma vez (contracts/erros.md §5)
- [X] T009 [P] Implementar `internal/platform/observability/observability.go` com `slog` em JSON, OpenTelemetry e extração de `traceparent` de cabeçalhos AMQP, espelhando `../pagamento/internal/platform/observability/`
- [X] T010 [P] Implementar as sondas de vivacidade e prontidão em `internal/platform/health/health.go`
- [X] T011 Escrever a migração `migrations/000001_criar_ingressos.up.sql` com as duas tabelas, as restrições `CHECK` e o índice `ingressos_por_pessoa`, exatamente como em `data-model.md` §1 e §3, e a `.down.sql` correspondente
- [X] T012 [P] Implementar a conexão e o pool PostgreSQL em `internal/adapter/postgres/postgres.go` (pgx v5)
- [X] T013 [P] Implementar a topologia AMQP em `internal/adapter/amqp/topologia.go`: fila quórum, DLX, e `x-delivery-limit = AMQP_LIMITE_ENTREGAS - 1` com o comentário explicando a tradução tentativas→reentregas (research D5, contracts/eventos.md §2)
- [X] T014 Implementar o núcleo de domínio em `internal/domain/ingresso/ingresso.go`: tipo `Status` (`VALIDO`, `UTILIZADO`, `CANCELADO`), construtor `Nova`, transição `Utilizar`, e rejeição de qualquer saída de estado terminal (FR-019)
- [X] T015 [P] **[Princípio II]** Escrever o teste de domínio em `internal/domain/ingresso/ingresso_test.go`, sem banco, sem rede e com relógio injetado: `VALIDO`→`UTILIZADO` grava o instante; `UTILIZADO`→qualquer coisa é rejeitado; `CANCELADO`→qualquer coisa é rejeitado; a invariante "tem instante se e somente se está utilizado" vale em cada transição; e **a transição não altera `ReservaID`, `UsuarioID`, `CodigoQR` nem `CriadoEm`** (FR-020)
- [X] T016 Implementar a geração e a verificação do código de acesso em `internal/adapter/codigo/codigo.go`: formato `CIN1.<base64url(id)>.<base64url(HMAC-SHA256)>`, comparação com `crypto/subtle` (research D3, data-model.md §4)
- [X] T017 [P] **[Princípio II]** Escrever o teste do código de acesso em `internal/adapter/codigo/codigo_test.go`: ida e volta; assinatura adulterada recusada; prefixo errado recusado; número de partes errado recusado; segredo diferente recusado; **nenhuma** verificação toca no banco
- [X] T018 Declarar as portas em `internal/usecase/ports.go`: `Ingressos`, `Avisos`, `Notificador`, `Relogio`, `GeradorID`, com as assinaturas de `data-model.md` §5
- [X] T019 [P] Implementar os adaptadores triviais de `Relogio` e `GeradorID` em `internal/adapter/sistema/sistema.go` (research D9)
- [X] T020 Montar o servidor HTTP em `internal/adapter/http/servidor.go`: roteamento com `net/http`, respostas `application/problem+json` (RFC 9457) e as rotas `/health/live` e `/health/ready`
- [X] T021 [P] Implementar as duas credenciais em `internal/adapter/http/auth.go`: JWT por JWKS (espelhando `../pagamento/internal/adapter/http/auth.go`) e `X-API-Key` comparada em tempo constante (research D7)
- [X] T022 [P] **[Princípio II]** Escrever o teste de credenciais em `internal/adapter/http/auth_test.go`: token ausente, expirado, emissor errado, público errado e assinatura inválida → todos 401 idênticos; chave de portaria ausente e chave errada → 401 idênticos

**Checkpoint**: domínio testado, código de acesso testado, esquema aplicável, servidor de pé. As user stories podem começar.

---

## Phase 3: User Story 1 — Emitir o ingresso quando o pagamento é confirmado (P1) 🎯 MVP

**Goal**: um `pagamento.sucesso` vira exatamente um ingresso válido, com código de acesso, e reentrega não duplica.

**Independent Test**: publicar um anúncio válido e verificar que um ingresso é registrado com código único, estado válido e instante de criação; republicar o mesmo anúncio e verificar que nenhum segundo ingresso aparece.

**Escopo desta fase**: emissão apenas. O aviso à pessoa entra na US4 — até lá, a FR-016 fica deliberadamente descoberta, e é isso que torna esta fase um incremento e não meio caminho.

### Tests (escrever primeiro; devem falhar)

- [X] T023 [P] [US1] **[Princípio II]** Escrever o teste do caso de uso em `internal/usecase/emitir_ingresso_test.go` com portas falsas: anúncio válido emite; reserva já com ingresso não emite e é inerte; campo obrigatório ausente, UUID malformado e `pago_em` fora do RFC 3339 → quarentena sem retentativa; falha transitória do repositório → nova tentativa; campos extras desconhecidos são tolerados (contracts/erros.md §4, research D1)
- [X] T024 [P] [US1] **[Princípio II]** Escrever o teste de desfechos do consumo em `internal/adapter/amqp/consumidor_test.go`: cada linha da tabela de `contracts/eventos.md` §3 mapeada ao gesto AMQP correto (`Ack`, `Nack(requeue=false)`, `Nack(requeue=true)`)

### Implementation

- [X] T025 [US1] Implementar `CriarSeAusente` em `internal/adapter/postgres/ingressos.go` com `INSERT ... ON CONFLICT (reserva_id) DO NOTHING RETURNING *` (research D2)
- [X] T026 [US1] Implementar o caso de uso em `internal/usecase/emitir_ingresso.go`: validar os quatro campos obrigatórios → gerar id → gerar código → gravar → classificar o desfecho (emitido / inerte / quarentena / nova tentativa)
- [X] T027 [US1] Implementar o consumidor em `internal/adapter/amqp/consumidor.go`: `Qos` com prefetch, ack manual, JSON ilegível → `Nack(requeue=false)`, e o desfecho do caso de uso traduzido em gesto AMQP. Registrar em log estruturado, a cada entrega, o desfecho (emitido / inerte / quarentena / nova tentativa), o `reserva_id` e o `ingresso_id` quando houver (FR-021)
- [X] T028 [US1] Fiar a composição em `cmd/notificacao/main.go`: config → observabilidade → banco → topologia → portas → consumidor → servidor, com desligamento gracioso
- [X] T029 [P] [US1] Implementar o publicador manual em `cmd/publicar/main.go` com as flags `-reserva`, `-usuario` e `-cru`, para os cenários 1, 2 e 7 do `quickstart.md`
- [X] T030 [US1] **[Princípio II]** Escrever o teste de integração em `test/integration/emissao_test.go` (Testcontainers, Postgres e RabbitMQ reais): duas entregas simultâneas da mesma reserva produzem **um** ingresso (SC-001); anúncio malformado vai direto para a fila morta; anúncio válido com banco fora volta à fila e cai na fila morta ao esgotar o limite (FR-022, SC-008, SC-010)

**Checkpoint**: pagamento confirmado vira ingresso, de forma idempotente e resiliente. Cenários 1, 2 e 7 do quickstart passam.

---

## Phase 4: User Story 2 — Validar o ingresso na entrada da sala (P2)

**Goal**: a portaria apresenta o código e recebe veredito imediato; a primeira leitura autoriza e dá baixa, a segunda nega.

**Independent Test**: a partir de um ingresso emitido, apresentar seu código e verificar que a primeira apresentação autoriza e dá baixa e que a segunda nega informando reuso.

### Tests (escrever primeiro; devem falhar)

- [X] T031 [P] [US2] **[Princípio II]** Escrever o teste do caso de uso em `internal/usecase/validar_ingresso_test.go`: válido autoriza e carimba; já utilizado nega sem alterar o instante original; cancelado nega; assinatura inválida nega **sem consultar o repositório** (porta falsa que registra se foi chamada)
- [X] T032 [P] [US2] **[Princípio II]** Escrever o teste de contrato em `internal/adapter/http/validar_test.go` sobre `httptest`, uma asserção por linha de `contracts/erros.md` §3: 200, 409 (utilizado), 409 (cancelado), 404, 401 (chave ausente), 401 (chave errada), 401 (token no lugar da chave), 422. Incluir a asserção de que as três respostas 404 — malformado, assinatura inválida, inexistente — são **idênticas byte a byte**

### Implementation

- [X] T033 [US2] Implementar `Utilizar` e `BuscarPorID` em `internal/adapter/postgres/ingressos.go`: `UPDATE ... SET status='UTILIZADO', utilizado_em=$1 WHERE id=$2 AND status='VALIDO'`, com a leitura de motivo só quando zero linhas foram afetadas (research D4). O `SET` toca **apenas** `status` e `utilizado_em` — nenhuma outra coluna é escrita depois da emissão (FR-020)
- [X] T034 [US2] Implementar o caso de uso em `internal/usecase/validar_ingresso.go`: verificar a assinatura antes de tudo, tentar a baixa, e só então classificar a recusa
- [X] T035 [US2] Implementar o manipulador `POST /api/v1/ingressos/validar` em `internal/adapter/http/handlers.go`, protegido por `X-API-Key`, com os corpos de `contracts/openapi.yaml` (`valido: true/false`, não RFC 9457). Registrar em log estruturado o desfecho de cada validação (autorizada / reuso / cancelado / não encontrado) e o `ingresso_id` quando houver — **nunca** o `codigo_qr` (FR-021)
- [X] T036 [US2] **[Princípio II]** Escrever o teste de integração em `test/integration/validacao_test.go`: duas leituras simultâneas do mesmo código resultam em exatamente uma autorização (FR-011, SC-004); e, após a baixa, `reserva_id`, `usuario_id`, `codigo_qr` e `criado_em` da linha continuam **idênticos** aos gravados na emissão (FR-020)

**Checkpoint**: a portaria funciona e não autoriza duas vezes. Cenários 4 e 5 do quickstart passam.

---

## Phase 5: User Story 3 — Consultar os meus ingressos (P3)

**Goal**: a pessoa autenticada vê os próprios ingressos, ordenados, com filtro opcional por estado.

**Independent Test**: autenticar-se como pessoa com ingressos emitidos e verificar que a listagem devolve exatamente os dela, do mais recente para o mais antigo, que o filtro recorta e que nenhum ingresso de terceiro aparece.

### Tests (escrever primeiro; devem falhar)

- [X] T037 [P] [US3] **[Princípio II]** Escrever o teste de contrato em `internal/adapter/http/listagem_test.go` sobre `httptest`, uma asserção por linha de `contracts/erros.md` §2: 200 com ingressos ordenados do mais recente ao mais antigo; 200 com `[]` para pessoa sem ingressos; 200 com filtro válido; 400 com filtro desconhecido; 401 sem token, com token expirado e com chave de portaria no lugar do token; e a asserção de que ingresso de outra pessoa **não** aparece (FR-014)

### Implementation

- [X] T038 [US3] Implementar `ListarPorUsuario` em `internal/adapter/postgres/ingressos.go` com `WHERE usuario_id = $1`, filtro opcional de estado e `ORDER BY criado_em DESC, id DESC` (research D8)
- [X] T039 [US3] Implementar o caso de uso em `internal/usecase/listar_ingressos.go`, recusando estado não reconhecido como pedido inválido em vez de ignorar o filtro (FR-024)
- [X] T040 [US3] Implementar o manipulador `GET /api/v1/ingressos/meus-ingressos` em `internal/adapter/http/handlers.go`, recortado pelo `sub` do token
- [X] T041 [P] [US3] Criar `scripts/token-teste.sh`, que assina um token de teste para o `quickstart.md`, espelhando `../pagamento/scripts/token-teste.sh`

**Checkpoint**: a pessoa alcança os próprios ingressos. Cenário 3 do quickstart passa.

---

## Phase 6: User Story 4 — Registrar o aviso de confirmação (P4)

**Goal**: cada ingresso emitido dispara um aviso e deixa registro do desfecho; a falha do aviso não derruba a emissão nem reprocessa a mensagem.

**Independent Test**: emitir um ingresso com o canal operante e verificar o registro `ENVIADO`; repetir com o canal falhando e verificar que o ingresso continua válido, que fica um registro `FALHA` com detalhe, e que a mensagem foi confirmada.

### Tests (escrever primeiro; devem falhar)

- [X] T042 [P] [US4] **[Princípio II]** Escrever o teste do aviso em `internal/usecase/emitir_ingresso_test.go` (acrescentando ao arquivo de T023): canal operante grava registro `ENVIADO`; **canal com erro grava `FALHA` com detalhe, mantém o ingresso válido e o desfecho continua sendo `Ack`** (FR-018, FR-025); reserva já com ingresso **não** dispara aviso novo (research D6)

### Implementation

- [X] T043 [P] [US4] Implementar o domínio do registro em `internal/domain/aviso/aviso.go`: canal, desfecho e a exigência de detalhe quando o desfecho é falha
- [X] T044 [P] [US4] Implementar o adaptador de aviso em `internal/adapter/notificador/simulado/simulado.go`, com modo de falha controlável por `NOTIFICADOR_MODO` para o cenário 6 do quickstart (research D6)
- [X] T045 [US4] Implementar `Registrar` em `internal/adapter/postgres/avisos.go`
- [X] T046 [US4] Fiar o disparo em `internal/usecase/emitir_ingresso.go`, na ordem fixa gravar ingresso → avisar → registrar → confirmar, com o erro do notificador capturado e **nunca** propagado. Registrar em log estruturado o desfecho do disparo (enviado / falha) e o `ingresso_id` (FR-021)
- [X] T047 [US4] **[Princípio II]** Escrever o teste de integração em `test/integration/aviso_test.go`: com o notificador em modo de falha, o ingresso é emitido e válido, o registro é `FALHA` com detalhe, e a **fila principal fica vazia** — a mensagem foi confirmada, não reprocessada (SC-007)

**Checkpoint**: as quatro user stories funcionam. Cenário 6 do quickstart passa.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T048 [P] Escrever o teste de contrato servido em `internal/adapter/http/openapi_test.go`, conferindo que as rotas e os códigos de `contracts/openapi.yaml` batem com o que o servidor expõe, espelhando `../pagamento/internal/adapter/http/openapi_test.go`
- [X] T049 [P] Escrever o teste de auditoria em `internal/adapter/http/log_test.go` e `internal/adapter/amqp/consumidor_test.go`, cobrindo as **duas** metades da FR-021: (a) emissão, validação e disparo de aviso registram o desfecho e o `ingresso_id` em log estruturado; (b) o `codigo_qr` **não** aparece em log, atributo de rastro nem mensagem de erro, em nenhum caminho — sucesso, recusa ou falha (research D13)
- [X] T050 [P] Escrever `README.md` com o propósito do serviço, as variáveis de ambiente e como rodar
- [X] T051 [P] Escrever a medição de listagem em `test/integration/vazao_listagem_test.go`: semear 200 ingressos para uma mesma pessoa e afirmar que a listagem completa responde em menos de 2 s no percentil 95 (SC-009)
- [X] T052 Escrever a medição de portaria e de emissão em `test/integration/vazao_test.go`: rajada de leituras simultâneas afirmando veredito em menos de 1 s no percentil 99 (SC-003), e intervalo entre publicar o anúncio e o ingresso ficar consultável menor que 5 s no percentil 95 (SC-002). Fixar no teste o volume que representa o horário de pico, hoje não quantificado na spec, e registrar o valor escolhido em comentário
- [ ] T053 Rodar `make lint` e `make test -race` pelos alvos de `Makefile` e deixar ambos verdes
      > **Parcial em 2026-08-30**: `make test -race` está verde, e a bateria de
      > integração também. `make lint` **não foi executado**: `golangci-lint` não está
      > instalado nesta máquina. `go vet ./...`, `go vet -tags=integration ./test/...` e
      > `gofmt -l` estão limpos, mas isso não substitui o linter — a tarefa fica aberta
      > até `make lint` rodar de fato.
- [X] T054 Executar o roteiro completo de `quickstart.md`, cenários 1 a 8, e **remover do arquivo o aviso de que nada foi executado**, substituindo-o pelo registro do que foi verificado em execução e do que não foi (princípio III)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: sem dependências
- **Foundational (Phase 2)**: depende do Setup — **bloqueia todas as user stories**
- **US1 (Phase 3)**: depende da Phase 2. É o MVP
- **US2 (Phase 4)**: depende da Phase 2. Precisa de ingressos para validar, mas o teste pode criá-los direto no banco — independentemente testável sem a US1 estar pronta
- **US3 (Phase 5)**: depende da Phase 2, pela mesma razão da US2
- **US4 (Phase 6)**: depende da **Phase 3**, e é a única com dependência real entre stories — o aviso é disparado de dentro do caso de uso de emissão (research D6)
- **Polish (Phase 7)**: depende das stories desejadas estarem completas

### Within Each User Story

- Os testes são escritos **antes** e devem falhar antes da implementação
- Adaptador de persistência → caso de uso → adaptador de entrada (HTTP ou AMQP)
- Teste de integração por último, quando há o que integrar

### Parallel Opportunities

- Setup: T002–T006 em paralelo
- Foundational: T008/T009/T010 juntos; T012/T013 juntos; T015 e T017 depois de T014 e T016; T021/T022 juntos
- US1: T023 e T024 juntos; T029 em paralelo com T025–T028
- US2: T031 e T032 juntos
- US4: T043 e T044 juntos
- Polish: T048, T049, T050 e T051 juntos; T052 depois de T051 (mesmo pacote de integração)
- **Entre stories**: com a Phase 2 pronta, US1, US2 e US3 podem ser tocadas por pessoas diferentes ao mesmo tempo. US4 espera a US1

### Parallel Example: User Story 1

```bash
# Os dois testes primeiro, juntos:
Task: "Teste do caso de uso em internal/usecase/emitir_ingresso_test.go"
Task: "Teste de desfechos do consumo em internal/adapter/amqp/consumidor_test.go"

# Depois, a implementação em cadeia, com o publicador em paralelo:
Task: "CriarSeAusente em internal/adapter/postgres/ingressos.go"
Task: "Publicador manual em cmd/publicar/main.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1: Setup
2. Phase 2: Foundational — **crítica, bloqueia tudo**
3. Phase 3: US1
4. **PARE E VALIDE**: cenários 1, 2 e 7 do quickstart, mais `make test-integration`
5. Nesse ponto o serviço já emite ingressos idempotentes e resilientes — sem aviso e sem leitura, mas com a peça que nenhuma outra substitui

### Incremental Delivery

1. Setup + Foundational → base pronta
2. + US1 → pagamento confirmado vira ingresso (**MVP**)
3. + US2 → a portaria passa a deixar entrar
4. + US3 → a pessoa passa a alcançar o próprio ingresso
5. + US4 → a pessoa passa a ser avisada, com trilha de reenvio
6. + Polish → contrato conferido, log auditado, quickstart executado

### Independent Test Criteria

| Story | Critério de teste independente |
|---|---|
| US1 (P1) | Anúncio válido → um ingresso com código único, estado válido e instante; reanúncio → nenhum segundo ingresso |
| US2 (P2) | Ingresso emitido → primeira leitura autoriza e dá baixa; segunda nega por reuso |
| US3 (P3) | Pessoa com ingressos → recebe só os dela, do mais recente ao mais antigo, com o filtro recortando |
| US4 (P4) | Canal operante → registro `ENVIADO`; canal falhando → ingresso válido, registro `FALHA` com detalhe, mensagem confirmada |
