# Fase 1 — Modelo de Dados

**Feature**: Bloqueio, Confirmação e Liberação de Poltronas (`Servico-Estoque`)
**Armazenamento**: PostgreSQL 16 (fonte de verdade) · Redis 7 (índice de prazo, sem estado autoritativo)

---

## 1. Visão geral

Cinco tabelas. As três primeiras vêm da ERS; as duas últimas existem para
sustentar requisitos que a ERS enuncia mas não modela (entrega confiável do
evento e idempotência do consumo).

```text
sessao (externa, não persistida)
   └── poltronas ──< reserva_poltronas >── reservas
                                              │
outbox_eventos (entrega de reserva.criada)    │
mensagens_processadas (idempotência do consumo)
```

---

## 2. `poltronas`

| Coluna | Tipo | Regras |
|---|---|---|
| `id` | `VARCHAR(36)` PK | UUID v5 determinístico de `sessao_id \| fileira \| numero` (D6) |
| `sessao_id` | `VARCHAR(36)` NOT NULL | identificador da sessão no catálogo |
| `fileira` | `VARCHAR(5)` NOT NULL | ex.: `A` |
| `numero` | `INT` NOT NULL | ex.: `1`, positivo |
| `rotulo` | `VARCHAR(10)` NOT NULL | `fileira \|\| numero`, identidade de negócio (FR-027) |
| `tipo` | `VARCHAR(50)` NOT NULL | `NORMAL` \| `PCD` \| `NAMORADEIRA` |
| `status` | `VARCHAR(50)` NOT NULL | `LIVRE` \| `RESERVADA` \| `OCUPADA`, padrão `LIVRE` |
| `criado_em` | `TIMESTAMPTZ` NOT NULL | padrão `now()` |
| `atualizado_em` | `TIMESTAMPTZ` NOT NULL | atualizado a cada transição (FR-032) |

**Restrições**
- `uk_sessao_poltrona UNIQUE (sessao_id, fileira, numero)` — FR-026 (da ERS).
- `uk_sessao_rotulo UNIQUE (sessao_id, rotulo)` — FR-027; torna o rótulo endereçável.
- `CHECK (status IN ('LIVRE','RESERVADA','OCUPADA'))`, `CHECK (tipo IN (...))`,
  `CHECK (numero > 0)`.

**Índices**
- `idx_poltronas_sessao (sessao_id)` — carrega o mapa da sessão (FR-029).
- `idx_poltronas_sessao_rotulo (sessao_id, rotulo)` — atendido pela unicidade acima;
  é por ele que o `SELECT ... FOR UPDATE` do bloqueio encontra as linhas.

**Adição em relação à ERS**: a coluna `rotulo`. Justificativa: sem ela, todo
bloqueio precisaria decompor `"A1"` em fileira e número em SQL, o que impede uso
de índice. Guardar o rótulo é redundância derivada e barata, protegida por
unicidade.

---

## 3. `reservas`

| Coluna | Tipo | Regras |
|---|---|---|
| `id` | `VARCHAR(36)` PK | UUID v4 gerado na concessão |
| `sessao_id` | `VARCHAR(36)` NOT NULL | |
| `usuario_id` | `VARCHAR(36)` NOT NULL | `sub` repassado pelo catálogo (FR-038) |
| `expira_em` | `TIMESTAMPTZ` NOT NULL | concessão + `RESERVA_TTL` (padrão 10 min) — FR-007 |
| `status` | `VARCHAR(50)` NOT NULL | `PENDENTE` \| `CONFIRMADA` \| `EXPIRADA` \| `CANCELADA` |
| `criado_em` | `TIMESTAMPTZ` NOT NULL | padrão `now()` |
| `finalizado_em` | `TIMESTAMPTZ` NULL | instante da transição para estado final |

**Restrições**: `CHECK (status IN (...))`; `CHECK (expira_em > criado_em)`;
`CHECK ((status = 'PENDENTE') = (finalizado_em IS NULL))`.

**Índices**
- `idx_reservas_expiracao (expira_em) WHERE status = 'PENDENTE'` — índice parcial
  que sustenta a varredura de expiração (D4) sem varrer reservas finalizadas.
- `idx_reservas_sessao_usuario (sessao_id, usuario_id)` — auditoria e suporte.

**Adição em relação à ERS**: `finalizado_em`. Justificativa: sem ela não há como
auditar quando uma reserva saiu de `PENDENTE`, exigência implícita de FR-022 e
FR-039.

---

## 4. `reserva_poltronas`

| Coluna | Tipo | Regras |
|---|---|---|
| `reserva_id` | `VARCHAR(36)` NOT NULL | FK → `reservas(id)` |
| `poltrona_id` | `VARCHAR(36)` NOT NULL | FK → `poltronas(id)` |

PK composta `(reserva_id, poltrona_id)`, conforme a ERS.

**Índice adicional**: `idx_reserva_poltronas_poltrona (poltrona_id)` — necessário
para responder "qual reserva prende esta poltrona" sem varredura, usado na
liberação e na auditoria de divergência.

**Invariante de negócio**: uma poltrona pertence a no máximo **uma** reserva não
finalizada por vez. Isso não é expressável em constraint declarativa; é garantido
pelo protocolo transacional (§7) — a poltrona só entra em um vínculo novo quando
está `LIVRE`, e só volta a `LIVRE` quando a reserva que a prende é finalizada.
Existe um teste de integração dedicado a essa invariante (SC-006).

---

## 5. `outbox_eventos`

| Coluna | Tipo | Regras |
|---|---|---|
| `id` | `BIGSERIAL` PK | ordem de publicação |
| `message_id` | `VARCHAR(64)` NOT NULL UNIQUE | `reserva_id` — chave de deduplicação do consumidor |
| `routing_key` | `VARCHAR(120)` NOT NULL | `reserva.criada` |
| `payload` | `JSONB` NOT NULL | corpo conforme `contracts/eventos.md` |
| `criado_em` | `TIMESTAMPTZ` NOT NULL | padrão `now()` |
| `publicado_em` | `TIMESTAMPTZ` NULL | preenchido após confirmação do broker |
| `trace_context` | `JSONB` NULL | contexto W3C (`traceparent`/`tracestate`) capturado na concessão, reinjetado pelo publicador (FR-044) |
| `tentativas` | `INT` NOT NULL | padrão 0, incrementado a cada falha |

**Índice**: `idx_outbox_pendentes (id) WHERE publicado_em IS NULL`.

**Razão de existir**: FR-018 e SC-005 — persistir antes de anunciar e reenviar
até conseguir, sem duplicar reserva e sem prender a resposta síncrona (D3).

`trace_context` existe porque o publicador roda depois e fora da requisição que
concedeu o bloqueio: sem guardar o contexto, o span de publicação nasceria órfão
e SC-009 (reconstituição ponta a ponta) seria inatingível.

---

## 6. `mensagens_processadas`

| Coluna | Tipo | Regras |
|---|---|---|
| `fila` | `VARCHAR(120)` NOT NULL | nome da fila de origem |
| `message_id` | `VARCHAR(64)` NOT NULL | `reserva_id` ou `sessao_id` |
| `processado_em` | `TIMESTAMPTZ` NOT NULL | padrão `now()` |

PK composta `(fila, message_id)`.

**Razão de existir**: FR-021 e FR-034 (D5). Conflito de chave = reentrega já
aplicada; a mensagem é confirmada sem reexecutar.

**Retenção**: linhas mais antigas que 30 dias são removidas por rotina de
limpeza; a guarda de máquina de estados continua protegendo mesmo sem o registro.

---

## 7. Máquinas de estado

### Reserva (FR-010, FR-011)

```text
            concessão do bloqueio
                    │
                    ▼
              ┌───────────┐
              │ PENDENTE  │
              └─────┬─────┘
      pagamento.sucesso │ pagamento.falhou │ prazo vencido
                    │                  │            │
                    ▼                  ▼            ▼
            ┌────────────┐     ┌───────────┐  ┌──────────┐
            │ CONFIRMADA │     │ CANCELADA │  │ EXPIRADA │
            └────────────┘     └───────────┘  └──────────┘
                        (estados finais — imutáveis)
```

Toda transição é `UPDATE reservas SET status = $novo, finalizado_em = now()
WHERE id = $id AND status = 'PENDENTE'`. Zero linhas afetadas significa "já
finalizada": o efeito é ignorado e a divergência registrada (FR-022).

### Poltrona (FR-010, FR-014)

```text
  LIVRE ──bloqueio concedido──▶ RESERVADA ──pagamento.sucesso──▶ OCUPADA
    ▲                               │
    └──── falha de pagamento ───────┤
    └──── expiração do prazo ───────┘
```

`OCUPADA` é terminal nesta feature (assumption da spec: não há devolução ao
estoque após a confirmação). Toda transição de poltrona acontece **na mesma
transação** da transição da reserva que a motiva (FR-015).

---

## 8. Protocolo transacional do bloqueio (FR-002..FR-009)

```sql
BEGIN;

-- 1. Trava as linhas em ordem determinística de rótulo (evita deadlock entre
--    solicitações com conjuntos que se cruzam). NOWAIT: preferimos recusar
--    rápido a esperar, para respeitar o orçamento de 100 ms (SC-001).
SELECT id, rotulo, status
  FROM poltronas
 WHERE sessao_id = $1 AND rotulo = ANY($2)
 ORDER BY rotulo
   FOR UPDATE NOWAIT;

-- 2. Verificações, no aplicativo:
--    contagem != len(rótulos)  -> FAILED_PRECONDITION (rótulo inexistente)
--    alguma status != 'LIVRE'  -> resposta sucesso=false (indisponível)

-- 3. INSERT reservas (PENDENTE, expira_em = now() + ttl)
-- 4. INSERT reserva_poltronas (uma linha por poltrona)
-- 5. UPDATE poltronas SET status='RESERVADA', atualizado_em=now() WHERE id = ANY(...)
-- 6. INSERT outbox_eventos (reserva.criada)

COMMIT;
-- 7. Fora da transação: SET reserva:{id} EX ttl no Redis (índice de prazo).
--    Falha aqui é registrada e ignorada — a varredura cobre (D4).
```

Validações **antes** de abrir a transação (nenhum recurso de banco gasto com
solicitação inválida): lista não vazia, sem rótulos repetidos, tamanho dentro de
`POLTRONAS_MAX_POR_BLOQUEIO`, `usuario_id` presente, formato de rótulo válido.

**Sessão não provisionada** (FR-036): a etapa 1 retornando zero linhas para uma
sessão sem nenhuma poltrona é `FAILED_PRECONDITION`, distinto de indisponibilidade.

---

## 9. Consultas de suporte

| Operação | Consulta | Índice usado |
|---|---|---|
| Mapa da sessão (FR-029) | `SELECT rotulo, fileira, numero, tipo, status FROM poltronas WHERE sessao_id=$1 ORDER BY fileira, numero` | `idx_poltronas_sessao` |
| Confirmação (FR-019) | `UPDATE reservas ... WHERE id=$1 AND status='PENDENTE'` + `UPDATE poltronas SET status='OCUPADA' WHERE id IN (SELECT poltrona_id FROM reserva_poltronas WHERE reserva_id=$1)` | PK + `idx_reserva_poltronas_*` |
| Cancelamento (FR-020) | idem, com `status='LIVRE'` | idem |
| Varredura de expiração (D4) | `UPDATE reservas SET status='EXPIRADA', finalizado_em=now() WHERE status='PENDENTE' AND expira_em < now() RETURNING id` | `idx_reservas_expiracao` |
| Outbox pendente (D3) | `SELECT ... WHERE publicado_em IS NULL ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED` | `idx_outbox_pendentes` |

`FOR UPDATE SKIP LOCKED` na caixa de saída permite mais de uma instância
publicando sem republicar a mesma linha.

---

## 10. Entidades do núcleo de domínio

Espelham as tabelas, sem tipos de banco:

- **`Poltrona`**: `Rotulo`, `Fileira`, `Numero`, `Tipo`, `Status`. Conhece as
  transições válidas e recusa as demais.
- **`Reserva`**: `ID`, `SessaoID`, `UsuarioID`, `ExpiraEm`, `Status`, `Poltronas`.
  Sabe responder `PodeConfirmar()`, `PodeCancelar()`, `Expirou(agora)`.
- **`SolicitacaoBloqueio`** (objeto de valor): valida lista não vazia, sem
  repetição, dentro do limite, com usuário presente — antes de tocar o banco.
- **`Relogio`** (porta): injetado, para que expiração seja testável sem espera real.
