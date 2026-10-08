# Data Model: Persistência do Pagamento via GORM

O esquema **não muda** (FR-006, FR-008): valem `migrations/000001_criar_transacoes.up.sql`
e `migrations/000002_forma_escolhida_depois.up.sql`. Este documento só descreve como a
tabela é mapeada para um modelo GORM privado do adaptador e qual consulta cada operação
da porta emite.

## Modelo (internal/adapter/postgres/modelos.go)

### `transacaoRow` → tabela `transacoes_pagamento`

| Campo Go | Coluna | Tipo Go | Observação |
|---|---|---|---|
| ID | `id` | string | `primaryKey` |
| ReservaID | `reserva_id` | string | `UNIQUE` no banco; alvo do `ON CONFLICT` |
| UsuarioID | `usuario_id` | string | |
| ValorTotal | `valor_total` | string | `DECIMAL(10,2)`; nunca passa por float (D3) |
| FormaPagamento | `forma_pagamento` | *string | `NULL` até a escolha; `""` do domínio ↔ `nil` |
| Status | `status` | string | `AGUARDANDO_FORMA`/`PROCESSANDO`/`PAGO`/`RECUSADO`/`CANCELADO`/`PENDENTE_VERIFICACAO` (CHECK no banco) |
| CodigoTransacaoGateway | `codigo_transacao_gateway` | *string | `""` ↔ `nil` |
| MotivoFalha | `motivo_falha` | *string | `""` ↔ `nil` |
| CobrancaEmitida | `cobranca_emitida` | bool | `false` gravado explicitamente |
| ResultadoAnunciado | `resultado_anunciado` | bool | `false` gravado explicitamente |
| ExpiraEm | `expira_em` | time.Time | `NOT NULL` desde a migração 000002 |
| PagoEm | `pago_em` | *time.Time | `NULL` salvo em `PAGO` (CHECK `pago_em_so_quando_pago`) |
| CriadoEm | `criado_em` | time.Time | vem do domínio, não de hook do GORM |
| AtualizadoEm | `atualizado_em` | time.Time | idem |

Nenhum campo usa `default:`, `autoCreateTime` ou `autoUpdateTime`. Os `*string` existem
porque as invariantes `forma_coerente_com_estado`, `forma_valida` e
`pago_em_so_quando_pago` distinguem `NULL` de vazio.

O `INSERT` do GORM grava todas as colunas do modelo; para uma transação recém-criada isso
equivale ao `INSERT` de hoje (as colunas omitidas lá — forma, código, motivo, pago_em —
são `NULL` e as duas booleanas `false`, que é também o `DEFAULT`).

## Operações → consultas

| Porta | GORM | SQL equivalente (o mesmo de hoje) |
|---|---|---|
| `CriarSeAusente` | `Clauses(OnConflict{reserva_id, DoNothing}, Returning{}).Create`; se `RowsAffected==0`, `BuscarPorReserva` | `INSERT ... ON CONFLICT (reserva_id) DO NOTHING RETURNING ...`; depois `SELECT ... WHERE reserva_id = $1` |
| `BuscarPorReserva` | `Where(reserva_id).Take` | `SELECT ... WHERE reserva_id = $1`; `ErrRecordNotFound` → `ErrNaoEncontrada` |
| `RegistrarEscolha` | `Where(id, status='AGUARDANDO_FORMA').Updates(map)` | `UPDATE ... SET forma_pagamento, status, motivo_falha=nullif(...), atualizado_em WHERE id=$1 AND status='AGUARDANDO_FORMA'`; 0 linhas → `ErrJaFinalizada` |
| `ReivindicarCobranca` | `Where(id, status='PROCESSANDO', cobranca_emitida=false).Updates(map)` | `UPDATE ... SET cobranca_emitida=true, atualizado_em WHERE ...`; `RowsAffected==1` |
| `LiberarCobranca` | `Where(id, status='PROCESSANDO').Updates(map)` | `UPDATE ... SET cobranca_emitida=false, atualizado_em WHERE id=$1 AND status='PROCESSANDO'` |
| `Finalizar` | `Where(id, status='PROCESSANDO').Updates(map)` | `UPDATE ... SET status, codigo=nullif, motivo=nullif, pago_em, atualizado_em WHERE id=$1 AND status='PROCESSANDO'`; 0 linhas → `ErrJaFinalizada` |
| `MarcarAnunciado` | `Where(id, status IN (PAGO,RECUSADO,CANCELADO)).Updates(map)` | `UPDATE ... SET resultado_anunciado=true, atualizado_em WHERE ...` (sem checar linhas, como hoje) |
| `AguardandoCobranca` | `Where(status='PROCESSANDO' AND NOT cobranca_emitida).Order(criado_em).Limit.Find` | `SELECT ... ORDER BY criado_em LIMIT $1` |
| `CancelarEsperasVencidas` | `Clauses(Returning{}).Where(id IN subselect FOR UPDATE SKIP LOCKED).Updates(map)` | `UPDATE ... SET status='CANCELADO', motivo_falha, atualizado_em WHERE id IN (SELECT id ... WHERE status='AGUARDANDO_FORMA' AND expira_em <= $2 ORDER BY expira_em LIMIT $3 FOR UPDATE SKIP LOCKED) RETURNING ...` |
| `AnunciosPendentes` | `Where(status IN (PAGO,RECUSADO,CANCELADO) AND NOT resultado_anunciado).Order(atualizado_em).Limit.Find` | `SELECT ... ORDER BY atualizado_em LIMIT $1` |

## Conexão (internal/adapter/postgres/postgres.go)

`Banco{db *gorm.DB, sql *sql.DB}` com `Conectar(ctx, url)`, `DB()`, `SQL()`, `Fechar()` e
`Verificar(ctx)`. `Schema = "pagamento"` permanece. A prontidão (`/health/ready`) passa a
registrar `banco.Verificar` no lugar de `repo.Ping`.
