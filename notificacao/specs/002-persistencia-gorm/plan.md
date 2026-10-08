# Implementation Plan: Persistência da Notificação via GORM

**Branch**: `master` (nenhuma branch de feature — regra do mantenedor) | **Date**: 2026-10-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/002-persistencia-gorm/spec.md`

## Summary

Trocar o acesso a dados do `Servico-Notificacao` de `pgxpool` com SQL à mão para o
GORM, sem alterar nada que um consumidor observe. A troca é inteira dentro de
`internal/adapter/postgres/` (175 linhas, 3 arquivos), mais a fiação em
`cmd/notificacao/main.go` e o harness dos testes de integração. As portas
`usecase.Ingressos` e `usecase.Avisos` não mudam.

Decisões que moldam o desenho (detalhe em [research.md](./research.md)):

1. **O GORM entra por cima do mesmo driver.** `gorm.io/driver/postgres` usa o pgx por
   baixo; a conexão continua montada a partir de `pgx.ParseConfig` com
   `search_path=notificacao` em `RuntimeParams`, entregue ao GORM como `*sql.DB` (D1).
2. **Idempotência e baixa continuam no banco**: `INSERT ... ON CONFLICT (reserva_id) DO
   NOTHING` via `clause.OnConflict` e `UPDATE ... WHERE status = 'VALIDO'` com
   `RowsAffected`. Nada de `Save`, hooks ou lock otimista (D2).
3. **Modelos GORM vivem no adaptador, não no domínio.** Structs privadas com tags
   `gorm:"column:..."` e `TableName()` sem schema (D3).
4. **Esquema só pela migração.** Sem `AutoMigrate` (D4).
5. **Teste de arquitetura passa a existir** (`test/arquitetura_test.go`): o serviço não
   tem um hoje, e o SC-005 pede verificação automática (D6).

Nenhum contrato muda (REST, AMQP), então esta feature **não tem `contracts/`**.

## Technical Context

**Language/Version**: Go 1.25 (`go.mod`; toolchain local 1.27)

**Primary Dependencies**: acrescenta `gorm.io/gorm` v1.31.2 e `gorm.io/driver/postgres`
v1.6.2 (ambos já no cache de módulos local). `github.com/jackc/pgx/v5` deixa de ser
`// indirect` (passa a ser usado direto: `pgx.ParseConfig` e `stdlib`). O pacote
`pgxpool` sai do código de produção.

**Storage**: PostgreSQL 16, schema `notificacao`, as mesmas 2 tabelas e a 1 migração
existente. RabbitMQ intocado.

**Testing**: `go test -race ./...` (domínio e casos de uso, sem banco) e
`make test-integration` (Testcontainers com Postgres e RabbitMQ reais — as mesmas
suítes, asserções inalteradas; só o harness troca de `pgxpool` para o novo `Banco`).

**Target Platform**: contêiner Linux (Dockerfile inalterado; `go mod download`
resolve os dois módulos novos).

**Project Type**: microsserviço (REST + consumidor AMQP).

**Performance Goals**: iguais aos da spec 001. `SkipDefaultTransaction: true` evita
ida-e-volta extra por escrita; sem `PrepareStmt` (o pgx já cacheia).

**Constraints**: emissão idempotente por `reserva_id` decidida pelo banco; baixa
atômica condicional; nenhum DDL emitido pelo serviço; logger do GORM descartado (não
vaza SQL/parâmetros nem "record not found" como ruído).

**Scale/Scope**: 3 arquivos de adaptador reescritos, 1 ajuste em `main.go`, 1 teste
novo de arquitetura, ~8 chamadas ajustadas nos testes de integração.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

A constituição do serviço (`notificacao/.specify/memory/constitution.md`) ainda é o
template não preenchido; vale a do workspace (`../.specify/memory/constitution.md`, v1.0.0).

| Princípio | Veredito | Evidência |
|---|---|---|
| I. Complexidade só se necessária ou pedida | **PASS** | A troca foi pedida. Sem camada de repositório nova, sem generics, sem `AutoMigrate`, sem hooks, sem associações. O teste de arquitetura novo é o mínimo para cumprir SC-005 (≈40 linhas, mesmo padrão do `estoque`) |
| II. Domínio e API têm teste automatizado | **PASS** | Nenhum comportamento muda; as suítes unitárias e de integração existentes são o critério de aceite (`emissao_test`, `validacao_test`, `aviso_test`, `vazao*_test`) |
| III. O código é a fonte da verdade | **PASS** | Comportamentos a preservar lidos de `ingressos.go`, `avisos.go`, `postgres.go` e dos testes, não da spec 001 |
| IV. Divergência código↔spec é pergunta | **ATENÇÃO** | Ver "Perguntas ao mantenedor" |

**Re-avaliação pós-Fase 1**: o desenho não introduziu violação. Nenhuma entrada em
Complexity Tracking.

### Perguntas ao mantenedor (princípio IV)

1. **Como verificar a fronteira do núcleo (FR-010/SC-005)**: hoje o `notificacao` não
   tem nenhuma verificação automática (o `CLAUDE.md` já registra que `pagamento` e
   `notificacao` não têm). O workspace usa duas formas: o `test/arquitetura_test.go` do
   `estoque` (`go list -json`, roda em `make test`) e a regra `depguard`
   `nucleo-sem-adaptadores` do `catalogo` (`.golangci.yml`, roda em `make lint`). Este
   plano **escolhe o teste do `estoque`** (≈40 linhas, falha já em `make test`, que é o
   que o SC-005 pede); o núcleo hoje está limpo, então passa de primeira. Se preferir a
   regra `depguard`, a tarefa T013 troca de arquivo (`.golangci.yml`) sem afetar o resto.
2. **Spec 001**: `plan.md`/`research.md`/`data-model.md` da 001 citam `pgx` com SQL à
   mão. Após esta feature deixam de descrever o código. O plano assume que a 001 fica
   como registro histórico e **não a edita**; nada nas tarefas depende da resposta.

## Project Structure

### Documentation (this feature)

```text
specs/002-persistencia-gorm/
├── plan.md              # Este arquivo
├── spec.md
├── research.md          # Fase 0 — decisões D1..D7
├── data-model.md        # Fase 1 — tabela → modelo GORM e consultas
├── quickstart.md        # Fase 1 — como validar a equivalência
├── checklists/requirements.md
└── tasks.md             # Fase 2 — /speckit-tasks
```

Sem `contracts/`: nenhum contrato externo muda.

### Source Code (notificacao/)

```text
notificacao/
├── cmd/notificacao/main.go              # postgres.Conectar → *Banco; Fechar/Verificar; repositórios recebem banco.DB()
├── internal/adapter/postgres/
│   ├── postgres.go                      # Banco{db *gorm.DB, sql *sql.DB}: Conectar, Fechar, Verificar, DB(), SQL()
│   ├── modelos.go                       # NOVO — ingressoRow, avisoRow (+ TableName) e conversões de/para o domínio
│   ├── ingressos.go                     # Ingressos{DB *gorm.DB} sobre GORM; mesma porta usecase.Ingressos
│   └── avisos.go                        # Avisos{DB *gorm.DB} sobre GORM; mesma porta usecase.Avisos
├── test/
│   ├── arquitetura_test.go              # NOVO — núcleo não importa adapter/platform/gorm/pgx/amqp
│   └── integration/*_test.go            # harness: Pool *pgxpool.Pool → Banco; QueryRow → SQL().QueryRowContext
├── go.mod / go.sum                      # + gorm, driver postgres; pgx deixa de ser indirect
└── internal/domain, internal/usecase    # NÃO mudam
```

**Structure Decision**: mesma estrutura hexagonal; a mudança fica toda no adaptador
`postgres`. `modelos.go` é novo para que as structs com tags GORM não vazem para fora
do pacote e a conversão fique em um lugar só.

## Complexity Tracking

Sem violações a justificar.
