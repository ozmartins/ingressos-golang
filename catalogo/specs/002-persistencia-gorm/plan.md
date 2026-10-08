# Implementation Plan: Persistência do Catálogo via GORM

**Branch**: `master` (nenhuma branch de feature — regra do mantenedor) | **Date**: 2026-10-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/002-persistencia-gorm/spec.md`

## Summary

Trocar o acesso a dados do `Servico-Catalogo` de `pgx` (`pgxpool`) com SQL à mão para o
GORM, sem alterar nada que um consumidor observe. A troca é inteira dentro de
`internal/adapter/postgres/` (820 linhas, 9 arquivos), mais a fiação em
`cmd/catalogo/main.go` e o harness dos testes de integração.

Decisões que moldam o desenho (detalhe em [research.md](./research.md)):

1. **O GORM entra por cima do mesmo driver.** `gorm.io/driver/postgres` usa o pgx; a
   conexão é montada de `pgx.ParseConfig` com `search_path=catalogo` e entregue ao GORM
   como `*sql.DB` (`stdlib.OpenDB`). `DATABASE_URL`, schema, pool (10 conexões, 1 h) e
   `Ping` de 5 s na largada ficam como hoje (D1).
2. **A caixa de saída mantém `FOR UPDATE SKIP LOCKED` e `ON CONFLICT DO NOTHING`**, agora
   por `clause.Locking` e `clause.OnConflict`; a publicação segue acontecendo dentro da
   transação que segura os locks (D2).
3. **Modelos GORM vivem no adaptador, não no domínio** — structs privadas `*Row` com tags
   e `TableName()` sem schema; sem associações, sem `AutoMigrate`, sem timestamps
   automáticos (D3, D4).
4. **Paginação deixa de montar SQL por string**: `consultarPaginado` recebe uma consulta
   `*gorm.DB` já filtrada e faz `Count` + `Order/Limit/Offset` (D5).
5. **O que o GORM não expressar** (hoje: a sobreposição de horário em `SalaOcupada`, com
   `INTERVAL`) usa `Raw`/`Scan` do próprio GORM dentro do adaptador — previsto na spec.
6. **O plano de consultas passa a ser testado sobre o SQL emitido pelo GORM**, não sobre
   SQL escrito no teste (D8) — sem isso, a spec (edge case de índices) não seria verificável.

Nenhum contrato muda (REST, gRPC, AMQP), então esta feature **não tem `contracts/`**.

## Technical Context

**Language/Version**: Go 1.25 (`go.mod`)

**Primary Dependencies**: acrescenta `gorm.io/gorm` v1.31.2 e `gorm.io/driver/postgres`
v1.6.2 (ambos já no cache de módulos local). Mantém `github.com/jackc/pgx/v5` v5.10.0
(agora como driver do GORM e para montar a conexão); `pgxpool` sai do código de produção.

**Storage**: PostgreSQL 16, schema `catalogo`, as mesmas 5 tabelas (`filmes`, `cinemas`,
`salas`, `sessoes`, `outbox_eventos`) e as 6 migrações existentes. RabbitMQ intocado.

**Testing**: `go test -race ./...` (domínio e casos de uso, sem banco) e
`make test-integration` (Testcontainers; mesmas suítes, asserções inalteradas), mais
`test/contract` (contrato HTTP).

**Target Platform**: contêiner Linux distroless (sem mudança no Dockerfile; o
`go mod download` resolve os módulos novos).

**Project Type**: microsserviço (REST + gRPC cliente + publicador AMQP).

**Performance Goals**: iguais aos da spec 001; as consultas principais continuam usando
seus índices (verificado por `TestConsultasUsamOsIndices`, ver D8).

**Constraints**: fato de saída na mesma transação do efeito; `SKIP LOCKED` na drenagem;
nenhum DDL emitido pelo serviço; logger do GORM silenciado (nunca registra SQL ou
parâmetros — princípio IV).

**Scale/Scope**: 9 arquivos de adaptador reescritos (`pool.go` → `postgres.go`,
`transacao.go` absorvido, `tipos.go` simplificado, + `modelos.go`), 1 ajuste em
`main.go`, 1 regra em `.golangci.yml`, harness de integração (~25 chamadas
`pool.Exec/QueryRow`).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Avaliado contra `.specify/memory/constitution.md` **v1.2.0**.

| Princípio | Veredito | Evidência no desenho |
|---|---|---|
| I. Dependências Apontam Para Dentro | **PASS** | GORM só em `internal/adapter/postgres`. `gorm.io` entra na regra `nucleo-sem-adaptadores` do depguard (`.golangci.yml`), verificação mecânica na esteira (`make lint`) |
| II. Configuração Externa, Falha na Largada | **PASS** | `DATABASE_URL` segue como única configuração; URL malformada derruba a largada (`pgx.ParseConfig`) e o `Ping` na abertura continua obrigatório |
| III. Fronteira de Estado Explícita | **PASS** | O catálogo continua dono do seu schema; nenhum estado alheio é persistido |
| IV. Erro é Contrato | **PASS** | Nenhuma categoria de erro muda; erros do GORM/driver são embrulhados no adaptador como hoje e a tradução para a resposta não os expõe. Logger do GORM descartado (sem SQL/parâmetros nos registros) |
| V. Integração Síncrona Tem Orçamento | **PASS** | Todo acesso usa `WithContext(ctx)`, preservando prazos e cancelamento da requisição; pool com os mesmos limites. O banco não é "serviço externo" de negócio, e nada de retentativa é adicionado |
| VI. Entrega de Fato é Ao Menos Uma Vez | **PASS** | Fato gravado na mesma `Transaction` do efeito; drenagem com `SKIP LOCKED` e marcação na mesma transação; idempotência de gravação por `message_id` com `ON CONFLICT DO NOTHING` |
| VII. Complexidade Só Entra Se For Necessária ou Pedida | **PASS** | A troca foi pedida. Sem camada nova, sem generics novos além do `consultarPaginado` que já existe, sem `AutoMigrate`, hooks ou associações. Alternativas rejeitadas em research D1/D3 |
| VIII. Domínio e API Têm Teste Automatizado | **PASS** | Nenhum comportamento de domínio/API muda; as suítes unitárias, de contrato e de integração existentes são o critério de aceite. D8 corrige uma lacuna: o teste de planos não exercitava o SQL do adaptador |
| IX. O Código é a Fonte da Verdade | **PASS** | Comportamentos a preservar lidos do código (`sessao_repository.go`, `outbox.go`, migrações), não da spec 001 |
| X. Divergência Entre Código e Spec é Pergunta | **ATENÇÃO** | Duas divergências, ver abaixo. Nenhuma foi resolvida unilateralmente |

**Re-avaliação pós-Fase 1**: o desenho não introduziu violação. Nenhuma entrada em
Complexity Tracking.

### Divergências resolvidas pelo mantenedor (princípio X)

Respondidas em 2026-10-07:

1. **Spec 001 × código após a troca** — decisão: **atualizar a 001** para refletir o GORM
   (tarefa T025). Trechos afetados: `research.md` (decisão de persistência e consequência
   sobre `preco_base`), `plan.md` (dependência `pgx/v5`) e `tasks.md` (T002).
2. **`CLAUDE.md` × código** — decisão: **corrigir os `CLAUDE.md`**. Feito: ambos agora dizem
   que `test/arquitetura_test.go` existe só no `estoque`, que o `catalogo` usa o depguard e
   que `pagamento`/`notificacao` não têm nenhum dos dois.
3. **FR-013 × código** — decisão: **ajustar a spec**. Feito: FR-013 agora diz "preservar o
   tratamento de erros de hoje", sem tradução nova de SQLSTATE.

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

### Source Code (catalogo/)

```text
catalogo/
├── cmd/catalogo/main.go                # NovoPool(...) → Abrir(...); repositórios recebem *Banco; Saude usa banco.Ping
├── internal/adapter/postgres/
│   ├── postgres.go                     # NOVO (substitui pool.go + transacao.go): Banco{db *gorm.DB}, Abrir, Fechar, Ping, SQL() *sql.DB, EmTransacao
│   ├── modelos.go                      # NOVO: structs *Row com tags gorm e TableName()
│   ├── paginacao.go                    # consultarPaginado recebe *gorm.DB
│   ├── tipos.go                        # dinheiroDeTexto (numeric como string → Dinheiro)
│   ├── filme_repository.go
│   ├── cinema_repository.go
│   ├── sala_repository.go
│   ├── sessao_repository.go
│   └── outbox.go                       # enfileirarFato + CaixaDeSaida.Drenar
├── .golangci.yml                       # + "gorm.io" na regra nucleo-sem-adaptadores
├── test/integration/                   # harness: pool vira *Banco + *sql.DB; planos_consulta_test sobre o SQL do GORM
├── go.mod / go.sum                     # + gorm, + driver/postgres (+ jinzhu/inflection, jinzhu/now)
└── internal/domain, internal/usecase   # INTOCADOS
```

**Structure Decision**: sem pacote novo. O GORM fica em `internal/adapter/postgres` (o
nome do pacote não muda: o banco continua PostgreSQL e renomear quebraria imports sem
ganho — princípio VII). `pool.go` e `transacao.go` se fundem em `postgres.go` porque
deixam de existir como conceitos separados (pool + helper de `pgx.Tx`); os modelos ganham
arquivo próprio só para não misturar tags com lógica de consulta.

## Complexity Tracking

Nenhuma violação a justificar.
