---

description: "Task list for feature implementation"
---

# Tasks: Bloqueio, Confirmação e Liberação de Poltronas (Servico-Estoque)

**Input**: Design documents from `specs/001-estoque-bloqueio-poltronas/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: incluídos. `research.md` (D9) define uma estratégia de testes em quatro camadas como decisão de projeto, cada história da spec declara um "Independent Test", e os critérios SC-002, SC-004, SC-006, SC-008, SC-011 e SC-012 só são verificáveis por teste automatizado com infraestrutura real. As tarefas de teste fazem parte do escopo, não são opcionais aqui.

**Organization**: agrupadas por história de usuário, para que cada uma seja implementável, testável e demonstrável de forma independente.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência pendente)
- **[Story]**: história a que a tarefa pertence (US1..US5)
- Caminhos de arquivo são relativos à raiz do módulo (`estoque/`)

## Path Conventions

Módulo Go único com arquitetura hexagonal, conforme `plan.md`: `cmd/`, `internal/domain/`, `internal/usecase/`, `internal/adapter/`, `internal/platform/`, `migrations/`, `gen/`, `test/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: esqueleto do módulo, dependências e ferramental

- [ ] T001 Inicializar o módulo Go em `go.mod` (`github.com/oseias/ingressos-golang/estoque`, Go 1.25) e criar a árvore de diretórios de `plan.md` com um `.gitkeep` por pacote ainda vazio
- [ ] T002 [P] Declarar as dependências de `research.md` (D8) em `go.mod`: `grpc`, `protobuf`, `pgx/v5`, `amqp091-go`, `go-redis/v9`, `golang-migrate/v4`, `otel` (SDK + OTLP + otelgrpc), `testcontainers-go`
- [ ] T003 [P] Configurar `.golangci.yml` com `govet`, `staticcheck`, `errcheck`, `revive` e o analisador de imports que impede `internal/domain` e `internal/usecase` de importarem `internal/adapter`
- [ ] T004 [P] Criar `Makefile` com alvos `build`, `test`, `test-integration`, `lint`, `proto`, `migrate-up`, `certs`, `publicar-sessao`, `publicar-pagamento`, conforme `quickstart.md`
- [ ] T005 [P] Copiar o contrato para `proto/estoque.proto` a partir de `specs/001-estoque-bloqueio-poltronas/contracts/estoque.proto` e configurar `buf.yaml` e `buf.gen.yaml`
- [ ] T006 Gerar o código do contrato em `gen/pb/estoque/` a partir de `proto/estoque.proto` (alvo `proto` do Makefile)
- [ ] T007 [P] Escrever `docker-compose.yml` com PostgreSQL 16, `migrate`, Redis 7, RabbitMQ (com management) e o serviço `estoque`, na ordem de dependência de `quickstart.md`
- [ ] T008 [P] Escrever `Dockerfile` multi-estágio com imagem final distroless e usuário não-root
- [ ] T009 [P] Criar o alvo `make certs` em `scripts/gerar-certs.sh`, gerando CA de desenvolvimento e pares servidor/cliente em `certs/`, e adicionar `certs/` ao `.gitignore`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: infraestrutura transversal exigida por todas as histórias. Nenhuma história pode começar antes desta fase terminar

- [ ] T010 Escrever a migração `migrations/000001_criar_esquema.up.sql` (+ `.down.sql`) com `poltronas`, `reservas`, `reserva_poltronas`, `outbox_eventos` (incluindo a coluna `trace_context`) e `mensagens_processadas`, com todos os `CHECK` e chaves de `data-model.md`
- [ ] T011 Escrever a migração `migrations/000002_criar_indices.up.sql` (+ `.down.sql`) com `idx_poltronas_sessao`, `idx_reservas_expiracao` (parcial), `idx_reserva_poltronas_poltrona` e `idx_outbox_pendentes` (parcial)
- [ ] T012 [P] Implementar a leitura e validação de configuração em `internal/platform/config/config.go` com todas as chaves de `research.md` (D10), recusando subir com variável ausente ou malformada
- [ ] T013 [P] Escrever os testes de configuração em `internal/platform/config/config_test.go`: valor padrão aplicado, variável obrigatória ausente falha, valor malformado falha
- [ ] T014 [P] Implementar erros de domínio em `internal/domain/shared/erros.go` (`ErrSolicitacaoInvalida`, `ErrLimiteExcedido`, `ErrSessaoNaoProvisionada`, `ErrPoltronaInexistente`, `ErrPoltronasIndisponiveis`, `ErrSessaoDesconhecida`, `ErrDependenciaIndisponivel`)
- [ ] T015 [P] Implementar a porta `Relogio` em `internal/domain/shared/relogio.go` com implementação real e uma controlável para teste
- [ ] T016 [P] Implementar logs estruturados, métricas e rastreamento em `internal/platform/observability/observability.go` (`slog` JSON + OTel, com desligamento gracioso)
- [ ] T017 Implementar o pool PostgreSQL e o auxiliar de transação em `internal/adapter/postgres/postgres.go` (`pgxpool`, `EmTransacao(ctx, fn)` com rollback em erro)
- [ ] T018 Implementar o servidor gRPC em `internal/adapter/grpc/servidor.go` com mTLS (`RequireAndVerifyClientCert`), interceptores de log, métrica e rastreamento, e `TLS_CLIENT_AUTH=require|off`, registrando de forma auditável toda recusa de handshake via `tls.Config.VerifyConnection` — o interceptor gRPC nunca vê essa falha — sem gravar material criptográfico (FR-039)
- [ ] T019 Implementar o mapeamento de erros de domínio para status gRPC em `internal/adapter/grpc/erros.go`, exatamente conforme `contracts/erros.md`, incluindo `ErrorInfo.reason` e `metadata["limite"]`
- [ ] T020 Implementar a conexão AMQP e a declaração idempotente de topologia em `internal/adapter/amqp/topologia.go` (exchange, DLX, 3 filas, 3 DLQs), recusando subir se não puder ser garantida
- [ ] T021 Implementar o laço genérico de consumo em `internal/adapter/amqp/consumidor.go`: prefetch configurável, ack manual após commit, requeue com atraso em falha transitória, envio à DLQ em falha definitiva, extração do contexto de rastreamento dos headers `traceparent`/`tracestate` abrindo o span de consumo como filho dele (FR-044), registro estruturado por mensagem (correlação, fila, `message_id`, desfecho, duração — FR-042) e métricas por fila (processado, ignorado por idempotência, requeue, DLQ — FR-043)
- [ ] T022 Implementar o registro de idempotência em `internal/adapter/postgres/mensagens.go` (`RegistrarProcessada(tx, fila, messageID) (novo bool, err)`) usando `ON CONFLICT DO NOTHING`
- [ ] T023 [P] Implementar os indicadores de saúde em `internal/platform/health/health.go` (liveness; readiness reprova sem PostgreSQL, reporta Redis como degradado) e servi-los na porta de administração
- [ ] T024 Compor o binário em `cmd/estoque/main.go`: configuração → observabilidade → adaptadores → casos de uso → servidor gRPC, consumidores e rotinas de fundo, com encerramento gracioso
- [ ] T025 [P] Criar o arcabouço de testes de contrato em `test/contract/harness_test.go` (servidor real sobre `bufconn`, com e sem certificado de cliente)
- [ ] T026 [P] Criar o arcabouço de testes de integração em `test/integration/main_test.go` (Testcontainers de PostgreSQL, Redis e RabbitMQ, migrações aplicadas, limpeza entre testes)

**Checkpoint**: infraestrutura pronta — as histórias podem ser implementadas em paralelo a partir daqui

---

## Phase 3: User Story 1 — Bloquear poltronas de uma sessão (Priority: P1) 🎯 MVP

**Goal**: conceder ou recusar, de forma atômica e em menos de 100 ms, o bloqueio de um conjunto de poltronas para uma pessoa, criando uma reserva pendente de 10 minutos e anunciando o fato.

**Independent Test**: provisionar a matriz de uma sessão (via carga direta, enquanto US5 não existe), solicitar o bloqueio de um subconjunto e verificar que a reserva é criada com prazo, que as poltronas ficam `RESERVADA` e que uma segunda solicitação sobre qualquer uma delas é recusada.

### Testes da história

- [ ] T027 [P] [US1] Testes de domínio em `internal/domain/poltrona/poltrona_test.go`: transições válidas e recusadas a partir de cada estado
- [ ] T028 [P] [US1] Testes de domínio em `internal/domain/reserva/reserva_test.go`: criação com prazo, `Expirou()`, imutabilidade de estado final
- [ ] T029 [P] [US1] Testes do objeto de valor em `internal/domain/reserva/solicitacao_test.go`: lista vazia, rótulos repetidos, acima do limite, usuário ausente, rótulo malformado
- [ ] T030 [P] [US1] Testes do caso de uso em `internal/usecase/bloquear_poltronas_test.go` com portas falsas: concessão, indisponibilidade, sessão não provisionada, rótulo inexistente, falha do repositório
- [ ] T031 [P] [US1] Teste de contrato em `test/contract/bloquear_test.go`: cada linha da tabela de `contracts/erros.md` para `BloquearPoltronas`, incluindo `sucesso=false` com `motivo=POLTRONAS_INDISPONIVEIS`, e que uma solicitação com `usuario_id` arbitrário e sem qualquer token de pessoa usuária é aceita quando o certificado do chamador é válido (FR-038 — este serviço não valida credencial de pessoa)
- [ ] T032 [P] [US1] Teste de integração de concorrência em `test/integration/concorrencia_test.go`: 100 solicitações paralelas sobre o mesmo conjunto, exatamente uma concedida, nenhuma poltrona em duas reservas (SC-002)
- [ ] T033 [P] [US1] Teste de integração em `test/integration/bloqueio_test.go`: concessão grava reserva, vínculos, poltronas `RESERVADA` e linha na caixa de saída, tudo na mesma transação
- [ ] T034 [P] [US1] Teste de integração em `test/integration/outbox_test.go`: broker derrubado no instante da concessão; o bloqueio permanece válido e o evento é entregue depois (SC-005)
- [ ] T035 [P] [US1] Teste de integração em `test/integration/indisponibilidade_test.go`: PostgreSQL fora do ar faz toda solicitação ser recusada com `UNAVAILABLE` e nenhuma reserva é criada (SC-012)

### Implementação da história

- [ ] T036 [P] [US1] Implementar a entidade `Poltrona` e o tipo `Rotulo` em `internal/domain/poltrona/poltrona.go` (estados, tipos, transições, formatação e leitura do rótulo)
- [ ] T037 [P] [US1] Implementar a entidade `Reserva` em `internal/domain/reserva/reserva.go` (estados, prazo, `PodeConfirmar`, `PodeCancelar`, `Expirou`)
- [ ] T038 [US1] Implementar o objeto de valor `SolicitacaoBloqueio` em `internal/domain/reserva/solicitacao.go` com todas as validações de FR-003 e FR-004, recebendo o limite configurado
- [ ] T039 [US1] Declarar as portas em `internal/usecase/ports.go`: `RepositorioPoltronas`, `RepositorioReservas`, `CaixaDeSaida`, `IndiceDePrazo`, `Relogio`
- [ ] T040 [US1] Implementar o caso de uso em `internal/usecase/bloquear_poltronas.go`: validar antes da transação, delegar o bloqueio atômico, montar a resposta com `reserva_id` e `expira_em`, sem esperar a publicação (FR-009)
- [ ] T041 [US1] Implementar a transação de bloqueio em `internal/adapter/postgres/bloqueio.go`: `SELECT ... FOR UPDATE NOWAIT` ordenado por rótulo, verificações, `INSERT` de reserva e vínculos, `UPDATE` das poltronas e `INSERT` na caixa de saída com `atualizado_em = now()` — exatamente o protocolo de `data-model.md` §8. Tarefa de maior risco do plano: exige revisão dedicada linha a linha contra aquele protocolo
- [ ] T042 [P] [US1] Implementar o repositório de poltronas em `internal/adapter/postgres/poltronas.go` (leitura por sessão e por rótulos)
- [ ] T043 [P] [US1] Implementar o repositório de reservas em `internal/adapter/postgres/reservas.go` (leitura por id, transições guardadas por `status='PENDENTE'`)
- [ ] T044 [US1] Implementar a gravação na caixa de saída em `internal/adapter/postgres/outbox.go` (`Enfileirar(tx, evento)` capturando o contexto de rastreamento corrente em `trace_context`, e `PendentesParaPublicar` com `FOR UPDATE SKIP LOCKED`)
- [ ] T045 [US1] Implementar o publicador da caixa de saída em `internal/adapter/amqp/publicador.go`: laço com *publisher confirms*, marcação de `publicado_em`, incremento de tentativas, recuo exponencial e reinjeção de `traceparent`/`tracestate` nos headers AMQP a partir de `trace_context` (FR-044)
- [ ] T046 [US1] Montar o payload de `reserva.criada` em `internal/adapter/amqp/eventos.go`, byte a byte conforme `contracts/eventos.md`, com `message_id = reserva_id`
- [ ] T047 [US1] Implementar o manipulador gRPC `BloquearPoltronas` em `internal/adapter/grpc/bloquear.go`, traduzindo erros pelo mapeamento de T019
- [ ] T048 [US1] Registrar as métricas do bloqueio em `internal/adapter/grpc/bloquear.go` (volume, latência e desfecho: concedido, indisponível, inválido, falha) conforme FR-043
- [ ] T049 [US1] Ligar caso de uso, repositórios, caixa de saída e publicador em `cmd/estoque/main.go`

**Checkpoint**: o serviço já concede e recusa bloqueios corretamente e anuncia `reserva.criada` — MVP demonstrável

---

## Phase 4: User Story 2 — Confirmar a reserva quando o pagamento é aprovado (Priority: P1)

**Goal**: reagir a `pagamento.sucesso` tornando a posse definitiva — reserva `CONFIRMADA`, poltronas `OCUPADA` — de forma idempotente.

**Independent Test**: criar uma reserva pendente, anunciar o sucesso do pagamento e verificar que a reserva fica confirmada, que as poltronas ficam ocupadas, que a expiração não as libera e que reprocessar o mesmo anúncio não altera nada.

### Testes da história

- [ ] T050 [P] [US2] Testes do caso de uso em `internal/usecase/confirmar_reserva_test.go`: reserva pendente confirma; reserva já finalizada é ignorada; reserva inexistente é ignorada e registrada
- [ ] T051 [P] [US2] Teste de integração em `test/integration/confirmacao_test.go`: efeito completo em uma transação e poltronas `OCUPADA`
- [ ] T052 [P] [US2] Teste de integração em `test/integration/idempotencia_test.go`: entrega duplicada de `pagamento.sucesso` produz o mesmo estado final e é confirmada normalmente (SC-004)
- [ ] T053 [P] [US2] Teste de integração em `test/integration/dlq_test.go`: JSON inválido e `reserva_id` desconhecido vão para a DLQ sem travar o consumo das demais mensagens (FR-023)

### Implementação da história

- [ ] T054 [US2] Implementar o caso de uso em `internal/usecase/confirmar_reserva.go`: guarda de idempotência, transição `PENDENTE → CONFIRMADA` e poltronas para `OCUPADA` na mesma transação (FR-015)
- [ ] T055 [US2] Implementar a transição de confirmação em `internal/adapter/postgres/reservas.go` (`UPDATE ... WHERE status='PENDENTE'` + `UPDATE` das poltronas vinculadas para `OCUPADA` com `atualizado_em = now()`, verificando as linhas afetadas)
- [ ] T056 [US2] Implementar o consumidor de `estoque.pagamento-sucesso` em `internal/adapter/amqp/consumidor_pagamento_sucesso.go` sobre o laço genérico de T021
- [ ] T057 [US2] Registrar auditoria de divergência em `internal/usecase/confirmar_reserva.go` quando a reserva já estiver finalizada ou não existir (FR-022)
- [ ] T058 [US2] Ligar o consumidor em `cmd/estoque/main.go`

**Checkpoint**: o ciclo bloqueio → pagamento aprovado → poltrona vendida funciona ponta a ponta

---

## Phase 5: User Story 3 — Liberar poltronas quando o pagamento falha (Priority: P1)

**Goal**: reagir a `pagamento.falhou` cancelando a reserva e devolvendo as poltronas ao estoque imediatamente.

**Independent Test**: criar uma reserva pendente, anunciar a falha do pagamento e verificar que a reserva fica cancelada, que as poltronas voltam a livres, que outra pessoa consegue bloqueá-las em seguida e que reprocessar o anúncio não produz efeito adicional.

### Testes da história

- [ ] T059 [P] [US3] Testes do caso de uso em `internal/usecase/cancelar_reserva_test.go`: cancelamento de pendente; reserva confirmada permanece `OCUPADA` e a divergência é registrada; reserva já cancelada é ignorada
- [ ] T060 [P] [US3] Teste de integração em `test/integration/cancelamento_test.go`: poltronas voltam a `LIVRE`, índice de prazo é liberado e outra solicitação as obtém em seguida
- [ ] T061 [P] [US3] Teste de integração em `test/integration/ordem_eventos_test.go`: aprovação e recusa para a mesma reserva, em ambas as ordens — prevalece o primeiro desfecho aplicado

### Implementação da história

- [ ] T062 [US3] Implementar o caso de uso em `internal/usecase/cancelar_reserva.go`: transição `PENDENTE → CANCELADA`, poltronas para `LIVRE` e liberação do índice de prazo, tudo guardado por idempotência
- [ ] T063 [US3] Implementar a transição de cancelamento em `internal/adapter/postgres/reservas.go` (reserva para `CANCELADA` e poltronas vinculadas de volta para `LIVRE` com `atualizado_em = now()`)
- [ ] T064 [US3] Implementar o consumidor de `estoque.pagamento-falhou` em `internal/adapter/amqp/consumidor_pagamento_falhou.go`
- [ ] T065 [US3] Ligar o consumidor em `cmd/estoque/main.go`

**Checkpoint**: o ciclo de vida da reserva está fechado nos dois desfechos de pagamento

---

## Phase 6: User Story 4 — Liberar automaticamente reservas não pagas (Priority: P2)

**Goal**: invalidar sozinho, em até 30 segundos do vencimento, toda reserva pendente que passou de 10 minutos, devolvendo as poltronas ao estoque.

**Independent Test**: criar uma reserva, avançar além do prazo sem anunciar desfecho e verificar que a reserva consta como expirada, que as poltronas voltam a livres e que podem ser bloqueadas de novo.

### Testes da história

- [ ] T066 [P] [US4] Testes do caso de uso em `internal/usecase/expirar_reservas_test.go` com relógio controlado: expira só o que venceu, ignora confirmadas, é idempotente
- [ ] T067 [P] [US4] Teste de integração em `test/integration/expiracao_test.go`: reserva vence e as poltronas voltam a `LIVRE` sem ação externa (SC-003)
- [ ] T068 [P] [US4] Teste de integração em `test/integration/expiracao_recuperacao_test.go`: serviço parado além do prazo de várias reservas; todas invalidadas no retorno (SC-008)
- [ ] T069 [P] [US4] Teste de integração em `test/integration/expiracao_sem_redis_test.go`: com o Redis fora, a liberação continua acontecendo pela varredura (D2/D4)
- [ ] T070 [P] [US4] Teste de integração em `test/integration/corrida_expiracao_test.go`: expiração e `pagamento.sucesso` disputando a mesma reserva resultam em um único estado final (FR-014)

### Implementação da história

- [ ] T071 [US4] Implementar o caso de uso em `internal/usecase/expirar_reservas.go`: `expirarReserva(id)` idempotente e varredura em lote, ambos guardados por `status='PENDENTE'`
- [ ] T072 [US4] Implementar a varredura em `internal/adapter/postgres/expiracao.go` com `UPDATE ... WHERE status='PENDENTE' AND expira_em < now() RETURNING id`, protegida por *advisory lock* para uma instância por vez
- [ ] T073 [US4] Implementar a rotina periódica em `internal/adapter/postgres/expiracao.go` com intervalo configurável e encerramento gracioso
- [ ] T074 [P] [US4] Implementar o índice de prazo em `internal/adapter/redis/prazo.go`: gravar `reserva:{id}` com TTL na concessão, apagar na finalização
- [ ] T075 [US4] Implementar a escuta de chave expirada em `internal/adapter/redis/expiracao.go`, disparando o mesmo `expirarReserva` da varredura, com reconexão automática
- [ ] T076 [US4] Ligar o índice de prazo ao bloqueio em `internal/adapter/postgres/bloqueio.go` (T041) e ao cancelamento em `internal/usecase/cancelar_reserva.go` (T062), tratando falha do Redis como degradação registrada, nunca como erro do bloqueio
- [ ] T077 [US4] Ligar varredura e escuta em `cmd/estoque/main.go`

**Checkpoint**: poltronas nunca ficam presas por abandono de compra

---

## Phase 7: User Story 5 — Provisionar a matriz e consultar o mapa (Priority: P3)

**Goal**: criar automaticamente a matriz de poltronas ao receber `sessao.criada` e permitir a consulta do estado atual de uma sessão pelo canal síncrono.

**Independent Test**: anunciar a criação de uma sessão com layout e verificar que todas as poltronas existem em `LIVRE` com fileira, número e tipo corretos, que reanunciar não duplica nem reinicia estados, e que o mapa é consultável.

### Testes da história

- [ ] T078 [P] [US5] Testes do caso de uso em `internal/usecase/provisionar_sessao_test.go`: provisionamento completo; layout com fileira e número repetidos é recusado inteiro; tipo desconhecido invalida a mensagem
- [ ] T079 [P] [US5] Testes do caso de uso em `internal/usecase/consultar_mapa_test.go`: mapa ordenado por fileira e número; sessão desconhecida distinguível de sessão sem poltronas
- [ ] T080 [P] [US5] Teste de contrato em `test/contract/consultar_mapa_test.go`: `NOT_FOUND` para sessão desconhecida, resposta completa para sessão provisionada, recusa sem certificado de cliente
- [ ] T081 [P] [US5] Teste de integração em `test/integration/provisionamento_test.go`: reentrega de `sessao.criada` não duplica poltronas nem reinicia uma poltrona já `RESERVADA` (SC-004), que a matriz fica disponível para bloqueio (SC-015) e que uma consulta ao mapa concorrente a um bloqueio em andamento devolve um retrato coerente (FR-030)
- [ ] T082 [P] [US5] Teste de integração em `test/integration/sessao_nao_provisionada_test.go`: bloqueio antes do provisionamento é recusado com `FAILED_PRECONDITION` e passa a ser aceito depois (FR-036)

### Implementação da história

- [ ] T083 [US5] Implementar a derivação do identificador em `internal/domain/poltrona/identidade.go` (UUID v5 de `sessao_id | fileira | numero`, rótulo `fileira || numero`), conforme `research.md` D6
- [ ] T084 [US5] Implementar o caso de uso em `internal/usecase/provisionar_sessao.go`: validar o layout inteiro antes de gravar, provisionar de forma indivisível, idempotente por `sessao_id`
- [ ] T085 [US5] Implementar a inserção em lote em `internal/adapter/postgres/poltronas.go` (`INSERT ... ON CONFLICT (sessao_id, fileira, numero) DO NOTHING`, em uma transação)
- [ ] T086 [US5] Implementar o consumidor de `estoque.sessao-criada` em `internal/adapter/amqp/consumidor_sessao_criada.go`
- [ ] T087 [US5] Implementar o caso de uso em `internal/usecase/consultar_mapa.go` (leitura sem alteração de estado, sessão desconhecida distinguível)
- [ ] T088 [US5] Implementar o manipulador gRPC `ConsultarMapaPoltronas` em `internal/adapter/grpc/consultar_mapa.go`
- [ ] T089 [US5] Ligar consumidor e manipulador em `cmd/estoque/main.go`

**Checkpoint**: o serviço se abastece sozinho a partir do catálogo e expõe o mapa da sessão

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: fechar requisitos transversais, verificar os critérios de sucesso restantes e deixar o serviço operável

- [ ] T090 [P] Escrever o teste de invariante em `test/integration/invariante_test.go`: 1.000 ciclos aleatórios de bloqueio seguidos de aprovação, recusa ou abandono; ao final toda poltrona está `LIVRE` ou `OCUPADA` e toda reserva em um único estado final (SC-006)
- [ ] T091 [P] Escrever o teste de desempenho em `test/integration/desempenho_test.go`: p99 do bloqueio abaixo de 100 ms em sala de 500 lugares (SC-001) e p99 da consulta do mapa abaixo de 200 ms (SC-013)
- [ ] T092 [P] Escrever o teste de segurança em `test/contract/mtls_test.go`: chamador sem certificado, com certificado expirado e de CA desconhecida são recusados antes de qualquer alteração de estado, e cada recusa gera registro auditável com o *subject* apresentado (SC-011, FR-039)
- [ ] T093 [P] Escrever o teste de correlação em `test/integration/rastreamento_test.go`: uma solicitação é reconstituível do bloqueio ao desfecho de pagamento por um único identificador (SC-009)
- [ ] T094 [P] Escrever o teste de largada em `test/integration/configuracao_test.go`: o processo sobe apenas com configuração externa e recusa subir com variável obrigatória ausente (SC-010)
- [ ] T095 [P] Verificar em `internal/platform/observability/observability_test.go` que nenhum log emite `usuario_id` em texto claro junto de material sensível, nem material criptográfico (FR-039, FR-042)
- [ ] T096 [P] Implementar a rotina de limpeza de `mensagens_processadas` (retenção de 30 dias) em `internal/adapter/postgres/mensagens.go` e ligá-la em `cmd/estoque/main.go`
- [ ] T097 [P] Escrever `README.md` na raiz do serviço com o roteiro de `quickstart.md`, as variáveis de ambiente de D10 e o apontamento para `specs/001-estoque-bloqueio-poltronas/`
- [ ] T098 Verificar mecanicamente, em `test/arquitetura_test.go`, que `internal/domain` e `internal/usecase` não importam `internal/adapter` (teste negativo deliberado que falha se a regra for quebrada)
- [ ] T099 Executar o roteiro completo de `quickstart.md` contra o `docker compose` e corrigir qualquer divergência entre o documento e o comportamento real

---

## Dependencies

### Entre fases

```text
Phase 1 (Setup)
   └─▶ Phase 2 (Foundational)   ← bloqueia todas as histórias
          ├─▶ Phase 3 (US1, P1) ← MVP
          ├─▶ Phase 4 (US2, P1) ─┐
          ├─▶ Phase 5 (US3, P1) ─┤ dependem de US1 apenas para ter o que confirmar/cancelar
          ├─▶ Phase 6 (US4, P2) ─┘
          └─▶ Phase 7 (US5, P3)  ← independente das demais; US1 usa carga direta enquanto ela não existe
                 └─▶ Phase 8 (Polish)
```

### Entre histórias

- **US1** não depende de nenhuma outra história. É o MVP.
- **US2**, **US3** e **US4** precisam de uma reserva pendente para agir; em teste isso é criado por fixture, então elas são desenvolvíveis em paralelo com US1 e verificáveis isoladamente.
- **US5** é independente de todas. Enquanto ela não existir, as demais provisionam a matriz por carga direta no banco (fixture de teste e alvo de `quickstart.md`).
- **T076** é o único ponto de costura entre histórias: liga o índice de prazo (US4) ao bloqueio (US1) e ao cancelamento (US3).

### Dentro de cada história

Testes antes da implementação. Domínio antes de caso de uso, caso de uso antes de adaptador, adaptador antes da ligação em `main.go`.

---

## Parallel Execution Examples

**Phase 1** — T002 a T005 e T007 a T009 em paralelo (arquivos distintos); T006 depois de T005.

**Phase 2** — três frentes simultâneas: configuração e domínio compartilhado (T012–T015), plataforma (T016, T023), adaptadores (T017, T020). T024 fecha, depois de todas.

**Phase 3 (US1)** — todos os testes T027 a T035 em paralelo; entidades T036 e T037 em paralelo; repositórios T042 e T043 em paralelo. T038 → T039 → T040 → T041 são sequenciais por dependência real.

**Phases 3 a 7** — com a Fase 2 pronta, cinco pessoas podem tocar uma história cada; os pontos de encontro são `cmd/estoque/main.go` (T049, T058, T065, T077, T089) e `internal/adapter/postgres/reservas.go` (T043, T055, T063), que devem ser coordenados ou feitos em sequência.

**Phase 8** — T090 a T097 em paralelo; T098 e T099 por último.

---

## Implementation Strategy

### MVP (entrega 1)

Fases 1, 2 e 3 (T001–T049). Entrega um serviço que concede e recusa bloqueios de forma atômica, cria reservas de 10 minutos e anuncia `reserva.criada`. Já é demonstrável e já impede *double-booking* — o valor central do serviço. Ainda não confirma, não cancela e não expira.

### Entrega 2 — ciclo de vida completo

Fases 4 e 5 (T050–T065). Fecha os dois desfechos de pagamento. A partir daqui o sistema vende ingressos de ponta a ponta.

### Entrega 3 — operação sustentável

Fase 6 (T066–T077). Sem ela, abandono de carrinho retira assentos do estoque de forma permanente — degradação rápida em operação real, daí a prioridade P2.

### Entrega 4 — autonomia de dados

Fase 7 (T078–T089). Elimina a carga manual da matriz. Depende de o `Servico-Catalogo` passar a publicar `sessao.criada`; o consumidor pode ser entregue e testado antes disso, contra o contrato proposto em `contracts/eventos.md`.

### Fechamento

Fase 8 (T090–T099). Verifica os critérios de sucesso que só existem como teste automatizado e deixa o serviço operável e documentado.
