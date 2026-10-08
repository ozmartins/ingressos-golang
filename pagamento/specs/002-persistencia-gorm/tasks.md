# Tasks: Persistência do Pagamento via GORM

**Input**: Design documents from `specs/002-persistencia-gorm/`
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [quickstart.md](./quickstart.md)

**Tests**: não há testes novos de comportamento — a equivalência é provada pelas suítes existentes, sem alterar asserções (SC-001). A fronteira do núcleo (US3, SC-005) é verificada por regra do linter `depguard`, sem teste novo. O único código de teste tocado é o harness de integração, que troca `*pgxpool.Pool` por `*postgres.Banco`.

**Regras do repositório**: commitar direto na `master`, sem branch de feature. Todos os comandos Go rodam dentro de `ingressos-golang/pagamento/`. Mensagens de commit seguem Conventional Commits, em inglês.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência pendente)
- **[Story]**: história de usuário da spec (US1, US2, US3)

## Phase 1: Setup

- [X] T001 Adicionar `gorm.io/gorm` v1.31.2 e `gorm.io/driver/postgres` v1.6.2 em `go.mod`/`go.sum` (`GOFLAGS=-mod=mod GOPROXY=off go get gorm.io/gorm@v1.31.2 gorm.io/driver/postgres@v1.6.2`; ambos estão no cache local de módulos). Só baixa os módulos; o `go.mod` final é acertado por `go mod tidy` no T011. Confirmar com `go build ./...` que continua compilando

---

## Phase 2: Foundational (bloqueia todas as histórias)

**Purpose**: a conexão GORM e o modelo, de que o repositório depende.

- [X] T002 Renomear `internal/adapter/postgres/pool.go` para `postgres.go` (`git mv`) e reescrevê-lo: `type Banco struct{ db *gorm.DB; sql *sql.DB }` e `Conectar(ctx, url) (*Banco, error)` conforme research D1 — `pgx.ParseConfig(url)` (erro `DATABASE_URL malformada: %w`), `RuntimeParams["search_path"] = Schema`, `stdlib.OpenDB(*cfg)`, `SetMaxOpenConns(max(4, runtime.NumCPU()))`, `gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{SkipDefaultTransaction: true, Logger: logger.Discard, DisableAutomaticPing: true})`, `PingContext` (erro `alcançar o banco: %w`, fechando o `*sql.DB` se falhar). Métodos: `Fechar()`, `Verificar(ctx) error`, `DB() *gorm.DB`, `SQL() *sql.DB`. Manter `const Schema = "pagamento"`. Modelo de referência: `../notificacao/internal/adapter/postgres/postgres.go`
- [X] T003 [P] Criar `internal/adapter/postgres/modelos.go` conforme data-model.md e research D3/D4: `transacaoRow` (`TableName() "transacoes_pagamento"`) com tags `gorm:"column:...;primaryKey"`, `ValorTotal string`, `FormaPagamento`/`CodigoTransacaoGateway`/`MotivoFalha` como `*string`, `PagoEm *time.Time`, sem `default:`/auto timestamps/associações; `paraLinha(transacao.Transacao) transacaoRow` (`""` → `nil` nos três `*string`) e `(transacaoRow) paraDominio() transacao.Transacao` (`nil` → `""`)
- [X] T004 Spike dos riscos R1–R3 (research D2, D3, D6) contra Postgres real, antes de escrever o repositório: num teste descartável (fora do repositório, ou em arquivo `_spike_test.go` apagado ao fim) subir `postgres:16-alpine`, aplicar as duas migrações e verificar com o `Banco` do T002 e o `transacaoRow` do T003 que (R1) `Clauses(OnConflict{reserva_id, DoNothing}, Returning{}).Create` dá `RowsAffected` 1 na criação (struct preenchida pelo `RETURNING`, `valor_total` normalizado) e 0 no conflito, sem erro; (R2) `ValorTotal` faz ida-e-volta exata de `"10.50"`, `"0.01"` e `"99999999.99"` por `Create` e `Find`; (R3) `UPDATE ... WHERE id IN (subselect ... FOR UPDATE SKIP LOCKED) RETURNING` preenche o slice via `Clauses(Returning{}).Model(&linhas)`. Anotar o resultado de cada um em research.md (decisão tomada ou fallback para `Raw(...).Scan` / `Select("valor_total::text")` / `gorm.Expr("?::decimal", v)`). Não deixar o spike no repositório — feito: R1, R2 e R3 passaram com GORM nativo, sem `Raw` nem casts; resultado em research.md ("Resultado do spike"); teste descartável removido

**Checkpoint**: o pacote `postgres` ainda não compila por inteiro (repositório antigo) até a Phase 3; T002–T004 são o alicerce.

---

## Phase 3: User Story 1 — Comportamento de negócio idêntico (Priority: P1) 🎯 MVP

**Goal**: toda leitura/escrita passa pelo GORM e o comportamento observável não muda.

**Independent Test**: `make test` e `make test-integration` verdes sem alterar nenhuma asserção; fluxo do quickstart §3.

- [X] T005 [US1] Reescrever `internal/adapter/postgres/transacoes.go`: `type Repositorio struct{ db *gorm.DB }` e `NovoRepositorio(db *gorm.DB) *Repositorio`, implementando os 10 métodos de `usecase.Repositorio` (`internal/usecase/ports.go`) conforme data-model.md e research D2/D6/D7, sempre com `WithContext(ctx)` e `Updates(map[string]any{...})` (nunca struct): `CriarSeAusente` (resultado do T004/R1; criada → devolve a linha do `RETURNING`, conflito → `BuscarPorReserva` e `(false, atual)`), `BuscarPorReserva` (`gorm.ErrRecordNotFound` → `usecase.ErrNaoEncontrada`), `RegistrarEscolha` e `Finalizar` (condicionais ao estado de origem; `RowsAffected==0` → `usecase.ErrJaFinalizada`; vazios gravados como `nil`), `ReivindicarCobranca` (`RowsAffected==1`), `LiberarCobranca`, `MarcarAnunciado` (sem checar linhas, como hoje), `AguardandoCobranca`, `AnunciosPendentes` (`PENDENTE_VERIFICACAO` fora, de propósito) e `CancelarEsperasVencidas` (resultado do T004/R3). Remover `colunas`, `scan`, `linha`, `formaOuNulo`, `consultarLista` e os imports de `pgx`/`pgxpool`. Preservar os comentários que explicam o porquê (condição de estado, forma nula só no cancelamento, `PENDENTE_VERIFICACAO` fora do anúncio). O método `Ping` sai: a prontidão passa a usar `Banco.Verificar`
- [X] T006 [US1] Ajustar `cmd/pagamento/main.go`: `postgres.Abrir` + `pool.Ping` → `banco, err := postgres.Conectar(ctx, cfg.DatabaseURL)`; `defer banco.Fechar()`; `repo := postgres.NovoRepositorio(banco.DB())`; `prontidao.Registrar("banco", banco.Verificar)`
- [X] T007 [US1] Adaptar o harness em `test/integration/ambiente_test.go`: campo `Pool *pgxpool.Pool` → `Banco *postgres.Banco`; `postgres.Conectar` + `t.Cleanup(banco.Fechar)`; `Repo: postgres.NovoRepositorio(banco.DB())`; `aplicarMigracao(t, banco)` usando `banco.SQL().ExecContext` (as duas migrações, na ordem de hoje); remover o import de `pgxpool`
- [X] T008 [P] [US1] Adaptar as consultas diretas ao banco nos testes: `test/integration/idempotencia_test.go:34`, `test/integration/vazao_test.go:46` e `test/integration/esquema_test.go:15,23` — `a.Pool.QueryRow(ctx, sql, args...)` → `a.Banco.SQL().QueryRowContext(ctx, sql, args...)`. Mesmas consultas, mesmas asserções — feito, mais um ponto além dos listados: `ambiente_test.go` (`escolherFormasPendentes`: `Pool.Query` → `Banco.SQL().QueryContext`) e `vazao_test.go` (`a.Repo.Ping` → `a.Banco.Verificar`, já que `Ping` saiu do `Repositorio`)
- [X] T009 [US1] Verificar o contrato de ida-e-volta do `valor_total` fora do spike: confirmar que `idempotencia_test`/`cobranca_test` exercitam um valor com casas decimais; se nenhum teste existente cobrir `10.5` → `10.50`, **perguntar ao mantenedor** antes de acrescentar teste (princípio IV: só ajustar o que a troca exige, sem ampliar escopo) — nenhum teste existente exercita ida-e-volta de `valor_total` com casas decimais; **não acrescentei teste** (ampliaria escopo). O spike T004/R2 provou `10.5`→`10.50`, `0.01` e `99999999.99`. Fica a pergunta ao mantenedor: quer um teste permanente?
- [X] T010 [US1] `go build ./... && go vet ./... && make test` e `go vet -tags=integration ./test/integration/...`; tudo deve compilar e passar. Corrigir qualquer sobra (`grep -rn "pgxpool\|\.Pool\b" --include=*.go .` não deve achar nada)
- [X] T011 [US1] `GOFLAGS=-mod=mod GOPROXY=off go mod tidy`; conferir que `pgx/v5` segue direto e `go.sum` ganhou as dependências do GORM; repetir `go build ./... && make test`
- [X] T012 [US1] `make test-integration` (Docker) e confirmar verde: em particular registro idempotente e concorrente (`idempotencia_test`), rajada/concorrência (`vazao_test`), cobrança e anúncio (`cobranca_test`), schema próprio (`esquema_test`) e as demais suítes de `test/integration`. Se algum comportamento divergir, corrigir o adaptador, nunca a asserção; se for preciso `Raw` do GORM, registrar em research.md

**Checkpoint**: US1 completa e demonstrável — MVP.

---

## Phase 4: User Story 2 — Esquema preservado, migrações valem (Priority: P2)

**Goal**: operar sobre banco existente sem migração de dados e sem o serviço emitir DDL.

**Independent Test**: quickstart §4 (banco populado pela versão anterior) e `docker compose up` em banco vazio.

- [X] T013 [US2] Verificar por inspeção que não há `AutoMigrate`/`Migrator()` no código (`grep -rn "AutoMigrate\|Migrator" --include=*.go .` vazio) e que `migrations/` e `Makefile` permanecem sem diff (`git status --short migrations Makefile` vazio); `docker-compose.yml` (raiz) também sem diff
- [X] T014 [US2] Validação manual do quickstart §3 e §4: semear o Postgres com transações como o adaptador antigo as gravava (aguardando forma, em processamento, pagas, recusadas, canceladas), subir o serviço novo sobre o mesmo banco e confirmar consulta por reserva, varredura de cobranças, cancelamento de esperas vencidas e anúncio dos resultados; `docker compose up --build pagamento` (com `postgres`, `rabbitmq`, `migrate-pagamento`) em banco vazio sobe saudável (`/api/v1/health/ready`). Reportar o resultado; se Docker/compose não estiver disponível, dizer isso em vez de marcar como feito — feito num projeto compose isolado (`-p pgto-gorm-check`, portas alternativas, volumes próprios, removido ao fim; os volumes `ingressos-golang_*` não foram tocados). Banco semeado como o adaptador antigo gravava (6 transações em estados variados): espera vencida → `CANCELADO`/`RESERVA_EXPIRADA` e anunciada; `PROCESSANDO` → cobrada (`PAGO`, código do gateway gravado); `PAGO` não anunciada → anunciada; `RECUSADO` anunciada, `PENDENTE_VERIFICACAO` e `AGUARDANDO_FORMA` válida → intocadas; `valor_total` preservado (`25.50`, `99999999.99`); `/health/ready` = 200; consulta sem token = 401. Limite: consulta autenticada não exercitada (sem token Keycloak)

---

## Phase 5: User Story 3 — Fronteira núcleo/infraestrutura mantida (Priority: P3)

**Goal**: o núcleo não conhece GORM nem driver, verificado automaticamente.

**Independent Test**: `make lint` passa; introduzir temporariamente um import de `gorm.io/gorm` em `internal/usecase` faz o `depguard` reprovar.

- [X] T015 [P] [US3] Acrescentar o linter `depguard` ao `.golangci.yml` com a regra `nucleo-sem-adaptadores`, no padrão de `../notificacao/.golangci.yml`: `files` = `**/internal/domain/**` e `**/internal/usecase/**`; `deny` = `github.com/oseias/ingressos-golang/pagamento/internal/adapter`, `.../pagamento/internal/platform`, `gorm.io`, `github.com/jackc/pgx`, `github.com/rabbitmq/amqp091-go`, `net/http`, `go.opentelemetry.io/otel`, cada um com `desc`
- [X] T016 [US3] Provar que a regra morde: acrescentar temporariamente `_ "gorm.io/gorm"` a um arquivo de `internal/usecase`, ver `golangci-lint run ./internal/usecase/...` reprovar com `depguard`, desfazer e ver passar. Não commitar a alteração temporária

---

## Phase 6: Polish

- [X] T017 [P] Rodar `make lint` e `gofmt -l .`; corrigir apenas o que a mudança introduziu e reportar separadamente achados anteriores à troca — `golangci-lint run ./...`: 3 achados `errcheck` (`cmd/pagamento/main.go:35,63`, `cmd/publicar/main.go:46`), todos em linhas que a troca não tocou; `gofmt` limpo
- [X] T018 [P] Atualizar `README.md` e `erp-pagamentp.md` do serviço **somente** se descreverem `pgx`/`pgxpool` como mecanismo de persistência (`grep -n "pgx" README.md erp-pagamentp.md`); não editar os artefatos da spec 001 (decisão registrada no plan.md) — `grep pgx` em `README.md` e `erp-pagamentp.md`: nada a atualizar
- [X] T019 Rodar o quickstart §1–§3 do início ao fim e confirmar SC-001..SC-006; declarar explicitamente se a latência (SC-006) não foi medida — §1 e §2 verdes (`make test`; `make test-integration` em 180s); §3 e §4 feitos no T014. SC-006 (latência) **não foi medida**; só os testes com prazo (`vazao_test`) passaram
- [X] T020 Commitar na `master` (sem criar branch) com `git add` dos caminhos de `pagamento/` alterados e de `pagamento/specs/002-persistencia-gorm/`; mensagem sugerida: `refactor(pagamento): persist and read through GORM instead of raw pgx`. Sem `push`

---

## Dependencies & Execution Order

- **Phase 1 → Phase 2 → Phase 3 (US1)**: sequencial. US2 e US3 só fazem sentido depois de US1.
- **Phase 2**: T002 e T003 em paralelo (arquivos diferentes); T004 (spike) depende dos dois e **bloqueia** T005, pois decide `CriarSeAusente`, `CancelarEsperasVencidas` e o tratamento de `valor_total`.
- **US1**: T005 após T004; T006 após T005; T007 antes de T008; T010 após T005–T008; T011 após T010; T012 após T011; T009 pode rodar a qualquer momento após T007.
- **US2** (T013, T014) depende de US1. **US3**: T015 pode ser escrito em paralelo a US1 (não depende do adaptador), mas só prova algo útil depois da troca; T016 após T015.
- **Polish**: após todas as histórias.

### Parallel Example

```
Phase 2:  T002 || T003            # postgres.go || modelos.go
Após T007: T008                   # ajustes de QueryRow nos testes (arquivos diferentes do ambiente)
Qualquer momento: T015            # .golangci.yml
```

## Implementation Strategy

1. **MVP = US1** (T001–T012): ao fim, o serviço roda sobre GORM e as suítes existentes provam a equivalência. O spike (T004) vem antes do repositório porque os três riscos mapeados decidem trechos inteiros do código.
2. US3 (T015–T016) é barata e pode entrar logo em seguida; US2 é verificação, sem código.
3. Parar e validar a cada checkpoint; qualquer divergência entre código e spec vai ao mantenedor (princípio IV), não é corrigida em silêncio.
