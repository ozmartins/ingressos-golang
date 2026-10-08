# Tasks: Persistência do Estoque via GORM

**Input**: `specs/002-persistencia-gorm/` — [spec.md](./spec.md), [plan.md](./plan.md), [research.md](./research.md), [data-model.md](./data-model.md), [quickstart.md](./quickstart.md)

**Tests**: nenhum teste novo de comportamento — o critério de aceite é a suíte existente (unitária, contrato e integração) passando **sem alterar asserções** (SC-001). A única adição de teste é a regra de arquitetura (US3). Comandos rodam dentro de `ingressos-golang/estoque/`.

**Commits**: direto na `master`, sem branch (regra do mantenedor). Não dar push sem pedido.

**Atenção de ordem**: trocar `Banco` (T005) quebra a compilação de `internal/adapter/postgres` até T012 terminar. Entre T005 e T012 só `go vet ./internal/adapter/postgres/...` por arquivo faz sentido; o build completo volta verde em T013.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência pendente)
- **[Story]**: US1, US2, US3 (só nas fases de história)

---

## Phase 1: Setup

- [X] T001 Antes de tocar no código, registrar a linha de base: rodar `make test` e `make test-integration` em `estoque/` e anotar o resultado (quais testes passam hoje **e o p99 de bloqueio medido por `test/integration/desempenho_test.go`**) em `specs/002-persistencia-gorm/quickstart.md` seção "Linha de base" — se algo já falhar, parar e perguntar ao mantenedor (princípio X)
- [X] T002 Adicionar dependências em `go.mod`/`go.sum`: `go get gorm.io/gorm@v1.31.2 gorm.io/driver/postgres@v1.6.2` e `go mod tidy` (confirmar que `pgx/v5` continua como dependência direta)
- [X] T003 Spike descartável (arquivo temporário fora do repo, no scratchpad) contra um Postgres de teste, para fechar os dois riscos da pesquisa antes de reescrever: (a) subconsulta `*gorm.DB` com `clause.Locking{Options:"SKIP LOCKED"}` + `clause.Returning` emite `FOR UPDATE SKIP LOCKED` e devolve ids (research D2); (b) `[]byte(nil)` em coluna `jsonb` grava `NULL` (research D6); (c) `Model(&poltronaRow{}).Where(...).Updates(map[string]any{"status": ..., "atualizado_em": gorm.Expr("now()")})` realmente altera `atualizado_em` com o modelo de T004 (ler a coluna de volta). Registrar o veredito de cada um em `research.md` ao final de D2 e D6, indicando se o fallback `Raw` será usado

---

## Phase 2: Foundational (bloqueia todas as histórias)

- [X] T004 Criar `internal/adapter/postgres/modelos.go` com as structs privadas `poltronaRow`, `reservaRow`, `reservaPoltronaRow`, `outboxRow`, `mensagemProcessadaRow`, cada uma com `TableName()` **sem** schema e tags `gorm:"column:...;primaryKey"` conforme `data-model.md` (`valor_total` `*string`, `finalizado_em`/`publicado_em` `*time.Time`, `payload`/`trace_context` `[]byte` com `type:jsonb`; `criado_em` e `processado_em` somente-leitura para o GORM (`<-:false`); `atualizado_em` com `<-:update`, porque é escrito por `Expr("now()")` em todo `UPDATE` de poltrona e nunca na criação). Mais funções de conversão modelo ↔ domínio (`poltrona.Poltrona`, `reserva.Reserva`)
- [X] T005 Reescrever `internal/adapter/postgres/postgres.go`: `Banco{db *gorm.DB}`; `Abrir(ctx, url)` conforme research D1 (`pgx.ParseConfig` + `search_path=estoque` + `stdlib.OpenDB` + `SetConnMaxIdleTime(5*time.Minute)` + `gorm.Open` com `SkipDefaultTransaction`, `logger.Discard`, `DisableAutomaticPing`, e `PingContext` com os mesmos erros "DATABASE_URL malformada"/"banco não respondeu"); `Fechar`, `Verificar`, `SQL() *sql.DB` (substitui `Pool()`), `EmTransacao(ctx, func(*gorm.DB) error)` usando `db.WithContext(ctx).Transaction`, e manter `indisponivel`, `ehConflitoDeTravamento`, `ehViolacaoDeUnicidade` com `errors.As(*pgconn.PgError)`. Remover `Pool()` e o import de `pgxpool`

**Checkpoint**: `Banco` novo definido; os demais arquivos do adaptador ainda não compilam.

---

## Phase 3: User Story 1 — Comportamento de negócio idêntico (P1) 🎯 MVP

**Goal**: toda leitura/escrita do estoque passa pelo GORM, com o mesmo SQL efetivo e as mesmas garantias de exclusão, atomicidade e idempotência.

**Independent Test**: `make test` e `make test-integration` verdes, sem alterar asserções; quickstart §4 ponta a ponta.

- [X] T006 [P] [US1] Reescrever `internal/adapter/postgres/mensagens.go`: `registrarProcessada(tx *gorm.DB, fila, messageID)` com `clause.OnConflict{DoNothing:true}` e `RowsAffected == 1`; `LimparMensagensProcessadas` com `Where("processado_em < now() - ?::interval", retencao.String()).Delete(&mensagemProcessadaRow{})`
- [X] T007 [P] [US1] Reescrever `internal/adapter/postgres/outbox.go`: `enfileirarFato` (`OnConflict` em `message_id`, `trace_context` `nil` quando vazio) e `PendentesParaPublicar` (`Locking{Strength:"UPDATE", Options:"SKIP LOCKED"}`, `Order("id")`, `Limit`, mesma transação marcando `publicado_em = now()` ou `tentativas = tentativas + 1`). Manter o tipo `FatoNaCaixa` e a assinatura (usados por `internal/adapter/amqp/publicador.go`)
- [X] T008 [P] [US1] Reescrever `internal/adapter/postgres/poltronas.go`: `MapaDaSessao` (`Where`+`Order("fileira, numero")`+`Find` → conversão para domínio) e `ProvisionarMatriz` (idempotência via `registrarProcessada`; inserção em lote com `CreateInBatches(rows, 500)` + `OnConflict{DoNothing:true}`; `RowsAffected == 0` → `TransicaoIgnoradaDuplicata`)
- [X] T009 [US1] Reescrever `Conceder` e `sessaoProvisionada` em `internal/adapter/postgres/bloqueio.go`: travar com `Locking{UPDATE, NOWAIT}` + `Where("sessao_id = ? AND rotulo IN ?")` + `Order("rotulo")`; `55P03` → `ErrPoltronasIndisponiveis` ("disputa simultânea"); mesmas verificações (`ErrSessaoNaoProvisionada`, `ErrPoltronaInexistente`, poltrona não-LIVRE); depois criar reserva, vínculos e `UPDATE poltronas … atualizado_em = now()` e `enfileirarFato`, tudo na mesma transação. Depende de T004, T005, T007
- [X] T010 [US1] Reescrever `aplicarDesfecho`, `Confirmar` e `Cancelar` em `internal/adapter/postgres/bloqueio.go`: idempotência por `registrarProcessada`, `UPDATE reservas … WHERE id = ? AND status = 'PENDENTE'` com `RowsAffected`, distinção `IgnoradaEstadoFinal`/`IgnoradaInexistente` por `Count`, e liberação/ocupação das poltronas via subconsulta de `reserva_poltronas`. Depende de T006, T009 (mesmo arquivo)
- [X] T011 [US1] Reescrever `ExpirarVencidas` e `ExpirarUma` em `internal/adapter/postgres/expiracao.go`: `pg_try_advisory_xact_lock` via `Raw(...).Scan`, `UPDATE … WHERE id IN (subconsulta ORDER BY expira_em LIMIT ? FOR UPDATE SKIP LOCKED) RETURNING id` (ou `Raw` conforme veredito de T003), e liberação das poltronas. Depende de T005
- [X] T012 [US1] Reescrever `CancelarPendentesDaSessao` em `internal/adapter/postgres/expiracao.go` com a mesma forma de T011 (sem limite, `ORDER BY criado_em`, contagem prévia de `CONFIRMADA`, idempotência por `registrarProcessada`). Depende de T011 (mesmo arquivo)
- [X] T013 [US1] Fechar a compilação: ajustar `cmd/estoque/main.go` se algo além de imports mudou, e rodar `go build ./...`, `go vet ./...` e `make lint` — corrigir apenas o que a troca causou. Conferir que nenhum arquivo fora de `internal/adapter/postgres`, `cmd/estoque`, `test/` e `go.mod/go.sum` foi alterado (`git status`)
- [X] T014 [US1] Adaptar o harness de integração: em `test/integration/harness_test.go` trocar `Pool *pgxpool.Pool` por `Pool *sql.DB` (`banco.SQL()`); em `bloqueio_test.go`, `invariante_test.go`, `esquema_test.go`, `broker_test.go` e `harness_test.go` trocar `Pool.QueryRow(`→`Pool.QueryRowContext(` e `Pool.Query(`→`Pool.QueryContext(` **sem mexer em asserções**; `test/integration/main_test.go` (`aplicarMigracoes`) permanece com `pgxpool`. Depende de T005
- [X] T015 [US1] Teste de integração do rollback conjunto em `test/integration/atomicidade_test.go` (build tag `integration`, Postgres real): provocar falha **dentro** da transação de `Conceder` depois do bloqueio das poltronas e da gravação da reserva (ex.: `fato.MessageID` com mais de 64 caracteres, violando `VARCHAR(64)` do outbox) e verificar que não restou reserva, vínculo, poltrona `RESERVADA` nem linha no outbox, e que o erro devolvido é `ErrDependenciaIndisponivel` sem texto de driver. Se algum teste existente já cobrir exatamente isso (conferir `bloqueio_test.go`, `largada_test.go`), citar qual em vez de duplicar
- [X] T016 [US1] Rodar `make test` e `make test-integration`; comparar com a linha de base de T001. Qualquer teste que passava e agora falha é bug da troca, não do teste — corrigir o adaptador. Se a única saída for alterar uma asserção, parar e perguntar ao mantenedor (princípio X)

**Checkpoint**: US1 entregue — equivalência funcional provada pela suíte existente.

---

## Phase 4: User Story 2 — Esquema preservado, sem migração de dados (P2)

**Goal**: o serviço novo opera sobre banco criado/populado pela versão anterior; ele não emite DDL.

**Independent Test**: quickstart §3.

- [X] T017 [P] [US2] Verificar por inspeção que nada no código de produção emite DDL nem `AutoMigrate`: `grep -rniE "AutoMigrate|CREATE TABLE|ALTER TABLE|Migrator\(" internal cmd` deve retornar vazio; registrar o resultado na seção de verificação ao fim de `quickstart.md`
- [X] T018 [US2] Executar o cenário de banco populado (quickstart §3): subir a versão anterior (`git worktree add --detach /tmp/estoque-antes HEAD`, sem criar branch; remover o worktree ao final) com `docker compose up --build`, criar sessão e reserva pendente, trocar para a versão nova **sem recriar o volume** e confirmar a reserva via `make publicar-pagamento`; anotar o resultado em `quickstart.md`
- [X] T019 [US2] Executar do zero `docker compose up --build` (banco vazio) e confirmar que `migrate-estoque` cria o esquema e `estoque` fica saudável (`/health` conforme `ingressos-golang/URLS.txt`)

**Checkpoint**: US2 verificada.

---

## Phase 5: User Story 3 — Fronteira núcleo × infraestrutura (P3)

**Goal**: o núcleo continua sem conhecer o GORM, e isso é verificado mecanicamente.

**Independent Test**: `go test ./test/ -run TestNucleoNaoImportaInfraestrutura`.

- [X] T020 [US3] Em `test/arquitetura_test.go`, acrescentar `"gorm.io"` à lista `proibidos`
- [X] T021 [US3] Prova negativa: importar temporariamente `gorm.io/gorm` em um arquivo de `internal/usecase/`, confirmar que `TestNucleoNaoImportaInfraestrutura` **falha** apontando o import, e reverter. (Garante que a nova regra pode falhar — constituição: teste que não pode falhar não conta)
- [X] T022 [US3] `go test -race ./internal/domain/... ./internal/usecase/...` sem Docker, confirmando que o núcleo segue testável sem banco, rede ou servidor

**Checkpoint**: todas as histórias entregues.

---

## Phase 6: Polish

- [X] T023 [P] Atualizar `estoque/README.md` e a seção Architecture dos `CLAUDE.md` (`ingressos-golang/CLAUDE.md`, `../CLAUDE.md`) **só se** citarem "pgx"/"SQL à mão" como mecanismo de persistência do estoque (verificado até aqui: os `CLAUDE.md` só citam pgx na lista de imports proibidos ao núcleo, que continua correta — então a tarefa pode terminar sem alteração; registrar "nada a alterar" se for o caso). Não editar `specs/001-*`: o mantenedor decidiu tratá-la como registro histórico
- [X] T024 [P] Rodar `make test-integration` mais uma vez com `-race` e `-count=3` nos testes de concorrência (`invariante_test.go`, `bloqueio_test.go`) para detectar flakiness introduzida pela troca (SC-002)
- [X] T025 Comparar o p99 de `desempenho_test.go` com a linha de base de T001 (SC-006): aceitar se ≤ 100 ms **e** ≤ 120% da linha de base; fora disso, investigar (ex.: `Select` faltando, consulta extra por `Updates`) antes de commitar. Anotar os dois números em `quickstart.md`
- [X] T026 Commitar direto na `master` (sem branch), mensagem em inglês no estilo Conventional Commits (ex.: `refactor(estoque): persist through GORM instead of hand-written pgx SQL`), incluindo `specs/002-persistencia-gorm/`. Sem push

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 → US1 (Phase 3) → US2 (Phase 4) e US3 (Phase 5) podem ir em paralelo depois de T016 → Polish.
- US2 e US3 dependem de US1 compilando (T013), mas T020 pode ser feito a qualquer momento.
- Dentro de US1: T006, T007, T008 em paralelo; T009 depende de T007; T010 depende de T006 e T009 (mesmo arquivo); T011→T012 (mesmo arquivo); T014 em paralelo com T009–T012 após T005.
- T003 (spike) decide a forma de T011/T012; fazer antes.

### Parallel example (US1)

```
T006 mensagens.go   ┐
T007 outbox.go      ├─ em paralelo
T008 poltronas.go   ┘
→ T009 → T010        (bloqueio.go)
→ T011 → T012        (expiracao.go)
T014 harness de integração (em paralelo com T009–T012)
```

## Implementation Strategy

- **MVP = US1 inteira** (T001–T016): a troca só é útil quando a suíte existente passa. Não há entrega parcial do adaptador, pois `Banco` é único.
- US2 e US3 são verificações e uma regra de teste; entram em seguida, no mesmo commit ou em commits seguidos na `master`.
- Cada commit intermediário só se o build estiver verde; caso contrário, um único commit ao fim de T016.
