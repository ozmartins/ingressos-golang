# Implementation Plan: Persistência do Estoque via GORM

**Branch**: `master` (nenhuma branch de feature — regra do mantenedor) | **Date**: 2026-10-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/002-persistencia-gorm/spec.md`

## Summary

Trocar o acesso a dados do `Servico-Estoque` de `pgx` com SQL à mão para o GORM,
sem alterar nada que um consumidor observe. A troca é inteira dentro de
`internal/adapter/postgres/` (682 linhas, 7 arquivos), mais a fiação em
`cmd/estoque/main.go` e o harness dos testes de integração.

Decisões que moldam o desenho (detalhe em [research.md](./research.md)):

1. **O GORM entra por cima do mesmo driver.** `gorm.io/driver/postgres` usa o pgx
   por baixo; a conexão continua sendo montada a partir de `pgx.ParseConfig` com
   `search_path=estoque` em `RuntimeParams`, e entregue ao GORM como `*sql.DB`.
   Assim a URL (`DATABASE_URL`), o schema e o `PgError` (códigos `55P03`, `23505`)
   se comportam exatamente como hoje (D1).
2. **A exclusão continua no PostgreSQL**, expressa com `clause.Locking`
   (`FOR UPDATE NOWAIT`, `FOR UPDATE SKIP LOCKED`) e `ORDER BY rotulo`. Nada de
   lock otimista, nada de `Save`/hooks (D2).
3. **Modelos GORM vivem no adaptador, não no domínio.** Structs privadas
   (`poltronaRow`, `reservaRow`, …) com tags `gorm:"column:..."` e `TableName()`
   sem qualificar o schema; o domínio segue sem tags nem import (D3).
4. **Esquema só pelas migrações.** Sem `AutoMigrate`; `schema_migrations` continua
   fixada em `estoque` (D4).
5. **Onde o GORM não expressar algo** (hoje: `pg_try_advisory_xact_lock`), usa-se
   `Raw`/`Exec` do próprio GORM dentro do adaptador — previsto na spec (D5).

Nenhum contrato muda (gRPC, REST, AMQP), então esta feature **não tem
`contracts/`**.

## Technical Context

**Language/Version**: Go 1.25 (`go.mod`; toolchain local 1.27)

**Primary Dependencies**: acrescenta `gorm.io/gorm` v1.31.2 e
`gorm.io/driver/postgres` v1.6.2 (ambos já no cache de módulos local). Mantém
`github.com/jackc/pgx/v5` (agora usado para montar a conexão e como driver do GORM).
Remove nada: `pgxpool` sai do código de produção, mas o pgx continua direto.

**Storage**: PostgreSQL 16, schema `estoque`, as mesmas 5 tabelas e as 3 migrações
existentes. Redis (índice de prazo) e RabbitMQ intocados.

**Testing**: `go test -race ./...` (domínio e casos de uso, sem banco) e
`make test-integration` (Testcontainers com Postgres/Redis/RabbitMQ reais — as
mesmas suítes, asserções inalteradas).

**Target Platform**: contêiner Linux (sem mudança no Dockerfile além do
`go mod download` resolver os dois módulos novos).

**Project Type**: microsserviço (gRPC + consumidores AMQP + rotinas de fundo).

**Performance Goals**: iguais aos da spec 001 (p99 do bloqueio < 100 ms; mapa de
500 lugares < 200 ms p99). `SkipDefaultTransaction: true` evita ida-e-volta extra
por escrita; sem `PrepareStmt` (o pgx já cacheia).

**Constraints**: `FOR UPDATE NOWAIT` obrigatório no bloqueio (falha imediata em vez
de espera); fato de saída na mesma transação da reserva; nenhum SQL de DDL emitido
pelo serviço; logger do GORM silenciado (não vaza SQL/parâmetros — princípio IV).

**Scale/Scope**: 7 arquivos de adaptador reescritos, ~1 ajuste em `main.go`, 1
regra nova no teste de arquitetura, todas as chamadas `Pool.QueryRow`/`Pool.Query` nos testes de integração, mais um teste novo de rollback.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Avaliado contra `.specify/memory/constitution.md` **v1.1.0**.

| Princípio | Veredito | Evidência no desenho |
|---|---|---|
| I. Dependências Apontam Para Dentro | **PASS** | GORM só em `internal/adapter/postgres`. `test/arquitetura_test.go` ganha `gorm.io` na lista de proibidos para o núcleo (FR-010), com a verificação mecânica que o princípio exige |
| II. Configuração Externa, Falha na Largada | **PASS** | `DATABASE_URL` continua sendo a única configuração; URL malformada continua derrubando a largada (`pgx.ParseConfig`), e o `Ping` na abertura segue obrigatório |
| III. Fronteira de Estado Explícita | **PASS** | O estoque continua dono do seu schema; Redis continua só como índice de prazo |
| IV. Erro é Contrato | **PASS** | Erros do GORM/driver são traduzidos no adaptador para `shared.Err*`; nada de texto de driver sobe. Logger do GORM descartado para não registrar SQL/parâmetros |
| V. Integração Síncrona Tem Orçamento | **PASS** | `NOWAIT` mantido; contexto da requisição propagado com `WithContext(ctx)` em toda chamada; pool com `ConnMaxIdleTime` de 5 min como hoje |
| VI. Entrega de Fato é Ao Menos Uma Vez | **PASS** | Outbox gravado na mesma `Transaction` da reserva; publicador lê com `SKIP LOCKED` na mesma transação que marca `publicado_em`; idempotência por `mensagens_processadas` com `ON CONFLICT DO NOTHING` + `RowsAffected` |
| VII. Complexidade Só Entra Se For Necessária ou Pedida | **PASS** | A troca foi pedida. Sem camada de repositório nova, sem generics, sem `AutoMigrate`, sem hooks, sem associações do GORM (`Preload`/`has many`): as relações continuam explícitas como hoje. Alternativas rejeitadas em research D2/D3 |
| VIII. Domínio e API Têm Teste Automatizado | **PASS** | Nenhum comportamento de domínio/API muda; as suítes de contrato e de integração existentes são o critério de aceite. Invariante de concorrência e idempotência seguem testadas com Postgres real (`invariante_test.go`, `bloqueio_test.go`, `desfechos_test.go`) |
| IX. O Código é a Fonte da Verdade | **PASS** | Os comportamentos a preservar foram lidos do código (`bloqueio.go`, `expiracao.go`, `outbox.go`, `poltronas.go`, `mensagens.go`), não da spec 001 |
| X. Divergência Entre Código e Spec é Pergunta | **PASS** (decisão do mantenedor registrada) | Os artefatos da 001 dizem "pgx, SQL à mão" e passarão a divergir do código **por decisão do mantenedor**, que os trata como histórico; ver "Decisão do mantenedor" abaixo |

**Re-avaliação pós-Fase 1**: o desenho não introduziu violação. Nenhuma entrada em
Complexity Tracking.

### Decisão do mantenedor (princípio X)

Os documentos da spec 001 citam `pgx` e "SQL à mão com `FOR UPDATE` explícito" como
decisão de persistência. Em 2026-10-07 o mantenedor decidiu **tratar a 001 como
registro histórico**: ela não é editada, e esta 002 supera a decisão de persistência.

## Project Structure

### Documentation (this feature)

```text
specs/002-persistencia-gorm/
├── plan.md              # Este arquivo
├── spec.md
├── research.md          # Fase 0 — decisões D1..D8
├── data-model.md        # Fase 1 — mapeamento tabela → modelo GORM e consultas
├── quickstart.md        # Fase 1 — como validar a equivalência
├── checklists/requirements.md
└── tasks.md             # Fase 2 — /speckit-tasks
```

Sem `contracts/`: nenhum contrato externo muda.

### Source Code (estoque/)

```text
estoque/
├── cmd/estoque/main.go                 # troca postgres.Abrir(...) — mesma assinatura; sem mudança esperada além de imports
├── internal/adapter/postgres/
│   ├── postgres.go                     # Banco{db *gorm.DB}: Abrir, Fechar, Verificar, EmTransacao, SQL() *sql.DB, tradutores de erro
│   ├── modelos.go                      # NOVO: structs *Row com tags gorm e TableName()
│   ├── bloqueio.go                     # Conceder + aplicarDesfecho (Confirmar/Cancelar)
│   ├── expiracao.go                    # ExpirarVencidas, ExpirarUma, CancelarPendentesDaSessao
│   ├── mensagens.go                    # registrarProcessada, LimparMensagensProcessadas
│   ├── outbox.go                       # enfileirarFato, PendentesParaPublicar
│   ├── poltronas.go                    # MapaDaSessao, ProvisionarMatriz
│   └── varredura.go                    # RotinaPeriodica — sem mudança
├── test/arquitetura_test.go            # + "gorm.io" nos imports proibidos ao núcleo
├── test/integration/                   # harness: Pool vira *sql.DB (banco.SQL()); QueryRow→QueryRowContext
├── go.mod / go.sum                     # + gorm, + driver/postgres (+ jinzhu/inflection, jinzhu/now)
└── internal/domain, internal/usecase   # INTOCADOS
```

**Structure Decision**: sem pacote novo. O GORM fica em `internal/adapter/postgres`
(o nome do pacote não muda, porque o banco continua sendo PostgreSQL e renomear
quebraria imports sem ganho — princípio VII). Os modelos ganham arquivo próprio
só para não misturar tags com lógica de consulta.

## Complexity Tracking

Nenhuma violação a justificar.
