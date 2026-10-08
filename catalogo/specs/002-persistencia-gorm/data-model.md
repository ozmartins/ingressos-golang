# Data Model: Persistência do Catálogo via GORM

O esquema **não muda**. Este documento mapeia cada tabela (migrações `000001`–`000006`) para
o modelo privado do adaptador e para a entidade de domínio que ele alimenta. Os modelos vivem
em `internal/adapter/postgres/modelos.go`; o domínio permanece sem tags.

| Tabela (`catalogo.`) | Modelo GORM | Entidade de domínio | Observações |
|---|---|---|---|
| `filmes` | `filmeRow` | `catalogo.Filme` | `status` validado por `StatusFilme.Valido()` na leitura; `sinopse`, `imagem_url` anuláveis → `*string` se o domínio os tratar assim, hoje são `string` (confirmar no spike) |
| `cinemas` | `cinemaRow` | `catalogo.Cinema` | `ativo` default `TRUE`; sem `Preload` de salas |
| `salas` | `salaRow` | `catalogo.Sala` | `layout` JSONB `NOT NULL` ↔ `[]fileiraJSON`; índice único parcial `(cinema_id, numero) WHERE ativo` continua no banco |
| `sessoes` | `sessaoRow` | `catalogo.Sessao` | `preco_base` NUMERIC(10,2) ↔ `Dinheiro` via `big.Rat` |
| `outbox_eventos` | `outboxRow` | `usecase.FatoPendente` / `postgres.FatoNaCaixa` | `id` BIGSERIAL; `message_id` UNIQUE; `trace_context` anulável; índice parcial `idx_outbox_pendentes` |

Leitura composta (sem tabela própria): **grade de sessões** — projeção de
`sessoes ⨝ filmes ⨝ salas ⨝ cinemas` em `sessaoDetalhadaRow` → `catalogo.SessaoDetalhada`.

## Consultas (equivalência com o SQL de hoje)

| Operação | Hoje (pgx) | Com GORM |
|---|---|---|
| Listar filmes | `status = ANY($1) ORDER BY titulo, id` | `Where("status IN ?")`, `Order("titulo, id")` |
| Listar cinemas | `($1 IS NULL OR ativo = $1) ORDER BY nome, id` | `Where("ativo = ?")` só se filtro ≠ nil; `Order("nome, id")` |
| Listar salas | filtro opcional por cinema/ativo; `ORDER BY cinema_id, numero, id` | `Where` condicionais; mesma ordem |
| Consultar sessões | 4 JOINs, `status = ANY`, filtros filme/cinema/dia, `ORDER BY data_hora_inicio, id` | `Table("sessoes s").Joins(...)`, mesmos filtros e ordem |
| Total da página | `SELECT COUNT(*)` com os mesmos filtros | `Count(&n)` sobre a mesma consulta |
| Criar/atualizar sessão + fato | `emTransacao` + `INSERT/UPDATE` + `enfileirarFato` | `Transaction` + `Create/Updates` + `Create` com `OnConflict DoNothing` |
| Sala ocupada | `EXISTS(... data_hora_inicio + duracao * INTERVAL '1 minute' > $5)` | `Raw`/`Scan(&bool)` — expressão de intervalo sem equivalente nativo (spec, Assumptions) |
| Número de sala em uso | `EXISTS(... ativo AND id <> $3)` | `Raw` com `EXISTS` ou `Count > 0` |
| Drenar caixa | `FOR UPDATE SKIP LOCKED ... ORDER BY id LIMIT n` | `clause.Locking{UPDATE, SKIP LOCKED}` |

## Regras que o modelo preserva

- `atualizado_em = CURRENT_TIMESTAMP` explícito em todo `UPDATE` (não há timestamp automático do GORM).
- `Desativar`/`MarcarForaDeCartaz`/`Cancelar`: 0 linhas afetadas ⇒ `shared.NaoEncontrado`.
- Fato e efeito na mesma transação; falha do fato desfaz o efeito.
