# Tasks: Persistência da Notificação via GORM

**Input**: Design documents from `specs/002-persistencia-gorm/`
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [quickstart.md](./quickstart.md)

**Tests**: não há testes novos de comportamento — a equivalência é provada pelas suítes existentes, sem alterar asserções (SC-001). A fronteira do núcleo (US3, SC-005) é verificada por regra do linter `depguard`, sem teste novo.

**Regras do repositório**: commitar direto na `master`, sem branch de feature. Todos os comandos Go rodam dentro de `ingressos-golang/notificacao/`. Mensagens de commit seguem Conventional Commits, em inglês.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência pendente)
- **[Story]**: história de usuário da spec (US1, US2, US3)

## Phase 1: Setup

- [X] T001 Adicionar `gorm.io/gorm` v1.31.2 e `gorm.io/driver/postgres` v1.6.2 em `go.mod`/`go.sum` (`GOFLAGS=-mod=mod GOPROXY=off go get gorm.io/gorm@v1.31.2 gorm.io/driver/postgres@v1.6.2`; ambos estão no cache local) e confirmar que `github.com/jackc/pgx/v5` deixa de ser `// indirect`. Só baixa os módulos; o `go.mod` final é acertado por `go mod tidy` no T009. Confirmar com `go build ./...` que continua compilando

---

## Phase 2: Foundational (bloqueia todas as histórias)

**Purpose**: a conexão GORM e os modelos, de que os dois repositórios dependem.

- [X] T002 Reescrever `internal/adapter/postgres/postgres.go`: `type Banco struct{ db *gorm.DB; sql *sql.DB }` e `Conectar(ctx, url) (*Banco, error)` conforme research D1 — `pgx.ParseConfig(url)` (erro `DATABASE_URL malformada: %w`), `RuntimeParams["search_path"] = Schema`, `stdlib.OpenDB(*cfg)`, `SetMaxOpenConns(max(4, runtime.NumCPU()))`, `gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{SkipDefaultTransaction: true, Logger: logger.Discard, DisableAutomaticPing: true})`, `PingContext` (erro `alcançar o banco: %w`, fechando o `*sql.DB` se falhar). Métodos: `Fechar()`, `Verificar(ctx) error`, `DB() *gorm.DB`, `SQL() *sql.DB`. Manter `const Schema = "notificacao"`
- [X] T003 [P] Criar `internal/adapter/postgres/modelos.go` conforme data-model.md: `ingressoRow` (`TableName() "ingressos_emitidos"`) e `avisoRow` (`TableName() "registros_notificacao"`) com tags `gorm:"column:...;primaryKey"`, `UtilizadoEm *time.Time`, `Detalhes *string`, sem `default:`/auto timestamps/associações; funções `paraLinha(ingresso.Ingresso) ingressoRow`, `(ingressoRow) paraDominio() ingresso.Ingresso` e `avisoParaLinha(aviso.Registro) avisoRow` (`Detalhes` nil quando vazio, `Status` = `string(reg.Desfecho)`)

**Checkpoint**: o pacote `postgres` ainda não compila por inteiro (repositórios antigos) até a Phase 3; T002 e T003 são o alicerce.

---

## Phase 3: User Story 1 — Comportamento de negócio idêntico (Priority: P1) 🎯 MVP

**Goal**: toda leitura/escrita passa pelo GORM e o comportamento observável não muda.

**Independent Test**: `make test` e `make test-integration` verdes sem alterar nenhuma asserção; fluxo do quickstart §3.

- [X] T004 [US1] Reescrever `internal/adapter/postgres/ingressos.go`: `type Ingressos struct{ DB *gorm.DB }` mantendo os 4 métodos da porta `usecase.Ingressos`, conforme research D2/D5/D7 e data-model.md — `CriarSeAusente` (`Clauses(clause.OnConflict{Columns: reserva_id, DoNothing: true}).Create`; `RowsAffected==0` → `Take` por `reserva_id`; `gorm.ErrRecordNotFound` → `usecase.ErrNaoEncontrado`), `Utilizar` (`Where("id = ? AND status = ?", id, "VALIDO").Updates(map[string]any{"status": "UTILIZADO", "utilizado_em": agora})`, `RowsAffected==1`), `BuscarPorID`, `ListarPorUsuario` (filtro de status só quando `!= ""`, `Order("criado_em DESC, id DESC")`, slice inicializado vazio). Toda chamada com `WithContext(ctx)`. Manter as mensagens de erro de hoje ("inserir ingresso", "dar baixa no ingresso", "buscar ingresso", "buscar ingresso por reserva", "listar ingressos"). Remover `escaneavel`/`ler` e os imports de `pgx`/`pgxpool`
- [X] T005 [P] [US1] Reescrever `internal/adapter/postgres/avisos.go`: `type Avisos struct{ DB *gorm.DB }`, `Registrar` = `DB.WithContext(ctx).Create(&linha)` usando `avisoParaLinha`; erro embrulhado como `registrar aviso: %w`
- [X] T006 [US1] Ajustar `cmd/notificacao/main.go`: `pool, err := postgres.Conectar(...)` → `banco`; `defer banco.Fechar()`; `postgres.Ingressos{DB: banco.DB()}` e `postgres.Avisos{DB: banco.DB()}`; readiness `prontidao.Registrar("postgres", banco.Verificar)`
- [X] T007 [US1] Adaptar o harness em `test/integration/ambiente_test.go`: campo `Pool *pgxpool.Pool` → `Banco *postgres.Banco`; `postgres.Conectar` + `t.Cleanup(banco.Fechar)`; `Ingressos: postgres.Ingressos{DB: banco.DB()}`, `Avisos: postgres.Avisos{DB: banco.DB()}`; `aplicarMigracao(t, banco)` usando `banco.SQL().ExecContext`; `contarIngressos` e `TestSchemaProprio` com `a.Banco.SQL().QueryRowContext(...).Scan(...)` (mesmas consultas, mesmas asserções); remover o import de `pgxpool`
- [X] T008 [P] [US1] Adaptar `test/integration/emissao_test.go` (`a.Pool.Close()` → `a.Banco.Fechar()`, `a.Pool.QueryRow(ctx, ...)` → `a.Banco.SQL().QueryRowContext(ctx, ...)`) e `test/integration/aviso_test.go` (idem `QueryRow`). Nenhuma asserção muda
- [X] T009 [US1] Rodar `GOFLAGS=-mod=mod GOPROXY=off go mod tidy` (confirmar que `pgx/v5` ficou direto e que `go.sum` ganhou as dependências do GORM), depois `go build ./... && go vet ./... && make test` e depois `go vet -tags=integration ./test/integration/...`; tudo deve compilar e passar. Corrigir qualquer sobra de `pgx`/`Pool` (`grep -rn "pgxpool\|\.Pool" --include=*.go .` não deve achar nada fora do cache)
- [X] T010 [US1] Rodar `make test-integration` (Docker) e confirmar verde (inclui `TestListagemDe200IngressosRespondeDentroDoPrazo` e `TestVeredictoDaPortariaSobPico`, que servem de evidência para SC-006): em particular emissão idempotente/concorrente (`emissao_test`, `vazao_test`), baixa concorrente (`validacao_test`), listagem (`vazao_listagem_test`), aviso (`aviso_test`) e falha transitória → fila morta. Se o `RowsAffected` do `ON CONFLICT DO NOTHING` divergir do esperado (research D2, risco), trocar `CriarSeAusente` para `Exec`/`Raw` do próprio GORM com o mesmo SQL de antes e registrar a decisão em research.md

**Checkpoint**: US1 completa e demonstrável — MVP.

---

## Phase 4: User Story 2 — Esquema preservado, migrações valem (Priority: P2)

**Goal**: operar sobre banco existente sem migração de dados e sem o serviço emitir DDL.

**Independent Test**: quickstart §4 (banco populado pela versão anterior) e `docker compose up` em banco vazio.

- [X] T011 [US2] Verificar por inspeção que não há `AutoMigrate`/`Migrator()` no código (`grep -rn "AutoMigrate\|Migrator" --include=*.go .` vazio) e que `migrations/` e `Makefile` (`MIGRATE_URL` com `search_path=notificacao`) permanecem sem diff (`git status --short migrations Makefile` vazio)
- [X] T012 [US2] Validação manual do quickstart §3 e §4: com a versão anterior (imagem ou binário do commit anterior, p.ex. via `git worktree` temporário) emitir alguns ingressos, subir o serviço novo sobre o mesmo banco e confirmar listagem, consulta e validação dos registros antigos; `docker compose up --build` em banco vazio sobe saudável (`/health` do notificacao). Reportar o resultado; se Docker/compose não estiver disponível, dizer isso em vez de marcar como feito — feito: (a) Postgres real semeado por SQL como o adaptador antigo gravava: lista, filtro, vazio≠nulo, inexistente, baixa 1x, reemissão devolve o original, `codigo_qr` repetido falha, CHECK de aviso recusa; (b) `docker compose up --build notificacao` (com postgres, rabbitmq, keycloak e `migrate-notificacao`): migração aplicada, `/health/ready` = pronto, serviço no ar. Limite: só o `notificacao` e suas dependências subiram, não a plataforma inteira

---

## Phase 5: User Story 3 — Fronteira núcleo/infraestrutura mantida (Priority: P3)

**Goal**: o núcleo não conhece GORM nem driver, verificado automaticamente.

**Independent Test**: `go test ./test/...` passa; introduzir temporariamente um import de `gorm.io/gorm` em `internal/usecase` faz o teste falhar.

- [X] T013 [P] [US3] Acrescentar o linter `depguard` ao `.golangci.yml` com a regra `nucleo-sem-adaptadores`, no padrão de `../catalogo/.golangci.yml`: `files` = `**/internal/domain/**` e `**/internal/usecase/**`; `deny` = `github.com/oseias/ingressos-golang/notificacao/internal/adapter`, `.../internal/platform`, `gorm.io`, `github.com/jackc/pgx`, `github.com/rabbitmq/amqp091-go`, `net/http`, `go.opentelemetry.io/otel`, cada um com `desc`
- [X] T014 [US3] Provar que a regra morde: acrescentar temporariamente `_ "gorm.io/gorm"` a um arquivo de `internal/usecase`, ver `golangci-lint run ./internal/usecase/...` reprovar com `depguard`, desfazer e ver passar. Não commitar a alteração temporária

---

## Phase 6: Polish

- [X] T015 [P] Rodar `make lint` (golangci-lint) e `gofmt -l .`; corrigir apenas o que a mudança introduziu — `gofmt`, `go vet` e `golangci-lint` rodados: 10 achados, todos em código anterior à troca (errcheck/bodyclose/staticcheck/unused em `cmd/`, `internal/adapter/http`, `codigo`, `usecase`); nenhum nos arquivos do adaptador `postgres`
- [X] T016 [P] Atualizar `README.md` do serviço se citar `pgx`/`pgxpool` como mecanismo de persistência (`grep -n "pgx" README.md`); não editar os artefatos da spec 001 (decisão registrada no plan.md)
- [X] T017 Rodar o quickstart §1–§2 do início ao fim e confirmar SC-001..SC-006; medir informalmente a latência de emissão/listagem (SC-006) comparando com a versão anterior, ou declarar que não foi medida — §1–§2 verdes; §3 no compose: `pagamento.sucesso` publicado 2× para a mesma reserva → 1 ingresso e 1 aviso; validação 200 / 409 (já utilizado) / 404 (código forjado); listagem por JWT mais recente primeiro, com filtro `status`, 401 sem token. SC-006 sem medição própria (só os testes com prazo, que passaram)
- [X] T018 Commitar na `master` (sem criar branch) com `git add` dos caminhos de `notificacao/` alterados e de `notificacao/specs/002-persistencia-gorm/`; mensagem sugerida: `refactor(notificacao): persist and read through GORM instead of raw pgx`. Sem `push` — feito na `master`, sem branch e sem push: `f79b452` (troca para GORM) e `1763c7a` (`depguard` no lugar do teste de arquitetura)

---

## Dependencies & Execution Order

- **Phase 1 → Phase 2 → Phase 3 (US1)**: sequencial. US2 e US3 só fazem sentido depois de US1.
- **Phase 2**: T002 antes de T004–T008; T003 em paralelo com T002 (arquivos diferentes), mas T004/T005 precisam dos dois.
- **US1**: T004 e T005 em paralelo (arquivos diferentes) após T002/T003; T006 após T004/T005; T007 antes de T008; T009 após T006–T008; T010 após T009.
- **US2** (T011, T012) depende de US1. **US3**: T013 pode ser escrito em paralelo a US1 (não depende do adaptador), mas só prova algo útil depois da troca; T014 após T013.
- **Polish**: após todas as histórias.

### Parallel Example

```
Após T002: T003 || (nada)         # modelos.go
Após T003: T004 || T005           # ingressos.go || avisos.go
Após T007: T008                   # emissao_test.go || aviso_test.go
Qualquer momento: T013            # .golangci.yml
```

## Implementation Strategy

1. **MVP = US1** (T001–T010): ao fim, o serviço roda sobre GORM e as suítes existentes provam a equivalência.
2. US3 (T013–T014) é barata e pode entrar logo em seguida; US2 é verificação, sem código.
3. Parar e validar a cada checkpoint; qualquer divergência entre código e spec vai ao mantenedor (princípio IV), não é corrigida em silêncio.
