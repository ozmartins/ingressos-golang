# Research: Persistência do Pagamento via GORM

Cada decisão foi checada contra o código atual (`internal/adapter/postgres/*.go`,
`internal/usecase/ports.go`, `migrations/*.up.sql`, `test/integration/*`), não contra a
spec 001. O desenho segue o já adotado no `notificacao` (`notificacao/specs/002-persistencia-gorm`),
com as diferenças que o modelo do pagamento impõe (D3, D4, D6).

## D1 — Como abrir a conexão

**Decisão**: `pgx.ParseConfig(url)` → `RuntimeParams["search_path"] = "pagamento"` →
`stdlib.OpenDB(*cfg)` → `*sql.DB` → `gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}),
&gorm.Config{SkipDefaultTransaction: true, Logger: logger.Discard, DisableAutomaticPing:
true})` → `PingContext` explícito. O `*sql.DB` fica no `Banco` para `Fechar`,
`Verificar` (`PingContext`, usado na prontidão) e para os testes de integração (`SQL()`).

**Rationale**: reproduz o que `Abrir` faz hoje — aceita URL e DSN, fixa o `search_path`
(FR-007), falha cedo com ping, preserva a mensagem `DATABASE_URL malformada`. O teste
`esquema_test` (lê `current_setting('search_path')`) segue valendo sem mudança.

**Pool**: o `pgxpool` tinha tamanho padrão `max(4, NumCPU)`; `database/sql` não limita por
padrão. Aplicar `SetMaxOpenConns(max(4, runtime.NumCPU()))` — o mesmo teto de antes,
para não esgotar conexões do Postgres sob a rajada de `vazao_test`.

**Alternativas**: (a) `postgres.Open(dsn)` com `search_path` concatenado — rejeitada:
quebra para DSN chave=valor; (b) `NamingStrategy{TablePrefix: "pagamento."}` — rejeitada:
qualifica tabela no código, contra o desenho "migração qualifica, serviço usa
`search_path`".

## D2 — Registro idempotente por reserva

**Decisão**: `db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name:
"reserva_id"}}, DoNothing: true}, clause.Returning{}).Create(&row)`.
`RowsAffected == 1` → criada, e a linha devolvida vem do `RETURNING` (mesmo
comportamento de hoje: o valor volta já normalizado pelo banco, ex.: `10.5` → `10.50`);
`0` → `BuscarPorReserva` e devolver `(false, atual)`.

**Rationale**: o SQL é o mesmo de hoje; a unicidade segue decidida pelo índice único. Sem
`clause.Returning{}`, o GORM só pediria de volta colunas com default e devolveria o valor
cru, não o normalizado — por isso o `Returning` é explícito.

**Alternativas**: `FirstOrCreate` (SELECT+INSERT, corrida entre os dois) e `Save` (upsert,
quebraria a idempotência) — rejeitados. Re-ler a linha após o `Create` — rejeitado: ida e
volta extra sem ganho.

**Risco R1 (spike na primeira tarefa)**: confirmar contra Postgres real que (i) com
`DoNothing` + `Returning{}` o GORM devolve `RowsAffected` 1 na criação e 0 no conflito
sem erro, e (ii) a struct é preenchida pelo `RETURNING`. Se não for, cai-se em
`db.Raw(<SQL de hoje>).Scan(&row)` (consulta bruta do próprio GORM, prevista na spec).

## D3 — `valor_total` sem float

**Decisão**: `ValorTotal string` no modelo, `gorm:"column:valor_total"`. Leitura:
`Select("*, valor_total::text AS valor_total")` não é necessário se o pgx/stdlib entregar
`numeric` como `string` ao `Scan` — o `database/sql` converte `numeric` em texto. Gravação:
`gorm.Expr("?::decimal", valor)` apenas se o pgx recusar `string` para parâmetro
`numeric`.

**Rationale**: hoje o código lê `valor_total::text` e grava `$4::decimal`, exatamente para
nunca passar por `float64` (a coluna é `DECIMAL(10,2)` e o domínio usa `string`). O
modelo mantém `string`; o spike R2 decide se os casts explícitos continuam necessários.

**Risco R2 (spike na primeira tarefa)**: ida-e-volta de `"10.50"`, `"0.01"` e
`"99999999.99"` sem perda e sem notação científica, via `Create` e via `Find`. Se o
driver não entregar `string` ao ler `numeric`, usar `Select` com `valor_total::text` (e
`Raw`/`Scan` nas consultas de lista).

**Alternativas**: `decimal.Decimal` / `shopspring` — rejeitado: dependência nova sem
necessidade (princípio I); `float64` — rejeitado: perde precisão.

## D4 — Modelo e campos "vazio = NULL"

**Decisão**: uma struct privada `transacaoRow` em `modelos.go`, `TableName()` =
`transacoes_pagamento` (sem schema), tags `gorm:"column:...;primaryKey"`. Campos que o
domínio representa como string vazia mas o banco guarda como `NULL` — `FormaPagamento`,
`CodigoTransacaoGateway`, `MotivoFalha` — viram `*string` no modelo; a conversão trata
`""` ↔ `nil` (o que hoje fazem `coalesce(...,'')` na leitura e `nullif($n,'')` /
`formaOuNulo` na escrita). `PagoEm` já é `*time.Time`. Sem `gorm.Model`, sem associações,
sem `default:`/`autoCreateTime`/`autoUpdateTime` (`criado_em`/`atualizado_em` vêm do
domínio).

**Rationale**: as invariantes `forma_coerente_com_estado`, `pago_em_so_quando_pago` e
`forma_valida` do banco distinguem `NULL` de vazio; gravar `""` seria recusado (ou, pior,
aceito onde não devia).

**Alternativas**: tags `gorm` nas entidades do domínio — rejeitada (arquitetura);
`sql.NullString` — rejeitado: ponteiro é mais simples e já é o idioma de `PagoEm`.

## D5 — Esquema

**Decisão**: nenhum `AutoMigrate`. As migrações `000001` e `000002` seguem como única
fonte; o serviço não emite DDL. O teste de integração continua aplicando os `.up.sql`
(agora por `Banco.SQL().ExecContext`, sem argumentos, o que o pgx executa pelo protocolo
simples e aceita várias instruções — as migrações têm várias).

## D6 — Atualizações condicionais e cancelamento em lote

**Decisão**:

- `Finalizar`: `Model(&transacaoRow{}).Where("id = ? AND status = ?", id, "PROCESSANDO").
  Updates(map[string]any{status, codigo_transacao_gateway, motivo_falha, pago_em,
  atualizado_em})` com `nil` para os vazios e `RowsAffected == 0` → `ErrJaFinalizada`.
- `RegistrarEscolha`: idem, `WHERE id = ? AND status = 'AGUARDANDO_FORMA'`, forma `nil`
  quando vazia, `RowsAffected == 0` → `ErrJaFinalizada`.
- `ReivindicarCobranca`: `WHERE id = ? AND status = 'PROCESSANDO' AND cobranca_emitida =
  false`, `RowsAffected == 1`. `LiberarCobranca` e `MarcarAnunciado`: os `WHERE` de hoje,
  sem checar linhas afetadas (como hoje).
- `CancelarEsperasVencidas`: `UPDATE` com subconsulta `id IN (SELECT id ... ORDER BY
  expira_em LIMIT ? FOR UPDATE SKIP LOCKED)` e `RETURNING`, via
  `Clauses(clause.Returning{})` + `Model(&linhas)` + `Where("id IN (?)", sub)`, com
  `sub` montada com `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}`.

**Rationale**: `Updates` com **mapa** (não struct) evita o descarte silencioso de zero
values pelo GORM e permite gravar `NULL`. O SQL emitido é o de hoje; atomicidade e
exclusão de concorrentes continuam decididas pelo banco.

**Risco R3 (spike)**: o GORM compor `UPDATE ... WHERE id IN (subselect FOR UPDATE SKIP
LOCKED) RETURNING *` e preencher o slice. Se não compuser, usar
`db.Raw(<SQL de hoje>, ...).Scan(&linhas)` (consulta bruta do próprio GORM) — é a mesma
instrução, e o teste de concorrência de `vazao_test` cobre a exclusão entre varreduras.

**Alternativas**: `Save`/lock otimista — rejeitados: mudam a semântica.

## D7 — Listagens e leitura

**Decisão**: `BuscarPorReserva` = `Where("reserva_id = ?").Take(&row)`;
`gorm.ErrRecordNotFound` → `usecase.ErrNaoEncontrada`. `AguardandoCobranca` =
`Where("status = ? AND NOT cobranca_emitida", "PROCESSANDO").Order("criado_em").Limit(n).Find`.
`AnunciosPendentes` = `Where("status IN ? AND NOT resultado_anunciado",
[PAGO,RECUSADO,CANCELADO]).Order("atualizado_em").Limit(n).Find` — `PENDENTE_VERIFICACAO`
continua de fora de propósito. Sem resultado devolve slice sem itens (hoje: `nil`; quem
consome só itera).

## D8 — Verificação de arquitetura (SC-005) e erros

**Fronteira**: regra `depguard` `nucleo-sem-adaptadores`, igual à do `notificacao`
(`files`: `**/internal/domain/**` e `**/internal/usecase/**`; `deny`:
`.../pagamento/internal/adapter`, `.../pagamento/internal/platform`, `gorm.io`,
`github.com/jackc/pgx`, `github.com/rabbitmq/amqp091-go`, `net/http`,
`go.opentelemetry.io/otel`). Verificado: o núcleo atual importa só a stdlib
(`encoding/json`, `log/slog`, …) e seus próprios pacotes, então a regra passa limpa.
**Erros**: `gorm.ErrRecordNotFound` toma o lugar de `pgx.ErrNoRows`; demais erros sobem
como hoje (o código atual não embrulha, e o plano mantém assim). Banco fora do ar
continua sendo erro comum, preservando a trajetória de nova tentativa → fila morta.

## Resultado do spike (T004) — Postgres 16 real, 2026-10-07

Os três riscos foram verificados num teste descartável (removido) e **passaram com GORM
nativo, sem recorrer a consulta bruta**:

- **R1** — `Clauses(OnConflict{reserva_id, DoNothing}, Returning{}).Create`: `RowsAffected`
  1 na criação (struct preenchida pelo `RETURNING`; `"10.5"` voltou como `"10.50"`) e 0 no
  conflito, sem erro.
- **R2** — `ValorTotal string` sem `Select`/cast: `"10.50"`, `"0.01"` e `"99999999.99"`
  fazem ida-e-volta exatos por `Create` e `Take`; o driver entrega `numeric` como texto e
  aceita `string` como parâmetro. Decisão: **nenhum `::text`/`::decimal` explícito**.
- **R3** — `Model(&linhas).Clauses(Returning{}).Where("id IN (?)", sub).Updates(map)`, com
  `sub` usando `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}`: uma única
  instrução, `RowsAffected` 1 e o slice preenchido com a linha já `CANCELADO`.
