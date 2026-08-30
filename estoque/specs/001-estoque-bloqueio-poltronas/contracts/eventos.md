# Contrato de Eventos — Servico-Estoque (RabbitMQ)

Exchange única: **`cinema.eventos`** (tipo `topic`, durável). Todas as mensagens
são JSON, `content_type: application/json`, `delivery_mode: 2` (persistente).

Campos de envelope obrigatórios em toda mensagem publicada por este serviço:
`evento` (nome do fato), `versao` (inteiro, começa em 1), `ocorrido_em` (RFC 3339
em UTC). O `message_id` AMQP carrega a chave de idempotência do fato.

Toda mensagem publicada e consumida carrega, nos **headers AMQP**, o contexto de
rastreamento W3C: `traceparent` e, quando presente, `tracestate` (FR-044). O
publicador reinjeta o contexto capturado no instante em que o fato ocorreu, não o
do instante da publicação — a caixa de saída publica de forma assíncrona.

## 1. Publicado — `reserva.criada`

**Routing key**: `reserva.criada` · **Requisitos**: FR-016..FR-018 · **`message_id`**: `reserva_id`

```json
{
  "evento": "RESERVA_CRIADA",
  "versao": 1,
  "ocorrido_em": "2026-08-29T21:33:00Z",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456",
  "usuario_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
  "poltronas_ids": ["A1", "A2"],
  "expira_em": "2026-08-29T21:43:00Z"
}
```

`poltronas_ids` traz os **mesmos rótulos recebidos na solicitação** (FR-016).
A publicação passa pela caixa de saída transacional: a reserva é persistida
primeiro e o evento é reenviado até ser aceito pelo broker (FR-018), portanto a
entrega é **ao menos uma vez** — consumidores devem deduplicar por `reserva_id`.

## 2. Consumido — `pagamento.sucesso`

**Fila**: `estoque.pagamento-sucesso` · **Binding**: `pagamento.sucesso` · **Requisitos**: FR-019, FR-021..FR-024

```json
{
  "evento": "PAGAMENTO_SUCESSO",
  "versao": 1,
  "ocorrido_em": "2026-08-29T21:36:12Z",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192"
}
```

**Efeito**: reserva `PENDENTE` → `CONFIRMADA`, poltronas vinculadas → `OCUPADA`.
Chave de idempotência: `reserva_id`.

## 3. Consumido — `pagamento.falhou`

**Fila**: `estoque.pagamento-falhou` · **Binding**: `pagamento.falhou` · **Requisitos**: FR-020..FR-024

```json
{
  "evento": "PAGAMENTO_FALHOU",
  "versao": 1,
  "ocorrido_em": "2026-08-29T21:36:12Z",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "motivo": "CARTAO_RECUSADO"
}
```

**Efeito**: reserva `PENDENTE` → `CANCELADA`, poltronas → `LIVRE`, índice de
expiração liberado. `motivo` é informativo e não altera o efeito.

## 4. Consumido — `sessao.criada`

**Fila**: `estoque.sessao-criada` · **Binding**: `sessao.criada` · **Requisitos**: FR-033..FR-036

```json
{
  "evento": "SESSAO_CRIADA",
  "versao": 1,
  "ocorrido_em": "2026-08-29T18:00:00Z",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456",
  "sala_id": "b2c3d4e5-1111-4222-8333-444455556666",
  "poltronas": [
    { "fileira": "A", "numero": 1, "tipo": "NORMAL" },
    { "fileira": "A", "numero": 2, "tipo": "NORMAL" },
    { "fileira": "F", "numero": 7, "tipo": "PCD" }
  ]
}
```

**Efeito**: provisiona a matriz da sessão, todas as poltronas em `LIVRE`.
Chave de idempotência: `sessao_id`. `tipo` aceita `NORMAL`, `PCD`, `NAMORADEIRA`;
valor desconhecido invalida a mensagem inteira (FR-035 — tudo-ou-nada).

**Dependência de integração**: este fato ainda não é publicado pelo
`Servico-Catalogo`, que é o dono do cadastro de sessões. Até que passe a
publicá-lo, a matriz é populada por carga administrativa (ver `quickstart.md`).
O contrato acima é a proposta deste serviço ao catálogo.

## Regras de consumo (todas as filas)

- **Ack manual, depois do commit**: a mensagem só é confirmada após a transação
  que aplica o efeito ser confirmada no banco (FR-024).
- **Idempotência**: tabela `mensagens_processadas` registra `(fila, message_id)`;
  reentrega já processada é confirmada sem reexecutar (FR-021, FR-034).
- **Guarda de estado**: mesmo sem registro de idempotência, as transições só se
  aplicam a partir de `PENDENTE` (FR-011), o que torna ordem invertida e
  duplicata inofensivas.
- **Sem retentativa infinita**: falha definitiva (JSON inválido, campo ausente,
  `reserva_id` desconhecido) vai para `estoque.<nome>.dlq` via
  `cinema.eventos.dlx`, com o motivo em header (FR-023). Falha transitória
  (banco fora) é devolvida à fila com atraso.
- **Rastreamento**: o contexto é extraído dos headers e o span de consumo é
  aberto como filho dele, ligando o bloqueio ao desfecho de pagamento (SC-009).
- **Prefetch** limitado (padrão 32) para não acumular trabalho não confirmado.

## Topologia declarada na largada

| Recurso | Tipo | Observação |
|---|---|---|
| `cinema.eventos` | exchange topic durável | compartilhada pelo sistema |
| `cinema.eventos.dlx` | exchange topic durável | destino das mensagens descartadas |
| `estoque.pagamento-sucesso` | fila durável | binding `pagamento.sucesso`, DLX configurada |
| `estoque.pagamento-falhou` | fila durável | binding `pagamento.falhou`, DLX configurada |
| `estoque.sessao-criada` | fila durável | binding `sessao.criada`, DLX configurada |
| `estoque.*.dlq` | filas duráveis | uma por fila de origem, sem consumidor automático |

A declaração é idempotente e roda na inicialização; o processo recusa subir se
a topologia não puder ser garantida.
