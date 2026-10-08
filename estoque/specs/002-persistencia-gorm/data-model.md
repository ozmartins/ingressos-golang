# Data Model: mapeamento para o GORM

O **esquema não muda** (migrações `000001`–`000003`). Este documento só descreve como
cada tabela é vista pelo adaptador. Fonte: `migrations/*.up.sql`.

| Tabela (`estoque.`) | Modelo (privado) | Colunas especiais |
|---|---|---|
| `poltronas` | `poltronaRow` | `id` PK string(36); `criado_em` pelo default do banco (`<-:false`); `atualizado_em` com `<-:update` — nunca escrito na criação, sempre `Expr("now()")` nos `UPDATE` |
| `reservas` | `reservaRow` | `valor_total` `*string` (DECIMAL anulável); `finalizado_em` `*time.Time`; checks `ck_reserva_*` seguem no banco |
| `reserva_poltronas` | `reservaPoltronaRow` | PK composta `(reserva_id, poltrona_id)` |
| `outbox_eventos` | `outboxRow` | `id` BIGSERIAL (autoIncrement); `payload`/`trace_context` JSONB como `[]byte`; `publicado_em` `*time.Time` |
| `mensagens_processadas` | `mensagemProcessadaRow` | PK composta `(fila, message_id)`; `processado_em` default do banco |

Regras do modelo:
- `TableName()` devolve o nome **sem** schema (o `search_path` resolve).
- Sem `gorm.Model`, sem `DeletedAt`, sem `CreatedAt/UpdatedAt` (não há soft delete e
  os carimbos são controlados explicitamente).
- Sem associações declaradas; `reserva_poltronas` é gravada e consultada como tabela.

## Consultas (equivalência com o SQL atual)

| Operação | Hoje (pgx) | Com GORM |
|---|---|---|
| Travar poltronas do pedido | `SELECT … ORDER BY rotulo FOR UPDATE NOWAIT` | `Clauses(clause.Locking{Strength:"UPDATE",Options:"NOWAIT"}).Where(…IN ?).Order("rotulo").Find` |
| Criar reserva + vínculos + marcar poltronas | `INSERT`, N× `INSERT`, `UPDATE … ANY` | `Create(&reservaRow)`, `Create(&[]reservaPoltronaRow)`, `Model(&poltronaRow{}).Where("id IN ?").Updates(map{status, atualizado_em: Expr("now()")})` |
| Fato de saída | `INSERT … ON CONFLICT (message_id) DO NOTHING` | `Clauses(clause.OnConflict{Columns:[message_id], DoNothing:true}).Create(&outboxRow)` |
| Registrar mensagem processada | `INSERT … ON CONFLICT DO NOTHING` + `RowsAffected` | idem com `clause.OnConflict` + `RowsAffected == 1` |
| Confirmar/cancelar | `UPDATE reservas … WHERE id AND status='PENDENTE'` + `RowsAffected` | `Model(&reservaRow{}).Where(…).Updates(map)` + `RowsAffected` |
| Varredura de expiração | `UPDATE … WHERE id IN (SELECT … FOR UPDATE SKIP LOCKED) RETURNING id` + advisory lock | idem (ver research D2, com fallback `Raw`) |
| Publicar outbox | `SELECT … FOR UPDATE SKIP LOCKED` / `UPDATE publicado_em` | `Locking{Options:"SKIP LOCKED"}` / `Update("publicado_em", Expr("now()"))` |
| Mapa da sessão | `SELECT … ORDER BY fileira, numero` | `Model(&poltronaRow{}).Select(…).Where(sessao_id).Order("fileira, numero").Rows()` + `Scan` direto para `poltrona.Poltrona` (medido: `Find` em `[]poltronaRow` triplicava o p99 da consulta de 500 lugares sob `-race`; ver quickstart) |
| Provisionar matriz | N× `INSERT … ON CONFLICT DO NOTHING` | `CreateInBatches(rows, 500)` com `OnConflict{DoNothing:true}`; `RowsAffected == 0` → duplicata |

## Máquinas de estado

Inalteradas (`reserva`: PENDENTE → CONFIRMADA | EXPIRADA | CANCELADA; `poltrona`:
LIVRE → RESERVADA → OCUPADA | LIVRE). Continuam vivendo no domínio e protegidas por
`WHERE status = 'PENDENTE'` na escrita.
