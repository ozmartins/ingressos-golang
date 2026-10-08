# Research: Persistência da Notificação via GORM

Cada decisão foi checada contra o código atual (`internal/adapter/postgres/*.go`,
`migrations/000001_criar_ingressos.up.sql`, `test/integration/*`), não contra a spec 001.

## D1 — Como abrir a conexão

**Decisão**: `pgx.ParseConfig(url)` → `RuntimeParams["search_path"] = "notificacao"` →
`stdlib.OpenDB(*cfg)` → `*sql.DB` → `gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}),
&gorm.Config{...})` → `PingContext` explícito. O `*sql.DB` é guardado no `Banco` para
`Fechar` (`sqlDB.Close()`), `Verificar` (`PingContext`) e para os testes de integração
(`SQL()`).

**Rationale**: reproduz o que `Conectar` faz hoje — aceita URL e DSN chave=valor, fixa o
`search_path` (FR-007), falha cedo com `Ping` (mensagens `DATABASE_URL malformada` e
`alcançar o banco` preservadas) — e mantém o tipo de erro do pgx por baixo. O teste
`TestSchemaProprio` (que lê `current_setting('search_path')`) segue valendo sem mudança.

**Config do GORM**: `SkipDefaultTransaction: true` (cada escrita é uma instrução só);
`Logger: logger.Discard`; `DisableAutomaticPing: true`; sem `PrepareStmt`.

**Alternativas**: (a) `postgres.Open(dsn)` com `search_path` concatenado na URL —
rejeitada: quebra para DSN chave=valor; (b) `NamingStrategy{TablePrefix: "notificacao."}` —
rejeitada: qualifica tabelas no código, contra o desenho "migração qualifica, serviço usa
`search_path`".

**Pool**: o `pgxpool` tinha tamanho padrão `max(4, NumCPU)`; o `database/sql` não limita
por padrão. Para manter a ordem de grandeza (e não esgotar conexões do Postgres sob a
rajada dos testes `vazao*`), aplicar `SetMaxOpenConns(max(4, runtime.NumCPU()))` — o mesmo
teto de antes. Sem `ConnMaxIdleTime` (o pool atual não define um).

## D2 — Emissão idempotente e baixa atômica

**Decisão**:

- `CriarSeAusente`: `db.WithContext(ctx).Clauses(clause.OnConflict{Columns:
  []clause.Column{{Name: "reserva_id"}}, DoNothing: true}).Create(&row)`.
  `RowsAffected == 1` → criado; `0` → buscar a linha existente por `reserva_id`
  (`Take`; `gorm.ErrRecordNotFound` → `usecase.ErrNaoEncontrado`).
- `Utilizar`: `Model(&ingressoRow{}).Where("id = ? AND status = ?", id, "VALIDO").
  Updates(map[string]any{"status": "UTILIZADO", "utilizado_em": agora})` →
  `RowsAffected == 1`.

**Rationale**: o SQL emitido é o mesmo de hoje; a unicidade segue decidida pelo índice
único e a baixa pela cláusula `WHERE`. Conflito em outra restrição (`codigo_qr`) continua
sendo erro, porque o alvo do `ON CONFLICT` é só `reserva_id`. `Updates` com mapa (não
struct) evita o descarte silencioso de zero values pelo GORM. Os dados que o GORM
reescreveria sozinho (`gorm.Model`, `CreatedAt`) não existem aqui: `criado_em` vem do
domínio.

**Alternativas**: `Save`/`FirstOrCreate` — rejeitados: `FirstOrCreate` faz SELECT+INSERT
(corrida entre os dois) e `Save` faz upsert, o que quebraria a idempotência; lock
otimista (`version`) — rejeitado, muda a semântica.

**Risco a verificar cedo (spike da primeira tarefa)**: com `DoNothing`, o GORM pode
omitir o `RETURNING` e devolver `RowsAffected` 0 no conflito — é o comportamento
esperado; confirmar contra Postgres real que `RowsAffected` é 1 na criação e 0 no
conflito. Se não for, cai-se em `Exec` bruto do próprio GORM com o mesmo SQL de hoje
(previsto na spec).

## D3 — Modelos

**Decisão**: duas structs privadas em `modelos.go` — `ingressoRow` (`ingressos_emitidos`)
e `avisoRow` (`registros_notificacao`) — com `TableName()` sem schema e
`gorm:"column:...;primaryKey"`; conversão explícita modelo ↔ domínio
(`ingresso.Ingresso`, `aviso.Registro`) nas bordas. Campos opcionais como ponteiros
(`UtilizadoEm *time.Time`, `Detalhes *string`) para gravar `NULL`, não zero value.
Sem `Preload`, sem associações (a FK `ingresso_id` continua só na migração), sem
`gorm.Model`, sem `default:` nas tags (o app sempre informa todos os campos, então o
GORM nunca "pula" um valor zero).

**Rationale**: o domínio não ganha tags nem import; `status` e `canal` são `string` no
modelo e convertidos para os tipos do domínio na borda, como o `Scan` faz hoje.

**Alternativas**: tags `gorm` nas entidades de domínio — rejeitada (arquitetura);
pacote `models` separado — rejeitado, indireção desnecessária (princípio I).

## D4 — Esquema

**Decisão**: nenhum `AutoMigrate`. A migração `000001` segue como única fonte; o serviço
não emite DDL. O teste de integração continua aplicando o `.up.sql` pelo `SQL()` (um
`ExecContext` sem argumentos, que o pgx executa pelo protocolo simples e aceita várias
instruções).

## D5 — Listagem e leitura

**Decisão**: `ListarPorUsuario` = `Where("usuario_id = ?", id)` + `Where("status = ?",
filtro)` somente quando `filtro != ""` + `Order("criado_em DESC, id DESC")` + `Find`; o
resultado é inicializado como slice vazio (não `nil`) antes do `Find`, para manter "lista
vazia, não nula" (borda da spec). `BuscarPorID`/por reserva: `Take`, tradução de
`gorm.ErrRecordNotFound` para `usecase.ErrNaoEncontrado`.

**Rationale**: o índice `ingressos_por_pessoa (usuario_id, criado_em DESC)` continua
servindo a consulta; a ordem estável (`id` como desempate) é preservada. O truque
`($2 = '' OR status = $2)` do SQL atual vira condicional em Go — mesmo resultado, plano de
consulta mais simples.

## D6 — Verificação de arquitetura (SC-005)

**Decisão**: regra `depguard` `nucleo-sem-adaptadores` no `.golangci.yml`, igual à do
`catalogo` (`files`: `**/internal/domain/**` e `**/internal/usecase/**`; `deny`:
`internal/adapter`, `internal/platform`, `gorm.io`, `github.com/jackc/pgx`,
`github.com/rabbitmq/amqp091-go`, `net/http`, `go.opentelemetry.io/otel`). Verificado: o
núcleo atual importa só a stdlib e seus próprios pacotes, então a regra passa limpa.

**Rationale**: decisão do mantenedor. Sem código novo de teste; reaproveita o linter que o
serviço já roda. A regra morde: um `import _ "gorm.io/gorm"` em `internal/usecase` é
reprovado pelo `golangci-lint`.

**Alternativas**: (a) `test/arquitetura_test.go` no padrão do `estoque` (`go list -json`) —
foi a escolha inicial; falharia já em `make test`, mas custa ≈40 linhas de teste;
(b) só inspeção manual de imports — rejeitada: não impede regressão.

## D7 — Tradução de erros

**Decisão**: `gorm.ErrRecordNotFound` → `usecase.ErrNaoEncontrado` (onde hoje é
`pgx.ErrNoRows`); demais erros continuam embrulhados com `fmt.Errorf("...: %w", err)` e as
mesmas mensagens de contexto ("inserir ingresso", "dar baixa no ingresso", "buscar
ingresso", "listar ingressos", "registrar aviso"). Falha de conexão (banco fora do ar)
continua sendo erro comum, o que mantém a trajetória de nova tentativa → fila morta
coberta por `TestFalhaTransitoriaRetenta...`.
