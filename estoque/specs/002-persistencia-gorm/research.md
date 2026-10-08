# Research: Persistência do Estoque via GORM

Cada decisão foi checada contra o código atual (`internal/adapter/postgres/*.go`),
não contra a spec 001 (princípio IX).

## D1 — Como abrir a conexão

**Decisão**: `pgx.ParseConfig(url)` → `RuntimeParams["search_path"] = "estoque"` →
`stdlib.OpenDB(*cfg)` → `*sql.DB` (com `SetConnMaxIdleTime(5*time.Minute)`) →
`gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{...})` → `PingContext`.

**Rationale**: reproduz o que `Abrir` faz hoje (aceita URL e DSN chave=valor, fixa o
`search_path`, falha cedo com `Ping`) e preserva o tipo de erro `*pgconn.PgError`, de
que `ehConflitoDeTravamento` (`55P03`) e `ehViolacaoDeUnicidade` (`23505`) dependem.

**Alternativas**: (a) `postgres.Open(dsn)` com `search_path` concatenado na URL —
rejeitada: quebra para DSN chave=valor e duplica a lógica de parsing; (b) `gorm.Open`
com `Config.DSN` e `NamingStrategy{TablePrefix: "estoque."}` — rejeitada: qualifica
tabelas no código, contra o desenho "migração qualifica, serviço usa `search_path`".

**Config do GORM**: `SkipDefaultTransaction: true`; `Logger: logger.Discard`;
`DisableAutomaticPing: true` (o `Ping` é explícito); sem `PrepareStmt`.

## D2 — Travamento de linhas

**Decisão**: `tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"})` no
`SELECT` das poltronas, com `Where("sessao_id = ? AND rotulo IN ?", ...)` e
`Order("rotulo")`. `Options: "SKIP LOCKED"` nas varreduras/outbox.

**Rationale**: o GORM suporta as duas formas nativamente; a ordem determinística é
mantida (princípio da exclusão no Postgres). Conflito `55P03` continua virando
`ErrPoltronasIndisponiveis`.

**Risco a verificar cedo (spike T-inicial)**: o `UPDATE ... WHERE id IN (subconsulta
... FOR UPDATE SKIP LOCKED) RETURNING id`. O GORM aceita `*gorm.DB` como subconsulta e
`clause.Returning`, mas é preciso confirmar que o `FOR` da subconsulta é emitido. Se
não for, usa-se `Raw(...).Scan(&ids)` com o mesmo SQL de hoje (já previsto na spec).

**Veredito do spike (T003, 2026-10-07, Postgres 16 real)**: a subconsulta com
`clause.Locking{Options:"SKIP LOCKED"}` + `clause.Returning` emite exatamente
`UPDATE "reservas" SET … WHERE id IN (SELECT "id" … ORDER BY id LIMIT $3 FOR UPDATE SKIP LOCKED) RETURNING "id"`
e devolve os ids. **Sem fallback `Raw`** para as varreduras.
Também confirmado: `Updates(map{"atualizado_em": gorm.Expr("now()")})` altera a
coluna mesmo com `<-:update` no modelo.

**Alternativas**: lock otimista/versionamento (`gorm:"version"`) — rejeitada, muda a
semântica de exclusão e a falha deixa de ser imediata.

## D3 — Modelos

**Decisão**: structs privadas por tabela no adaptador, com `TableName()` sem schema e
`gorm:"column:...;primaryKey"`; conversão explícita modelo ↔ domínio nas bordas.
Uso sempre com `Select`/mapas para `Updates` (evitar o descarte silencioso de zero
values do GORM). Sem `Preload`, sem associações declaradas, sem `gorm.Model`,
sem `CreatedAt/UpdatedAt` automáticos (`atualizado_em` continua `now()` explícito).

**Rationale**: o domínio não pode ganhar tags nem import (arquitetura); associações
mágicas escondem o que roda no banco, e a exclusividade depende de saber exatamente
quais linhas são travadas e em que ordem.

**Alternativas**: tags `gorm` nas entidades de domínio — rejeitada, viola o
princípio I; pacote `models` separado — rejeitado, indireção sem necessidade (VII).

## D4 — Esquema

**Decisão**: nenhum `AutoMigrate`. Migrações SQL de `migrations/` seguem como única
fonte; o serviço não emite DDL.

**Rationale**: FR-006/FR-007; `schema_migrations` já fixada no schema `estoque` por
`search_path` no `migrate`.

## D5 — Consultas sem equivalente nativo

**Decisão**: `pg_try_advisory_xact_lock($1)` via `tx.Raw(...).Scan(&obtido)`;
`now()` em atualizações via `gorm.Expr("now()")`; `ANY($1)` vira `IN ?` com slice;
`ON CONFLICT DO NOTHING` via `clause.OnConflict{DoNothing: true}`; `RETURNING id` via
`clause.Returning`.

**Rationale**: mantém o SQL efetivo igual ao de hoje, linha a linha.

## D6 — Colunas JSONB, DECIMAL e anuláveis

**Decisão**: `payload` e `trace_context` como `[]byte` com `gorm:"type:jsonb"`;
`trace_context` ausente vira `NULL` (usar `nil` explícito, não `[]byte{}`);
`valor_total` como `*string` (coluna anulável para reservas antigas, e o domínio já a
trata como `string`); `finalizado_em` como `*time.Time`.

**Veredito do spike (T003)**: `[]byte(nil)` em `jsonb` grava `NULL`; com conteúdo,
grava e lê de volta o JSON. Modelo `[]byte` aprovado.

**Risco (resolvido)**: `[]byte(nil)` passando por `database/sql` → pgx stdlib para uma coluna
`jsonb`. Coberto por `rastreamento_test.go` e `broker_test.go` (com e sem
`trace_context`); se falhar, usar `*string`/`sql.NullString` no modelo.

## D7 — Mapeamento de erros

**Decisão**: manter `indisponivel(err)`, `ehConflitoDeTravamento` e
`ehViolacaoDeUnicidade` com `errors.As(*pgconn.PgError)`. `gorm.ErrRecordNotFound`
nunca é produzido (usa-se `Find`, não `First`), então "0 linhas" continua sendo
tratado pelos casos de uso como hoje.

## D8 — Testes de integração

**Decisão**: `Cenario.Pool` passa de `*pgxpool.Pool` para `*sql.DB` obtido de
`banco.SQL()`; as ~12 chamadas viram `QueryRowContext/QueryContext`. Asserções
intocadas. `TestSchemaProprio` continua verificando o `search_path` **da conexão do
próprio `Banco`** — por isso o harness não ganha um pool independente. Em
`main_test.go`, `aplicarMigracoes` fica com `pgxpool` (é só preparação do banco).

**Teste de arquitetura**: acrescentar `gorm.io` aos imports proibidos ao núcleo.

**Nenhum NEEDS CLARIFICATION restante.**
