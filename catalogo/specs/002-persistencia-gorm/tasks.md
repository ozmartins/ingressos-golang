# Tasks: Persistência do Catálogo via GORM

**Input**: Design documents from `specs/002-persistencia-gorm/`
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [quickstart.md](./quickstart.md)

**Tests**: nenhum teste novo de comportamento — o critério de aceite é a suíte existente, com asserções inalteradas (SC-001). Há duas adaptações de teste, ambas de infraestrutura — o harness de integração (T014) e o teste de planos de consulta (T015) — e um teste novo, o de concorrência da caixa de saída (T016), que cobre uma lacuna da linha de base.

**Organization**: por user story. Todo o trabalho fica em `catalogo/`; os caminhos abaixo são relativos a ele. Os repositórios e `main.go` mudam de assinatura juntos, então o pacote só volta a compilar ao fim de T013 — rodar `go build ./...` antes disso não é sinal útil.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência pendente)
- **[Story]**: user story a que a tarefa serve

---

## Phase 1: Setup

- [X] T001 Adicionar `gorm.io/gorm` v1.31.2 e `gorm.io/driver/postgres` v1.6.2 ao `go.mod`/`go.sum` (`go get`, com `GOFLAGS=-mod=mod`; ambos estão no cache local) e rodar `go mod tidy`; conferir que `go build ./...` ainda compila (nada usa o GORM ainda) e que a versão do `pgx/v5` não regrediu.

---

## Phase 2: Foundational (bloqueia todas as user stories)

**Purpose**: a nova base de conexão, os modelos e os utilitários que todo repositório usa.

- [X] T002 Criar `internal/adapter/postgres/postgres.go` com `Banco{db *gorm.DB}`: `Abrir(ctx, url)` conforme research D1 (`pgx.ParseConfig` → `RuntimeParams["search_path"]="catalogo"` → `stdlib.OpenDB` → `SetMaxOpenConns(10)`/`SetConnMaxLifetime(time.Hour)` → `gorm.Open` com `SkipDefaultTransaction`, `Logger: logger.Discard`, `DisableAutomaticPing` → `PingContext` de 5 s; mensagem "DATABASE_URL inválida" preservada), `Fechar()`, `Ping(ctx) error` (satisfaz `health.Verificador`), `SQL() *sql.DB` e `EmTransacao(ctx, fn func(*gorm.DB) error) error`. Manter a constante `Schema = "catalogo"`.
- [X] T003 [P] Criar `internal/adapter/postgres/modelos.go` com os modelos privados `filmeRow`, `cinemaRow`, `salaRow`, `sessaoRow`, `outboxRow` e a projeção `sessaoDetalhadaRow`, conforme [data-model.md](./data-model.md): `TableName()` sem schema, tags `gorm:"column:...;primaryKey"`, `layout`/`payload`/`trace_context` como `[]byte`, `preco_base` como `string`, funções de conversão modelo ↔ domínio. Confirmar contra as migrações `000001`–`000006` quais colunas são anuláveis (`sinopse`, `imagem_url`, `trace_context`, `atualizado_em`).
- [X] T004 [P] Reescrever `internal/adapter/postgres/tipos.go`: trocar `dinheiroDeNumeric(pgtype.Numeric)` por `dinheiroDeTexto(string)` via `big.Rat.SetString` → `catalogo.DinheiroDeRat`, preservando as mensagens de erro para valor vazio/inválido (research D6).
- [X] T005 Reescrever `internal/adapter/postgres/paginacao.go`: `consultarPaginado[T]` passa a receber uma consulta `*gorm.DB` já filtrada e uma função de conversão; faz `Count(&total)` e depois `Order/Limit/Offset/Find`, devolvendo `shared.NovaPage(...)` com as mesmas mensagens de erro (research D5). Depende de T002, T003.
- [X] T006 Remover `internal/adapter/postgres/pool.go` e `internal/adapter/postgres/transacao.go` (absorvidos por `postgres.go`).

**Checkpoint**: base pronta; os repositórios podem ser migrados.

---

## Phase 3: User Story 1 — Comportamento de negócio idêntico (Priority: P1) 🎯 MVP

**Goal**: toda leitura e escrita passa pelo GORM, com o mesmo comportamento observável.

**Independent Test**: `make test-integration` verde, sem alterar nenhuma asserção; fluxo do [quickstart](./quickstart.md) §4 com os mesmos resultados.

- [X] T007 [P] [US1] Migrar `internal/adapter/postgres/filme_repository.go` para GORM (`Listar` com `Where("status IN ?")` + `Order("titulo, id")`, `BuscarPorID`, `Criar`, `Atualizar`, `MarcarForaDeCartaz`; `RowsAffected == 0` ⇒ `shared.NaoEncontrado`; `atualizado_em` via `gorm.Expr("CURRENT_TIMESTAMP")`; validação de status desconhecido preservada). Construtor passa a receber `*Banco`. Toda chamada ao GORM usa `WithContext(ctx)` (prazos e cancelamento da requisição — princípio V); o mesmo vale para T008–T011.
- [X] T008 [P] [US1] Migrar `internal/adapter/postgres/cinema_repository.go` (`Listar` com `Where("ativo = ?")` só quando `filtro.Ativo != nil`, `Order("nome, id")`; `BuscarPorID`, `Criar`, `Atualizar`, `Desativar`, `Existe`). `ativo = false` precisa ser gravado: usar `Select`/mapa nos updates (research D3).
- [X] T009 [P] [US1] Migrar `internal/adapter/postgres/sala_repository.go` (`Listar` com filtros condicionais e `Order("cinema_id, numero, id")`; `BuscarPorID`, `Criar`, `Atualizar`, `Desativar`, `NumeroEmUso`; layout JSONB via `layoutParaJSON`/`lerSala` sobre `[]byte`).
- [X] T010 [P] [US1] Migrar `internal/adapter/postgres/outbox.go`: `enfileirarFato(tx *gorm.DB, fato)` com `clause.OnConflict{Columns: message_id, DoNothing: true}` (`trace_context` ausente ⇒ `nil`/`NULL`); `CaixaDeSaida.Drenar` dentro de `EmTransacao`, com `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}`, `Order("id")`, `Limit`, contagem de tentativa com `gorm.Expr("tentativas + 1")` e `publicado_em = now()`; `publicar` continua sendo chamado com a transação aberta (research D2). Registrar no código, em comentário, por que a publicação fica dentro da transação.
- [X] T011 [US1] Migrar `internal/adapter/postgres/sessao_repository.go` (depende de T010 por `enfileirarFato`): `Consultar` com `Table("sessoes s")` + 3 `Joins`, `Select` das mesmas colunas, filtros de filme/cinema/dia e `Order("s.data_hora_inicio, s.id")`; `avisarSobreSessoesOrfas` com `Count`; `Criar`/`Atualizar`/`Cancelar` em `EmTransacao` com o fato; `SalaOcupada` por `Raw(...).Scan(&bool)` mantendo a expressão `INTERVAL` (sem equivalente nativo); `BuscarPorID` com `dinheiroDeTexto` e `.UTC()`. `statusVisiveis()` passa a ser usado com `IN ?`.
- [X] T012 [US1] Verificar o spike dos riscos de research D2/D6 antes de seguir: rodar T010/T011 contra um Postgres e confirmar (i) `[]byte`/`nil` → `jsonb`/`NULL`, (ii) `numeric` → `string`, (iii) `ON CONFLICT DO NOTHING` sem erro quando há conflito, (iv) `SKIP LOCKED` realmente presente no SQL emitido. Se algum falhar, aplicar o plano B registrado em research (`*string`/`sql.NullString`, `Raw`) e anotar a mudança em research.md.
- [X] T013 [US1] Atualizar `cmd/catalogo/main.go`: `postgres.NovoPool` → `postgres.Abrir`; `defer banco.Fechar()` (encerramento ordenado, FR-011 — conferir manualmente que `SIGTERM` fecha o banco depois do servidor HTTP, como hoje); repositórios e `CaixaDeSaida` recebem o `*Banco`; `health.Handler(banco)`. Ao fim de T013 `go build ./...` e `go vet ./...` devem passar.
- [X] T014 [US1] Adaptar o harness de `test/integration/` à nova API **sem tocar nas asserções**: em `main_test.go`, `pool` passa de `*pgxpool.Pool` a `*postgres.Banco` (criado por `Abrir`); um `*sql.DB` (`banco.SQL()`) serve às chamadas diretas — converter `pool.Exec/QueryRow` para `ExecContext/QueryRowContext` em `main_test.go`, `leitura_atual_test.go`, `sessoes_test.go`, `caixa_de_saida_test.go`, `performance_test.go`, `planos_consulta_test.go` e demais usos (`grep -rn "pool\." test/integration`); os construtores `pgadapter.Novo...Repository(pool)` passam o `*Banco`. `aplicarMigracoes` pode seguir com `pgxpool` apenas para preparar o banco.
- [X] T015 [US1] Fazer `TestConsultasUsamOsIndices` (`test/integration/planos_consulta_test.go`) executar o **SQL emitido pelo adaptador GORM** em vez de SQL escrito no teste (research D8). Mecanismo: `Abrir` aceita opções funcionais (`...func(*gorm.Config)`); o teste abre o seu `Banco` com um `logger.Interface` próprio cujo `Trace` guarda o SQL e os argumentos de cada consulta, executa `Listar` de filmes e de salas e `Consultar` de sessões (inclusive com filtro de filme, cinema e dia) e roda `EXPLAIN` no SQL capturado, mantendo a mesma lista de índices esperados. É um ponto de injeção só para teste em código de produção; justificar em comentário (princípio VII: sem ele, o edge case de índices da spec não é verificável). Se algum índice deixar de ser usado, corrigir a consulta no adaptador (ex.: `= ANY(?)`), nunca afrouxar o teste.
- [X] T016 [US1] Adicionar `test/integration/caixa_de_saida_concorrencia_test.go` (build tag `integration`): enfileirar N fatos (ex.: 50) e drenar com duas ou mais goroutines chamando `Drenar` ao mesmo tempo, cada uma com um `publicar` que registra os `MessageID` recebidos; afirmar que a união é exatamente os N fatos, sem repetição entre as goroutines e sem bloqueio de uma pela outra (SC-002, FR-004, princípio VI). Hoje esse teste não existe — rodá-lo antes sobre o código pgx (`git stash` da troca) para confirmar que ele passa na linha de base.
- [X] T017 [US1] Rodar `make test` e `make test-integration`; tudo verde, com o diff de testes limitado a T014–T016 (sem asserção enfraquecida — SC-001). Conferir em especial `caixa_de_saida_test.go`, o novo `caixa_de_saida_concorrencia_test.go` (duas réplicas, SC-002) e `escrita_salas_sessoes_test.go` (rollback conjunto de sala/sessão e fato).

**Checkpoint**: US1 entregue; o serviço funciona de ponta a ponta sobre o GORM.

---

## Phase 4: User Story 2 — Esquema preservado, migrações valem (Priority: P2)

**Goal**: nenhum DDL emitido pelo serviço; banco existente opera sem migração de dados.

**Independent Test**: [quickstart](./quickstart.md) §3 — subir a versão nova sobre o volume populado pela anterior.

- [X] T018 [US2] Confirmar por busca que nada emite DDL: `grep -rniE "AutoMigrate|Migrator\(|CREATE TABLE|ALTER TABLE" internal cmd` não deve achar nada fora de `migrations/`; registrar o resultado no PR/commit. (Os testes de integração que fazem `ALTER TABLE` em `sessoes_test.go` são do teste, não do serviço.)
- [X] T019 [US2] Garantir que o teste existente de `search_path` do harness (`test/integration/main_test.go`) continua verificando a conexão do **próprio `Banco`** (`current_setting('search_path') = 'catalogo'` e nenhuma tabela em `public`), e que `make test-integration` o inclui (FR-007).
- [X] T020 [US2] Executar o roteiro do [quickstart](./quickstart.md) §3 (banco populado pela versão anterior → versão nova, sem recriar volume) e anotar o resultado; confirma SC-004 e FR-008.

---

## Phase 5: User Story 3 — Fronteira núcleo × infraestrutura (Priority: P3)

**Goal**: o núcleo não conhece o GORM, e isso é verificado mecanicamente.

**Independent Test**: `make lint` falha se `internal/domain` ou `internal/usecase` importar `gorm.io`.

- [X] T021 [P] [US3] Acrescentar `gorm.io` à regra `nucleo-sem-adaptadores` de `.golangci.yml` (`desc`: "biblioteca de persistência é detalhe de adaptador"), ao lado de `github.com/jackc/pgx`.
- [X] T022 [US3] Provar que a regra morde: adicionar temporariamente um `import _ "gorm.io/gorm"` em `internal/usecase/`, ver `make lint` falhar com a mensagem do depguard e reverter; depois `make lint` limpo (SC-005, FR-010). Conferir também que `go test ./internal/domain/... ./internal/usecase/...` roda sem banco.

---

## Phase 6: Polish & Cross-Cutting

- [ ] T023 [P] _(parcial: §1, §2 e §3 feitos; falta o §4, o fluxo ponta a ponta em `docker compose`, e o build da imagem já foi verificado; medição de SC-006 feita: sem diferença além do ruído — ver relatório)_ Percorrer o [quickstart](./quickstart.md) §1–§4 inteiro e registrar o resultado; incluir uma medição simples antes/depois da listagem da grade de sessões e da reserva em ambiente local (SC-006): medir primeiro no commit anterior à troca (`git stash`/checkout), depois na versão nova, com o mesmo volume de `performance_test.go`.
- [X] T024 [P] Revisar comentários nos arquivos tocados: manter apenas os que explicam decisão (por que a publicação fica dentro da transação, por que `SalaOcupada` usa `Raw`, por que `Select`/mapa nos updates) e remover os que descreviam `pgx`/`pgxpool` e ficaram obsoletos.
- [X] T025 Atualizar a spec 001 do catálogo para refletir a persistência por GORM (decisão do mantenedor, plan.md §"Divergências resolvidas"): em `specs/001-catalogo-sessoes-reserva/research.md` reescrever a decisão de persistência (linhas ~25–31: de `pgx/v5` com SQL à mão para GORM sobre o driver pgx, registrando que o pedido do mantenedor superou a rejeição original e que parte do SQL continua à mão, como `SalaOcupada`) e a "Consequência" sobre `preco_base` (hoje `pgtype.Numeric`; passa a trafegar como texto e converter por `big.Rat`); em `plan.md` (linha ~18) ajustar a dependência; em `tasks.md` apenas anotar, sem desmarcar, que T002 foi superada pela 002. Só depois de T017 verde, para descrever o que o código de fato faz (princípio IX).
- [ ] T026 Commitar direto na `master` (regra do mantenedor — sem branch), em commits pequenos seguindo Conventional Commits em inglês (ex.: `refactor(catalogo): persist through GORM instead of raw pgx`). Push só se pedido.

---

## Dependencies & Execution Order

- **Setup (T001)** → **Foundational (T002–T006)** → **US1** → **US2/US3** → **Polish**.
- Foundational: T002 e T003 podem andar juntos ([P] entre si e com T004); T005 depende de T002+T003; T006 depois de T002.
- US1: T007, T008, T009, T010 em paralelo; T011 depende de T010; T012 depende de T010+T011; T013 depende de T007–T011; T014, T015 e T016 dependem de T013; T017 fecha.
- US2 e US3 só dependem de US1 concluída; entre si são independentes (T021 pode começar já após T001).

### Parallel example (US1)

```
T007 filme_repository.go    T008 cinema_repository.go    T009 sala_repository.go    T010 outbox.go
```

## Implementation Strategy

**MVP = US1.** Entregar T001–T017 deixa o serviço inteiro no GORM com a suíte verde; US2 e US3 são verificações e travas (T018–T022) que não mudam comportamento. Como o pacote só compila ao fim de T013, a validação intermediária é o spike T012 num Postgres de teste, não `go build`.

## Notes

- As três divergências levantadas foram decididas pelo mantenedor e estão em [plan.md](./plan.md) §"Divergências resolvidas"; qualquer outra encontrada na implementação **não é corrigida** sem perguntar (princípio X).
- A estimativa de conclusão de cada tarefa é aferida no código, não na marcação desta lista (constituição, Fluxo de Desenvolvimento).
