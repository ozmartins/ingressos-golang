---

description: "Task list for feature implementation"
---

# Tasks: Catálogo de Filmes, Sessões e Reserva de Poltronas

**Input**: Design documents from `specs/001-catalogo-sessoes-reserva/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: incluídos. `research.md` (D9) define uma estratégia de testes em três camadas como decisão de projeto, e cada história da spec declara um "Independent Test" — portanto as tarefas de teste fazem parte do escopo, não são opcionais aqui.

**Organization**: agrupadas por história de usuário, para que cada uma seja implementável, testável e demonstrável de forma independente.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência pendente)
- **[Story]**: história a que a tarefa pertence (US1..US4)
- Caminhos de arquivo são relativos à raiz do módulo (`catalogo/`)

## Path Conventions

Módulo Go único com arquitetura hexagonal, conforme `plan.md`: `cmd/`, `internal/domain/`, `internal/usecase/`, `internal/adapter/`, `internal/platform/`, `migrations/`, `test/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: esqueleto do módulo, dependências e ferramental

- [X] T001 Inicializar o módulo Go em `go.mod` (Go 1.22+) e criar a árvore de diretórios de `plan.md` com um `.gitkeep` por pacote ainda vazio
- [X] T002 [P] Declarar as dependências de `research.md` em `go.mod`: `pgx/v5`, `grpc`, `protobuf`, `go-oidc/v3`, `gobreaker/v2`, `otel` (SDK + OTLP + otelhttp + otelgrpc), `golang-migrate/v4`, `testcontainers-go`
- [X] T003 [P] Configurar `.golangci.yml` com `govet`, `staticcheck`, `errcheck`, `revive` e o import-linter que impede `internal/domain` e `internal/usecase` de importarem `internal/adapter`
- [X] T004 [P] Criar `Makefile` com alvos `build`, `test`, `test-integration`, `lint`, `proto`, `migrate-up`
- [X] T005 [P] Escrever `docker-compose.yml` com PostgreSQL 16, Keycloak e um `estoque-simulado`, conforme o roteiro de `quickstart.md`
- [X] T006 Gerar o código do contrato gRPC em `gen/pb/estoque/` a partir de `specs/001-catalogo-sessoes-reserva/contracts/estoque.proto` via `protoc` (alvo `proto` do Makefile)
- [X] T007 [P] Escrever `Dockerfile` multi-estágio com imagem final distroless e usuário não-root

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: infraestrutura transversal exigida por todas as histórias

**⚠️ CRÍTICO**: nenhuma história pode começar antes desta fase terminar

- [X] T008 Implementar o carregamento e a validação de configuração em `internal/platform/config/config.go`, com todas as variáveis de `research.md` (D10) e falha na inicialização quando faltar obrigatória
- [X] T009 [P] Escrever teste unitário de configuração em `internal/platform/config/config_test.go`, cobrindo variável ausente, valor malformado e aplicação dos padrões
- [X] T010 [P] Criar a migração inicial do esquema em `migrations/000001_criar_esquema.up.sql` / `.down.sql` com a DDL de `data-model.md` (filmes, cinemas, salas, sessoes)
- [X] T011 [P] Criar a migração de índices em `migrations/000002_criar_indices.up.sql` / `.down.sql` com os seis índices da seção "Consultas e índices" de `data-model.md`
- [X] T012 Implementar o pool de conexões PostgreSQL em `internal/adapter/postgres/pool.go`, com verificação de conectividade na inicialização
- [X] T013 [P] Definir os tipos de paginação `PageRequest` e `Page[T]` em `internal/domain/shared/page.go`, com as regras de `data-model.md` (padrão 20, teto 100, `TemProxima`, recusa de tamanho acima do teto)
- [X] T014 [P] Escrever teste unitário de paginação em `internal/domain/shared/page_test.go`, cobrindo tamanho acima do teto, página além do fim e cálculo de `TemProxima`
- [X] T015 [P] Definir os erros sentinela de domínio em `internal/domain/shared/errors.go` (`ErrNaoEncontrado`, `ErrValidacao`, `ErrSessaoNaoReservavel`, `ErrPoltronasIndisponiveis`, `ErrEstoqueIndisponivel`, `ErrRespostaInvalidaDoParceiro`)
- [X] T016 Implementar a serialização RFC 9457 em `internal/adapter/http/problem.go`, mapeando cada erro sentinela para o `type` e o status de `contracts/errors.md`, incluindo `instance` com o `trace_id`
- [X] T017 [P] Escrever teste unitário de `problem.go` em `internal/adapter/http/problem_test.go`, verificando as dez categorias de `contracts/errors.md` e a ausência de vazamento de detalhe interno
- [X] T018 [P] Configurar a observabilidade em `internal/platform/observability/`: `logger.go` (slog JSON com `trace_id`/`span_id`), `tracing.go` (provedor OTel + OTLP) e `metrics.go` (medidores nomeados de `research.md` D6)
- [X] T019 Implementar o parsing e a validação de paginação em `internal/adapter/http/pagination.go`, traduzindo `page`/`page_size` para `PageRequest` e erros para `parametro-invalido`
- [X] T020 Montar o roteador em `internal/adapter/http/router.go` com `net/http.ServeMux` e as seis rotas de `contracts/openapi.yaml`, ainda com handlers vazios
- [X] T021 [P] Implementar os middlewares em `internal/adapter/http/middleware/`: `telemetria.go` (otelhttp + métricas), `correlacao.go` (extrai ou inicia o contexto W3C), `recuperacao.go` (pânico → 500 sem vazar stack) e `log.go` (registro estruturado por requisição, FR-034)
- [X] T022 [P] Implementar o indicador de saúde em `internal/platform/health/health.go` e o handler `GET /health`, verificando o banco e ignorando deliberadamente o estado do estoque (SC-012)
- [X] T023 Escrever a composição em `cmd/catalogo/main.go`: configuração → observabilidade → pool → adaptadores → casos de uso → servidor HTTP, com encerramento gracioso
- [X] T024 [P] Criar o utilitário de teste de integração em `test/integration/main_test.go`, subindo PostgreSQL via Testcontainers e aplicando as migrações uma vez por execução
- [X] T025 [P] Criar as fixtures de catálogo em `test/fixtures/catalogo_exemplo.sql`, com volume suficiente para exercitar paginação (mínimo 3 páginas de sessões) e casos de borda (filme sem sinopse e sem imagem)

**Checkpoint**: o serviço sobe, responde `/health`, aplica migrações e emite sinais — as histórias podem começar

---

## Phase 3: User Story 1 - Descobrir filmes em cartaz (Priority: P1) 🎯 MVP

**Goal**: expor publicamente a listagem paginada de filmes, com filtro por situação e omissão de `FORA_DE_CARTAZ` por padrão.

**Independent Test**: carregar filmes nas três situações, consultar `GET /api/v1/filmes` sem credencial e verificar os campos, o envelope de paginação e o comportamento do filtro — inclusive o filtro inválido.

### Tests for User Story 1

- [X] T026 [P] [US1] Teste de contrato de `GET /api/v1/filmes` em `test/contract/filmes_test.go`, validando a resposta contra `contracts/openapi.yaml`: envelope `{itens, pagina}`, campos obrigatórios e omissão dos opcionais nulos
- [X] T027 [P] [US1] Teste de integração em `test/integration/listar_filmes_test.go` cobrindo os quatro cenários de aceitação da US1, mais página além do fim e `page_size` acima do teto

### Implementation for User Story 1

- [X] T028 [P] [US1] Definir a entidade `Filme` e o tipo `StatusFilme` em `internal/domain/catalogo/filme.go`, com validação do conjunto de situações conhecidas
- [X] T029 [US1] Declarar a porta `FilmeRepository` em `internal/usecase/ports.go`
- [X] T030 [US1] Implementar o caso de uso `ListarFilmes` em `internal/usecase/listar_filmes.go`, aplicando a regra de omitir `FORA_DE_CARTAZ` quando nenhum filtro for informado (FR-008)
- [X] T031 [P] [US1] Escrever teste unitário do caso de uso em `internal/usecase/listar_filmes_test.go`, com repositório falso, cobrindo filtro ausente, filtro válido e filtro desconhecido
- [X] T032 [US1] Implementar `internal/adapter/postgres/filme_repository.go` com a consulta paginada de `data-model.md` (`COUNT(*) OVER ()`, `ORDER BY titulo, id`)
- [X] T033 [US1] Implementar o handler e os DTOs em `internal/adapter/http/handler_filmes.go` e `internal/adapter/http/dto.go`, ligando a rota registrada em T020

**Checkpoint**: a vitrine de filmes funciona ponta a ponta, paginada e sem autenticação — MVP demonstrável

---

## Phase 4: User Story 2 - Consultar a grade de sessões (Priority: P1)

**Goal**: expor a grade paginada de sessões com dados consolidados de filme, cinema e sala, filtrável por filme, cinema e data.

**Independent Test**: cadastrar sessões em cinemas, salas e datas distintas e verificar cada combinação de filtros, a estabilidade da paginação entre páginas consecutivas e a exclusão de sessões canceladas e finalizadas.

### Tests for User Story 2

- [X] T034 [P] [US2] Teste de contrato de `GET /api/v1/sessoes` em `test/contract/sessoes_test.go`, validando o envelope e todos os campos consolidados contra `contracts/openapi.yaml`
- [X] T035 [P] [US2] Teste de integração em `test/integration/consultar_sessoes_test.go` cobrindo os seis cenários da US2, com atenção a: páginas consecutivas sem repetição nem omissão (FR-004) e `total` refletindo o filtro, não a página
- [X] T036 [P] [US2] Teste de integração em `test/integration/sessoes_juncao_test.go` para o caso de borda de sessão órfã: sessão cujo filme ou sala não existe é omitida da grade sem derrubar a consulta

### Implementation for User Story 2

- [X] T037 [P] [US2] Definir `Sessao`, `StatusSessao` e `Idioma` em `internal/domain/catalogo/sessao.go`, incluindo o predicado `AceitaReserva()` com a dupla condição de `data-model.md` (status `AGENDADA` **e** início futuro)
- [X] T038 [P] [US2] Definir a vista de leitura `SessaoDetalhada` em `internal/domain/catalogo/sessao_detalhada.go`, com `preco_base` em decimal exato — nunca `float64`
- [X] T039 [P] [US2] Escrever teste unitário de `AceitaReserva()` em `internal/domain/catalogo/sessao_test.go`, cobrindo as quatro situações e a sessão `AGENDADA` com horário já passado
- [X] T040 [US2] Declarar a porta `SessaoRepository` em `internal/usecase/ports.go` (métodos de consulta paginada e de busca por id)
- [X] T041 [US2] Implementar o caso de uso `ConsultarSessoes` em `internal/usecase/consultar_sessoes.go`, com o recorte de situação de FR-016 e a validação de filtros
- [X] T042 [P] [US2] Escrever teste unitário em `internal/usecase/consultar_sessoes_test.go` para combinação de filtros e recorte de situação
- [X] T043 [US2] Implementar `internal/adapter/postgres/sessao_repository.go` com a junção de quatro tabelas, o filtro de data por intervalo (`[data, data+1)`, para preservar o índice) e `ORDER BY data_hora_inicio, id`
- [X] T044 [US2] Implementar o handler em `internal/adapter/http/handler_sessoes.go`, incluindo a validação do formato `YYYY-MM-DD` e dos UUIDs de filtro
- [X] T045 [US2] Mapear `pgtype.Numeric` para a representação decimal exata em `internal/adapter/postgres/tipos.go`, com teste de ida e volta que prova a ausência de perda de precisão

**Checkpoint**: US1 e US2 funcionam de forma independente; a jornada de navegação está completa

---

## Phase 5: User Story 3 - Reservar poltronas (Priority: P1)

**Goal**: autenticar a pessoa, validar a solicitação localmente e delegar o bloqueio ao `Servico-Estoque`, traduzindo cada desfecho em uma resposta padronizada.

**Independent Test**: com uma sessão conhecida e credenciais válidas, exercitar os desfechos de sucesso, indisponibilidade, sessão não reservável e estoque inacessível contra um estoque simulado em `bufconn`.

**Dependência**: precisa da busca de sessão por id (T040/T043 da US2). Se esta história for construída antes da US2, T040 e a consulta por id de T043 entram aqui como pré-requisito.

### Tests for User Story 3

- [X] T046 [P] [US3] Teste de contrato de `POST /api/v1/sessoes/{id}/reservar` em `test/contract/reservar_test.go`, validando os corpos de 201 e de cada erro contra `contracts/openapi.yaml` e `contracts/errors.md`
- [X] T047 [P] [US3] Teste de integração em `test/integration/reservar_poltronas_test.go` cobrindo os seis cenários da US3, com estoque simulado em `grpc/test/bufconn`; o simulado registra o `trace_id` recebido nos metadados e o teste confirma que é o mesmo enviado no cabeçalho `traceparent` da requisição HTTP (SC-011, FR-036)
- [X] T048 [P] [US3] Teste em `test/integration/reservar_sem_contato_test.go` provando que 401 e 404 **não** chamam o estoque, verificando que o contador `estoque.bloqueio.total` não varia
- [X] T049 [P] [US3] Teste de resiliência em `test/integration/reservar_resiliencia_test.go`: timeout de 2s (SC-004), abertura da recusa rápida após falhas consecutivas com resposta em < 200 ms (SC-007), retomada automática, e resposta de sucesso sem `reserva_id`/`expira_em` mapeada para 502; ao final, verificar que `estoque.bloqueio.total` foi emitida com os quatro rótulos de desfecho (`sucesso`, `indisponivel`, `timeout`, `recusa_rapida`), dos quais T048 depende (FR-035)
- [X] T050 [P] [US3] Teste de concorrência em `test/integration/reservar_concorrencia_test.go`: 50 solicitações paralelas sobre as mesmas poltronas, exatamente uma confirmada (SC-006), executado com `-race`

### Implementation for User Story 3

- [X] T051 [P] [US3] Definir `SolicitacaoReserva` e `ResultadoReserva` em `internal/domain/reserva/reserva.go`, com as quatro validações e o invariante de integridade de `data-model.md`
- [X] T052 [P] [US3] Escrever teste unitário em `internal/domain/reserva/reserva_test.go` para lista vazia, duplicatas, `usuario_id` ausente e resultado de sucesso sem `reserva_id`
- [X] T053 [US3] Implementar a verificação de credenciais em `internal/adapter/identidade/keycloak.go` com `go-oidc` e `RemoteKeySet`, validando assinatura RS256, `exp`, `iss` e `aud`, e extraindo `sub`
- [X] T054 [P] [US3] Escrever teste em `internal/adapter/identidade/keycloak_test.go` com emissor de JWKS local: token válido, expirado, assinatura inválida, emissor errado e token sem `sub`
- [X] T055 [US3] Implementar o middleware de autenticação em `internal/adapter/http/middleware/autenticacao.go`, aplicado somente à rota de reserva, colocando o `usuario_id` no contexto
- [X] T056 [US3] Declarar a porta `EstoqueGateway` em `internal/usecase/ports.go`
- [X] T057 [US3] Implementar o cliente gRPC em `internal/adapter/estoque/client.go`: conexão duradoura, `context.WithTimeout(2s)`, **sem** política de retentativa, instrumentado com `otelgrpc`
- [X] T058 [US3] Implementar a recusa rápida em `internal/adapter/estoque/breaker.go` com `gobreaker`, parametrizada pela configuração (5 falhas, 30s), expondo `estoque.breaker.state`
- [X] T059 [US3] Implementar o mapeamento de desfechos em `internal/adapter/estoque/mapper.go` conforme `research.md` D5: OK/`sucesso=true` → resultado; `sucesso=false` → indisponíveis; `DeadlineExceeded`/`Unavailable`/breaker aberto → estoque indisponível; sucesso incompleto → resposta inválida do parceiro
- [X] T060 [US3] Implementar o caso de uso `ReservarPoltronas` em `internal/usecase/reservar_poltronas.go`, na ordem: validar entrada → buscar sessão → verificar `AceitaReserva()` → chamar o estoque
- [X] T061 [P] [US3] Escrever teste unitário em `internal/usecase/reservar_poltronas_test.go` com portas falsas, provando que cada recusa local ocorre **antes** de qualquer chamada ao estoque
- [X] T062 [US3] Implementar o handler em `internal/adapter/http/handler_reservar.go`, traduzindo cada erro sentinela para o `problem+json` correspondente
- [X] T063 [US3] Emitir o registro de auditoria da reserva em `internal/usecase/reservar_poltronas.go` conforme FR-033: `usuario_id`, `sessao_id`, poltronas, desfecho e `reserva_id`, sem jamais registrar o token

**Checkpoint**: a jornada completa de compra funciona; as três histórias P1 estão entregues

---

## Phase 6: User Story 4 - Consultar cinemas e salas (Priority: P2)

**Goal**: expor as listagens paginadas de cinemas e das salas de um cinema.

**Independent Test**: cadastrar cinemas com salas, consultar ambas as listagens sem credencial e verificar os campos, a paginação e o 404 para cinema inexistente.

### Tests for User Story 4

- [X] T064 [P] [US4] Teste de contrato de `GET /api/v1/cinemas` e `GET /api/v1/cinemas/{id}/salas` em `test/contract/cinemas_test.go`
- [X] T065 [P] [US4] Teste de integração em `test/integration/listar_cinemas_test.go` cobrindo os três cenários da US4

### Implementation for User Story 4

- [X] T066 [P] [US4] Definir `Cinema` em `internal/domain/catalogo/cinema.go`
- [X] T067 [P] [US4] Definir `Sala` e `TipoTela` em `internal/domain/catalogo/sala.go`
- [X] T068 [US4] Declarar as portas `CinemaRepository` e `SalaRepository` em `internal/usecase/ports.go`
- [X] T069 [US4] Implementar os casos de uso em `internal/usecase/listar_cinemas.go` e `internal/usecase/listar_salas.go`, com a verificação de existência do cinema antes de listar salas (FR-013)
- [X] T070 [P] [US4] Implementar `internal/adapter/postgres/cinema_repository.go` (`ORDER BY nome, id`)
- [X] T071 [P] [US4] Implementar `internal/adapter/postgres/sala_repository.go` (`ORDER BY numero, id`, filtrado por `cinema_id`)
- [X] T072 [US4] Implementar os handlers em `internal/adapter/http/handler_cinemas.go`

**Checkpoint**: todas as quatro histórias estão funcionais e testáveis de forma independente

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T073 [P] Executar o roteiro completo de `quickstart.md` contra o ambiente do `docker-compose`, corrigindo qualquer divergência entre o guia e o comportamento real
- [X] T074 [P] Medir SC-003 nos dois pontos que o critério agora fixa, registrando os tempos em `test/integration/performance_test.go`: volume base (500 filmes, 50 cinemas, 300 salas, 5.000 sessões) abaixo de 1 segundo, e dez vezes esse volume abaixo de 2 segundos
- [X] T075 [P] Verificar com `EXPLAIN ANALYZE` que as quatro consultas de listagem usam os índices de T011, registrando os planos em `test/integration/planos_consulta_test.go` e ajustando `migrations/000002_criar_indices.up.sql` se alguma cair em varredura sequencial
- [X] T076 Revisar todas as respostas de erro contra `contracts/errors.md`, confirmando que nenhuma vaza mensagem do gRPC, endereço do estoque, SQL ou stack trace (FR-028)
- [X] T077 [P] Escrever `README.md` do serviço com as variáveis de ambiente, como subir localmente e as duas divergências em relação à ERS (envelope de paginação e formato do 409)
- [X] T078 [P] Confirmar em `.golangci.yml` que o import-linter falha quando `internal/domain` ou `internal/usecase` importam `internal/adapter`, com um teste negativo deliberado e descartado após a verificação
- [X] T079 [P] Provar a leitura sempre atual em `test/integration/leitura_atual_test.go`: alterar um filme e uma sessão diretamente no banco e verificar que a consulta seguinte já reflete a mudança, sem defasagem (FR-010, SC-002)
- [X] T080 Subir a imagem construída a partir do `Dockerfile` (T007) com apenas as variáveis de ambiente obrigatórias, sem arquivo de configuração nem alteração no artefato, e verificar que `/health` responde 200; registrar o procedimento na seção de execução local do `README.md` (SC-010)
- [X] T081 Executar `go test -race ./...` e `go test -tags=integration ./test/integration/` com todos os testes verdes, e registrar o comando no alvo `test` do `Makefile`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Fase 1)**: sem dependências
- **Foundational (Fase 2)**: depende da Fase 1 — **bloqueia todas as histórias**
- **US1, US2, US4 (Fases 3, 4, 6)**: dependem apenas da Fase 2; são mutuamente independentes
- **US3 (Fase 5)**: depende da Fase 2 e da busca de sessão por id (T040, T043)
- **Polish (Fase 7)**: depende das histórias que se pretende entregar

### User Story Dependencies

- **US1 (P1)**: independente
- **US2 (P1)**: independente
- **US3 (P1)**: precisa de `SessaoRepository.BuscarPorID` — a única dependência real entre histórias nesta feature
- **US4 (P2)**: independente

### Within Each User Story

- Testes antes da implementação; devem falhar primeiro
- Entidades de domínio → portas → casos de uso → repositórios → handlers
- Handler por último: ele só traduz, e traduzir algo que ainda não existe gera retrabalho

### Parallel Opportunities

- Fase 1: T002–T005 e T007 em paralelo (T006 depende de T001)
- Fase 2: T009, T010, T011, T013, T014, T015, T017, T018, T021, T022, T024, T025 em paralelo
- Concluída a Fase 2, US1, US2 e US4 podem ser desenvolvidas simultaneamente por pessoas diferentes
- Dentro de cada história, os testes marcados [P] e as entidades de domínio [P] rodam em paralelo

---

## Parallel Example: User Story 2

```bash
# Testes da US2, juntos:
Task: "Teste de contrato de GET /api/v1/sessoes em test/contract/sessoes_test.go"
Task: "Teste de integração dos seis cenários em test/integration/consultar_sessoes_test.go"
Task: "Teste do caso de borda de sessão órfã em test/integration/sessoes_juncao_test.go"

# Entidades de domínio da US2, juntas:
Task: "Definir Sessao, StatusSessao e Idioma em internal/domain/catalogo/sessao.go"
Task: "Definir SessaoDetalhada em internal/domain/catalogo/sessao_detalhada.go"
```

---

## Implementation Strategy

### MVP (User Story 1)

1. Fase 1: Setup
2. Fase 2: Foundational — bloqueia tudo, não pode ser encurtada
3. Fase 3: US1
4. **PARAR E VALIDAR**: `GET /api/v1/filmes` paginado, com filtro e sem credencial
5. Demonstrável: a vitrine de programação já entrega valor sozinha

### Entrega incremental

1. Setup + Foundational → base pronta
2. + US1 → vitrine de filmes (MVP)
3. + US2 → jornada de navegação completa (filme → onde e quando)
4. + US3 → conversão: a navegação vira reserva
5. + US4 → descoberta por localização
6. + Fase 7 → verificação de desempenho, índices e documentação

Cada incremento é entregável sem quebrar o anterior.

### Estratégia com várias pessoas

Concluída a Fase 2: uma pessoa em US1+US2 (compartilham o vocabulário de catálogo), outra em US3 (autenticação e integração gRPC são um bloco coeso e isolado), uma terceira em US4. US3 precisa que T040 e T043 estejam prontos — combinar essa entrega cedo evita bloqueio.

---

## Notes

- `[P]` = arquivos diferentes, sem dependência pendente
- A ordem dentro de cada história respeita a inversão de dependência: nada em `internal/domain` ou `internal/usecase` pode importar `internal/adapter` (T003 e T078 verificam isso mecanicamente)
- Commitar por tarefa ou por grupo lógico
- Parar em qualquer checkpoint para validar a história de forma isolada
