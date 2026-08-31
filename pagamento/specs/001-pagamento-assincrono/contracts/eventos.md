# Contrato de Eventos — Servico-Pagamento (RabbitMQ)

Exchange única: **`cinema.eventos`** (`topic`, durável), compartilhada pelo sistema.
Mensagens em JSON, `content_type: application/json`, `delivery_mode: 2`.

Envelope obrigatório em toda mensagem publicada: `evento`, `versao` (inteiro,
começa em 1), `ocorrido_em` (RFC 3339 em UTC). O `message_id` AMQP carrega a chave
de idempotência do fato. Os cabeçalhos levam o contexto de rastreamento W3C
(`traceparent`, e `tracestate` quando presente), extraído da mensagem que originou
o processamento e reinjetado nos fatos publicados.

---

## 1. Consumido — `reserva.criada`

**Fila**: `pagamento.reserva-criada` · **Binding**: `reserva.criada` ·
**Requisitos**: FR-001..FR-008 · **Chave de idempotência**: `reserva_id`

```json
{
  "evento": "RESERVA_CRIADA",
  "versao": 1,
  "ocorrido_em": "2026-08-29T21:33:00Z",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456",
  "usuario_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
  "poltronas_ids": ["A1", "A2"],
  "valor_total": 84.00,
  "forma_pagamento": "PIX",
  "expira_em": "2026-08-29T21:43:00Z"
}
```

**Campos exigidos**: `reserva_id`, `usuario_id`, `valor_total` (> 0),
`forma_pagamento` (`PIX` ou `CARTAO_CREDITO`), `expira_em`. Ausência, valor não
positivo ou forma desconhecida invalidam a mensagem inteira (FR-003, FR-004).
`sessao_id` e `poltronas_ids` são aceitos e **ignorados** — nenhum requisito os usa
e este serviço não é dono desse estado. Campos extras não invalidam a mensagem.

> ### ⚠ Dependência de integração — este fato ainda não é publicado assim
>
> O `Servico-Estoque` publica `reserva.criada` **sem** `valor_total` e sem
> `forma_pagamento`: ver `estoque/internal/usecase/bloquear_poltronas.go:15`
> (`EventoReservaCriada`, oito campos) e `estoque/proto/estoque.proto:30`
> (`SolicitacaoBloqueio`, que sequer recebe esses dados do catálogo).
>
> A divergência foi levada ao mantenedor em 2026-08-30 (princípio IV) e a
> resolução escolhida foi manter este contrato como o que o pagamento **exige**,
> tratando a lacuna como dependência de integração — o mesmo padrão que o estoque
> usa para `sessao.criada`, que o catálogo ainda não publica.
>
> **Enquanto o estoque não propagar os dois campos**, todo evento vindo dele é
> inválido para este serviço e vai para a fila morta. A validação ponta a ponta
> usa o publicador manual `cmd/publicar` (ver `quickstart.md`). Fechar a lacuna
> exige, no estoque: os dois campos em `SolicitacaoBloqueio`, na tabela `reservas`
> e em `reserva.criada` v2 — trabalho fora do escopo desta feature.

---

## 2. Publicado — `pagamento.sucesso`

**Routing key**: `pagamento.sucesso` · **Requisitos**: FR-010, FR-012, FR-014 ·
**`message_id`**: `transacao_id`

```json
{
  "evento": "PAGAMENTO_SUCESSO",
  "versao": 1,
  "ocorrido_em": "2026-08-29T21:35:10Z",
  "transacao_id": "e402a129-8812-4211-b123-000129381293",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "usuario_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
  "valor_total": 84.00,
  "pago_em": "2026-08-29T21:35:10Z"
}
```

## 3. Publicado — `pagamento.falhou`

**Routing key**: `pagamento.falhou` · **Requisitos**: FR-010, FR-011, FR-014 ·
**`message_id`**: `transacao_id`

```json
{
  "evento": "PAGAMENTO_FALHOU",
  "versao": 1,
  "ocorrido_em": "2026-08-29T21:35:10Z",
  "transacao_id": "e402a129-8812-4211-b123-000129381293",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "usuario_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
  "motivo": "SALDO_INSUFICIENTE"
}
```

`motivo` pertence ao vocabulário fechado de `data-model.md` §3:
`RESERVA_EXPIRADA`, `SALDO_INSUFICIENTE`, `CARTAO_RECUSADO`,
`RECUSADO_PELO_ADQUIRENTE`. Consumidores devem tolerar motivo desconhecido.

**Compatibilidade verificada com o consumidor real**: o `Servico-Estoque` lê apenas
`reserva_id` (e `motivo`, de forma informativa) e ignora campos desconhecidos —
`estoque/specs/001-estoque-bloqueio-poltronas/contracts/eventos.md` §2 e §3. Os
payloads acima são superconjuntos do que ele espera; nenhuma mudança é exigida lá.

**Transação `PENDENTE_VERIFICACAO` não publica nada** (FR-010, FR-022). É o único
desfecho em que o sistema fica em silêncio, deliberadamente.

---

## 4. Regras de consumo

- **Ordem de execução, sempre**: gravar estado final → publicar com *publisher
  confirm* → marcar `resultado_anunciado` → confirmar a mensagem (`ack`). A
  mensagem nunca é confirmada antes da publicação (FR-014).
- **Idempotência**: a unicidade de `reserva_id` decide quem cobra. Reentrega de
  reserva já em estado terminal **com** anúncio feito é confirmada sem efeito;
  **sem** anúncio feito republica o resultado a partir do que está gravado, sem
  tocar no adquirente.
- **Entrega ao menos uma vez**: uma queda entre publicar e marcar republica o
  resultado. Consumidores devem deduplicar por `reserva_id`. Este contrato **não**
  promete exatamente-uma-vez.
- **Reentrega já em `PROCESSANDO`**: outra entrega da mesma reserva está em curso
  ou ficou órfã. A mensagem é devolvida à fila; se a situação persistir, o limite
  de entregas a leva à fila morta.
- **Falha transitória** (banco ou broker fora): `nack` com reenfileiramento,
  nenhum resultado publicado.
- **Falha definitiva** (JSON inválido, campo obrigatório ausente, forma
  desconhecida, valor não positivo): `nack` sem reenfileiramento, direto para a
  fila morta, com o motivo em cabeçalho.
- **Ausência de resposta do adquirente**: transação em `PENDENTE_VERIFICACAO`,
  `nack` sem reenfileiramento, nenhuma publicação.
- **Prefetch**: igual ao teto de cobranças simultâneas (padrão 10, FR-019).

## 5. Topologia declarada na largada

| Recurso | Tipo | Observação |
|---|---|---|
| `cinema.eventos` | exchange topic durável | compartilhada; declaração idempotente |
| `cinema.eventos.dlx` | exchange topic durável | destino das mensagens descartadas |
| `pagamento.reserva-criada` | fila **quórum** durável | binding `reserva.criada`; `x-delivery-limit: 3` (configurável), `x-dead-letter-exchange: cinema.eventos.dlx` |
| `pagamento.reserva-criada.dlq` | fila durável | quarentena, sem consumidor automático |

A declaração roda na inicialização e é idempotente; o processo recusa subir se a
topologia não puder ser garantida.
