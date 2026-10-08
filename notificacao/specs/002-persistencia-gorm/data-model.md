# Data Model: Persistência da Notificação via GORM

O esquema **não muda** (FR-006, FR-008): vale `migrations/000001_criar_ingressos.up.sql`.
Este documento só descreve como cada tabela é mapeada para um modelo GORM privado do
adaptador e qual consulta cada operação da porta emite.

## Modelos (internal/adapter/postgres/modelos.go)

### `ingressoRow` → tabela `ingressos_emitidos`

| Campo Go | Coluna | Tipo Go | Observação |
|---|---|---|---|
| ID | `id` | string | `primaryKey` |
| ReservaID | `reserva_id` | string | único no banco; alvo do `ON CONFLICT` |
| UsuarioID | `usuario_id` | string | |
| CodigoQR | `codigo_qr` | string | único no banco |
| Status | `status` | string | `VALIDO` / `UTILIZADO` / `CANCELADO` (CHECK no banco) |
| UtilizadoEm | `utilizado_em` | *time.Time | `NULL` enquanto não utilizado (CHECK de igualdade com o status) |
| CriadoEm | `criado_em` | time.Time | vem do domínio, não de hook do GORM |

### `avisoRow` → tabela `registros_notificacao`

| Campo Go | Coluna | Tipo Go | Observação |
|---|---|---|---|
| ID | `id` | string | `primaryKey` |
| IngressoID | `ingresso_id` | string | FK só na migração; sem associação GORM |
| UsuarioID | `usuario_id` | string | |
| Canal | `canal` | string | `EMAIL` / `PUSH` / `SMS` |
| Status | `status` | string | coluna `status` guarda o `Desfecho` (`ENVIADO` / `FALHA`) |
| Detalhes | `detalhes` | *string | `NULL` quando vazio; obrigatório na falha (CHECK) |
| EnviadoEm | `enviado_em` | time.Time | |

Nenhum campo usa `default:`, `autoCreateTime` ou `autoUpdateTime`.

## Operações → consultas

| Porta | GORM | SQL equivalente (o mesmo de hoje) |
|---|---|---|
| `Ingressos.CriarSeAusente` | `Clauses(OnConflict{reserva_id, DoNothing}).Create`; se `RowsAffected==0`, `Take(reserva_id)` | `INSERT ... ON CONFLICT (reserva_id) DO NOTHING`; depois `SELECT ... WHERE reserva_id = $1` |
| `Ingressos.Utilizar` | `Where(id, status='VALIDO').Updates(map)` | `UPDATE ... SET status='UTILIZADO', utilizado_em=$2 WHERE id=$1 AND status='VALIDO'` |
| `Ingressos.BuscarPorID` | `Take` | `SELECT ... WHERE id = $1` |
| `Ingressos.ListarPorUsuario` | `Where(usuario_id)` [+ `Where(status)`] + `Order` + `Find` | `SELECT ... WHERE usuario_id=$1 [AND status=$2] ORDER BY criado_em DESC, id DESC` |
| `Avisos.Registrar` | `Create(&avisoRow)` | `INSERT INTO registros_notificacao (...)` |

## Transições de estado

Inalteradas: `VALIDO → UTILIZADO` (por `Utilizar`, condicional) e `VALIDO → CANCELADO`
(domínio; não há operação de persistência hoje). `UTILIZADO`/`CANCELADO` são terminais.
