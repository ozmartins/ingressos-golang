# Implementation Plan: Persistência do Pagamento via GORM

**Branch**: `master` (nenhuma branch de feature — regra do mantenedor) | **Date**: 2026-10-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/002-persistencia-gorm/spec.md`

## Summary

Trocar o acesso a dados do `Servico-Pagamento` de `pgxpool` com SQL à mão para o GORM,
sem alterar nada que um consumidor observe. A troca é inteira dentro de
`internal/adapter/postgres/` (`pool.go`, `transacoes.go`; `varredura.go` não toca em
banco e fica como está), mais a fiação em `cmd/pagamento/main.go` e o harness dos testes
de integração. A porta `usecase.Repositorio` (10 métodos) não muda.

Decisões que moldam o desenho (detalhe em [research.md](./research.md)):

1. **O GORM entra por cima do mesmo driver.** `gorm.io/driver/postgres` usa o pgx por
   baixo; a conexão é montada com `pgx.ParseConfig` + `search_path=pagamento` em
   `RuntimeParams`, entregue ao GORM como `*sql.DB` via `stdlib.OpenDB` (D1) — o mesmo
   desenho já adotado no `notificacao`.
2. **Toda decisão de concorrência continua no banco**: `INSERT ... ON CONFLICT
   (reserva_id) DO NOTHING RETURNING`, `UPDATE ... WHERE status = ...` com
   `RowsAffected`, e o `UPDATE ... WHERE id IN (SELECT ... FOR UPDATE SKIP LOCKED)
   RETURNING` do cancelamento em lote. Nada de `Save`, hooks ou lock otimista (D2, D6).
3. **Um modelo GORM privado no adaptador** (`transacaoRow`), com `valor_total` como
   `string` e os três campos "vazio = NULL" (forma, código do gateway, motivo) como
   `*string` convertidos nas bordas (D3, D4).
4. **Esquema só pela migração.** Sem `AutoMigrate` (D5).
5. **A fronteira do núcleo passa a ser verificada pelo `depguard`** (regra
   `nucleo-sem-adaptadores` no `.golangci.yml`, como no `catalogo` e no `notificacao`):
   o serviço não tem verificação hoje, e o SC-005 pede que seja automática (D7).

Nenhum contrato muda (REST, AMQP), então esta feature **não tem `contracts/`**.

## Technical Context

**Language/Version**: Go 1.25 (`go.mod`; toolchain local 1.27)

**Primary Dependencies**: acrescenta `gorm.io/gorm` v1.31.2 e `gorm.io/driver/postgres`
v1.6.2 (as mesmas versões do `notificacao`, já no cache de módulos local).
`github.com/jackc/pgx/v5` continua direto (`pgx.ParseConfig`, `stdlib`); o pacote
`pgxpool` sai do código de produção e dos testes.

**Storage**: PostgreSQL 16, schema `pagamento`, a mesma tabela (`transacoes_pagamento`) e
as 2 migrações existentes. RabbitMQ intocado.

**Testing**: `go test -race ./...` (domínio e casos de uso, sem banco) e
`make test-integration` (Testcontainers com Postgres e RabbitMQ reais — as mesmas
suítes, asserções inalteradas; só o harness troca `*pgxpool.Pool` pelo novo `*Banco`).

**Target Platform**: contêiner Linux (Dockerfile inalterado; `go mod download` resolve
os dois módulos novos).

**Project Type**: microsserviço (REST + consumidor AMQP + varredura periódica).

**Performance Goals**: iguais aos da spec 001. `SkipDefaultTransaction: true` evita
ida-e-volta extra por escrita; sem `PrepareStmt` (o pgx já cacheia).

**Constraints**: registro idempotente por `reserva_id` decidido pelo banco; reivindicação
e escolha atômicas e condicionais; `valor_total` ida-e-volta sem float; nenhum DDL
emitido pelo serviço; logger do GORM descartado (não vaza SQL/parâmetros nem "record not
found" como ruído).

**Scale/Scope**: 2 arquivos de adaptador reescritos (+1 novo, `modelos.go`), 1 ajuste em
`main.go`, 1 regra `depguard` nova no `.golangci.yml`, ~5 chamadas ajustadas nos testes
de integração.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

O serviço não tem constituição própria (`pagamento/.specify/memory/` não existe); vale a
do workspace (`../.specify/memory/constitution.md`, v1.0.0).

| Princípio | Veredito | Evidência |
|---|---|---|
| I. Complexidade só se necessária ou pedida | **PASS** | A troca foi pedida. Sem camada de repositório nova, sem generics, sem `AutoMigrate`, sem hooks, sem associações. A regra `depguard` é o mínimo para cumprir SC-005: sem código novo, só configuração do linter que o serviço já usa |
| II. Domínio e API têm teste automatizado | **PASS** | Nenhum comportamento muda; as suítes unitárias e de integração existentes são o critério de aceite (`idempotencia_test`, `vazao_test`, `esquema_test`, `cobranca_test` e demais em `test/integration`) |
| III. O código é a fonte da verdade | **PASS** | Comportamentos a preservar lidos de `transacoes.go`, `ports.go`, das migrações e dos testes, não da spec 001. Uma imprecisão da própria spec 002 (varredura vazia devolve `nil`, não coleção vazia) foi corrigida para refletir o código |
| IV. Divergência código↔spec é pergunta | **ATENÇÃO** | Ver "Perguntas ao mantenedor" |

**Re-avaliação pós-Fase 1**: o desenho não introduziu violação. Nenhuma entrada em
Complexity Tracking.

### Perguntas ao mantenedor (princípio IV)

1. **Como verificar a fronteira do núcleo (FR-010/SC-005)**: o plano adota a regra
   `depguard` `nucleo-sem-adaptadores` no `.golangci.yml` (mesmo mecanismo do
   `catalogo` e a escolha já feita para o `notificacao`), em vez do
   `test/arquitetura_test.go` do `estoque`. Consequência: a verificação roda em
   `make lint`, não em `make test`. Se preferir o teste em Go, é só dizer; só as
   tarefas T015 e T016 dependem da resposta.
2. **Spec 001**: `plan.md`/`research.md`/`data-model.md` da 001 citam `pgx` com SQL à
   mão. Após esta feature deixam de descrever o código. O plano assume que a 001 fica
   como registro histórico e **não a edita**.

## Project Structure

### Documentation (this feature)

```text
specs/002-persistencia-gorm/
├── plan.md              # Este arquivo
├── spec.md
├── research.md          # Fase 0 — decisões D1..D8
├── data-model.md        # Fase 1 — tabela → modelo GORM e consultas
├── quickstart.md        # Fase 1 — como validar a equivalência
├── checklists/requirements.md
└── tasks.md             # Fase 2 — /speckit-tasks
```

Sem `contracts/`: nenhum contrato externo muda.

### Source Code (pagamento/)

```text
pagamento/
├── cmd/pagamento/main.go                # postgres.Conectar → *Banco; defer Fechar; NovoRepositorio(banco.DB()); prontidão usa banco.Verificar
├── internal/adapter/postgres/
│   ├── pool.go → postgres.go            # Banco{db *gorm.DB, sql *sql.DB}: Conectar, Fechar, Verificar, DB(), SQL() (Schema = "pagamento" permanece)
│   ├── modelos.go                       # NOVO — transacaoRow (+ TableName) e conversões de/para o domínio
│   ├── transacoes.go                    # Repositorio{db *gorm.DB} sobre GORM; mesma porta usecase.Repositorio
│   └── varredura.go                     # NÃO muda (RotinaPeriodica não toca em banco)
├── .golangci.yml                        # + linter depguard, regra nucleo-sem-adaptadores
├── test/integration/*_test.go           # harness: Pool *pgxpool.Pool → Banco; Pool.QueryRow → Banco.SQL().QueryRowContext (4 pontos)
├── go.mod / go.sum                      # + gorm, driver postgres
└── internal/domain, internal/usecase    # NÃO mudam
```

**Structure Decision**: mesma estrutura hexagonal; a mudança fica toda no adaptador
`postgres`. `modelos.go` é novo para que a struct com tags GORM não vaze para fora do
pacote e a conversão fique em um lugar só. `pool.go` é renomeado para `postgres.go`
(passa a abrir um `Banco`, não um pool), como no `notificacao`.

## Complexity Tracking

Sem violações a justificar.
