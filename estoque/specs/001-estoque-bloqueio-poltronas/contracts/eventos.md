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
  "valor_total": "84.00",
  "expira_em": "2026-08-29T21:43:00Z"
}
```

`poltronas_ids` traz os **mesmos rótulos recebidos na solicitação** (FR-016).

`valor_total` é texto decimal, e não número JSON, pelo mesmo motivo que o preço
é texto no catálogo: nenhum ponto flutuante binário representa centavo sem erro,
e este valor vira cobrança. Ele chega na solicitação de bloqueio e é repassado
sem interpretação — quem tem autoridade sobre o preço é o `Servico-Catalogo`,
dono do cadastro da sessão. Este serviço confere apenas o formato.

O campo é **adição compatível**: quem já consumia o fato sem ele continua
funcionando, e por isso a versão segue `1`.
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

**Produtor**: o `Servico-Catalogo`, dono do cadastro de sessões, publica este fato
ao criar uma sessão, a partir da planta da sala que ele guarda. O contrato acima
nasceu como proposta deste serviço e foi adotado por ele sem alteração; o lado de
lá está em
[`catalogo/.../contracts/eventos.md`](../../../../catalogo/specs/001-catalogo-sessoes-reserva/contracts/eventos.md).

A entrega é ao menos uma vez — o catálogo publica por caixa de saída —, e é o
`sessao_id` que descarta a repetição. Nada disso muda o efeito descrito acima: o
provisionamento já era idempotente por essa mesma chave.

Duas consequências que o produtor registra e que valem para quem lê este
contrato: o fato carrega a planta que a sala tinha no instante da criação da
sessão, e redesenhar a sala depois não o reemite; e alterar ou cancelar uma
sessão não emite fato algum, de modo que mover uma sessão já anunciada para outra
sala deixa a matriz provisionada aqui apontando para a planta antiga.

Para exercitar o consumo sem subir o catálogo, `make publicar-sessao` publica um
payload equivalente (ver `quickstart.md`).

## 5. Consumido — `sessao.cancelada`

**Fila**: `estoque.sessao-cancelada` · **Binding**: `sessao.cancelada`

```json
{
  "evento": "SESSAO_CANCELADA",
  "versao": 1,
  "ocorrido_em": "2026-09-06T18:20:00Z",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456"
}
```

**Efeito**: as reservas `PENDENTE` da sessão passam a `CANCELADA` e as poltronas
delas voltam a `LIVRE`; o prazo de cada uma sai do índice de expiração. Chave de
idempotência: `sessao_id`.

**As reservas `CONFIRMADA` não são tocadas.** Uma reserva confirmada é um
ingresso pago, e apagá-la porque a sessão caiu destruiria o registro de uma venda
sem devolver o dinheiro — reembolso não existe neste sistema. Elas são contadas, e
o serviço registra um aviso quando existem: uma sessão cancelada com ingresso
vendido é problema que alguém precisa resolver fora daqui.

**A matriz de poltronas permanece.** Ninguém reserva numa sessão que saiu da
grade, e apagá-la levaria consigo o histórico das confirmadas.

Sessão desconhecida, ou sem reserva pendente alguma, não é erro: não há o que
soltar, e a mensagem é confirmada.

**Não consumimos `sessao.alterada`**, e não há fila para ela. Com a sala imutável
do lado do produtor, nada que uma alteração mude — horário, idioma, preço — afeta
a matriz de poltronas. Sem binding, o exchange descarta a mensagem, que é o
destino certo de um fato sem interessado.

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
