# Research: Persistência do Catálogo via GORM

Cada decisão foi checada contra o código atual (`internal/adapter/postgres/*.go`,
`migrations/`, `test/integration/`), não contra a spec 001 (princípio IX).

## D1 — Como abrir a conexão

**Decisão**: `pgx.ParseConfig(url)` → `RuntimeParams["search_path"] = "catalogo"` →
`stdlib.OpenDB(*cfg)` → `*sql.DB` (`SetMaxOpenConns(10)`, `SetConnMaxLifetime(time.Hour)`,
os mesmos números de `pool.go` hoje) → `gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{...})`
→ `PingContext` com timeout de 5 s.

**Rationale**: reproduz o que `NovoPool` faz (URL ou DSN, `search_path` fixo, falha cedo
no `Ping`) e preserva `*pgconn.PgError` para quem precisar inspecionar códigos SQLSTATE.

**Alternativas**: `postgres.Open(dsn)` com `search_path` concatenado na URL — rejeitada:
quebra DSN chave=valor; `NamingStrategy{TablePrefix: "catalogo."}` — rejeitada: qualifica
tabelas no código, contra o desenho "migração qualifica, serviço usa `search_path`".

**Config do GORM**: `SkipDefaultTransaction: true`; `Logger: logger.Discard` (nunca
registra SQL/parâmetros — princípio IV); `DisableAutomaticPing: true`; sem `PrepareStmt`.

## D2 — Caixa de saída (SKIP LOCKED, ON CONFLICT)

**Decisão**:
- Enfileirar: `tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "message_id"}}, DoNothing: true}).Create(&outboxRow{...})`.
- Drenar: dentro de `db.WithContext(ctx).Transaction(...)`,
  `tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("publicado_em IS NULL").Order("id").Limit(n).Find(&lote)`;
  depois `UPDATE ... SET tentativas = tentativas + 1` (`gorm.Expr`) ou `publicado_em = now()` por `id`.

**Rationale**: o GORM expressa as duas coisas nativamente, e o desenho é o de hoje: a
publicação acontece **dentro** da transação que segura os locks, para que duas réplicas
não tomem o mesmo fato. `publicar` continua sendo chamado com a transação aberta.

**Risco a verificar cedo (spike)**: `Create` com `ON CONFLICT DO NOTHING` não preenche `id`
quando o conflito ocorre; o código atual não usa o `id` do insert, então nada depende
disso. Confirmar também que `RETURNING id` não faz o GORM tratar "0 linhas" como erro.

## D3 — Modelos

**Decisão**: structs privadas por tabela no adaptador (`filmeRow`, `cinemaRow`, `salaRow`,
`sessaoRow`, `outboxRow`), com `TableName()` sem schema e `gorm:"column:..."`; conversão
explícita modelo ↔ domínio nas bordas. Escritas sempre por `Select`/`map` ou `Updates`
com mapa (evitar o descarte silencioso de zero values — `ativo = false` precisa gravar).
Sem `Preload`, sem associações, sem `gorm.Model`, sem `CreatedAt/UpdatedAt` automáticos
(`atualizado_em` continua `CURRENT_TIMESTAMP` explícito, via `gorm.Expr`).

**Rationale**: o domínio (`catalogo.Filme`, `Sala`, …) não pode ganhar tags nem import. As
junções da grade de sessões são lidas por projeção (`Select` + `Scan`) em uma struct de
leitura, exatamente as colunas de hoje.

**Alternativas**: tags `gorm` nas entidades de domínio — rejeitada (princípio I); pacote
`models` separado — rejeitado (indireção sem necessidade, princípio VII).

## D4 — Esquema

**Decisão**: nenhum `AutoMigrate`. As 6 migrações em `migrations/` seguem como única fonte;
o serviço não emite DDL. `schema_migrations` continua em `catalogo` pelo `search_path` do
`migrate` (Makefile `MIGRATE_URL`).

## D5 — Listagens paginadas e filtros

**Decisão**: `consultarPaginado[T]` deixa de receber SQL e passa a receber uma consulta
`*gorm.DB` já filtrada: `Count(&total)` primeiro, depois `Order(...).Limit(...).Offset(...)`.
O filtro "nulo = qualquer" (`($1::boolean IS NULL OR ativo = $1)`) vira `Where` condicional
(`if filtro.Ativo != nil`). Listas de status usam `Where("status IN ?", lista)`.

**Rationale**: o resultado é o mesmo conjunto com a mesma ordenação; o SQL emitido fica mais
simples e **melhor** para o planejador (sem o `OR ... IS NULL`).

**Risco**: `status IN ($1,$2)` no lugar de `status = ANY($1)` pode mudar o plano. O
`TestConsultasUsamOsIndices` atual executa **SQL escrito no próprio teste**, não o que o
adaptador emite — portanto não protegeria contra regressão introduzida pelo GORM. Ver D8.

## D6 — Tipos: JSONB, NUMERIC, TIMESTAMPTZ

**Decisão**:
- `salas.layout` (JSONB): campo `[]byte` no modelo; (de)serialização continua em
  `layoutParaJSON` / `lerSala` com `encoding/json`, como hoje.
- `outbox_eventos.payload` e `trace_context` (JSONB): `[]byte`; `trace_context` ausente é
  `nil` (grava `NULL`), nunca `[]byte{}`.
- `sessoes.preco_base` (`DECIMAL(10,2)`): lido como `string` (o driver stdlib do pgx devolve
  `numeric` como texto) e convertido por `big.Rat.SetString` → `catalogo.DinheiroDeRat`;
  gravado como `PrecoBase.String()`, como hoje. `pgtype.Numeric` e `tipos.go` deixam de
  ser necessários — o tratamento de nulo/NaN/infinito é preservado na nova função.
- Datas: `time.Time`, com `.UTC()` na leitura como hoje.

**Risco**: `[]byte` → `jsonb` e `numeric` → `string` via `database/sql`. Coberto por
`caixa_de_saida_test.go`, `escrita_salas_sessoes_test.go` e `sessoes_test.go`; plano B:
`*string`/`sql.NullString` no modelo.

## D7 — Mapeamento de erros e "não encontrado"

**Decisão**: `Find(...).Limit(1)` + `RowsAffected == 0` (ou `Take` tratando
`gorm.ErrRecordNotFound`) → `shared.NaoEncontrado(...)`, nos mesmos pontos de hoje
(`BuscarPorID`, `Atualizar`, `Desativar`, `Cancelar`). Demais erros continuam embrulhados
com `%w` como hoje; nenhum texto de driver chega à resposta (a tradução para categorias
HTTP já é feita nos casos de uso/adaptador HTTP e não muda).

## D8 — Testes de integração e plano de consultas

**Decisão**:
- `main_test.go`: a variável `pool` passa a ser o `*postgres.Banco`; um `*sql.DB` obtido
  por `banco.SQL()` serve às chamadas diretas dos testes (`Exec`/`QueryRow` →
  `ExecContext`/`QueryRowContext`). `aplicarMigracoes` e o teste de `search_path` continuam
  verificando a conexão do próprio `Banco`. Asserções intocadas.
- `TestConsultasUsamOsIndices`: passa a executar o **SQL realmente emitido pelo GORM**
  — capturado com `db.Session(&gorm.Session{DryRun: true})` sobre as mesmas consultas do
  adaptador — em vez de SQL próprio. É a única maneira de cumprir o Edge Case "a troca
  não pode degradar consultas que hoje usam índice". Se uma consulta perder o índice, o
  ajuste é feito no adaptador (ex.: `= ANY(?)` com `Raw`), não no teste.
- Verificação de dependências: acrescentar `gorm.io` à regra `nucleo-sem-adaptadores` do
  `.golangci.yml` (hoje só `github.com/jackc/pgx` está proibido).

**Nenhum NEEDS CLARIFICATION restante.**
