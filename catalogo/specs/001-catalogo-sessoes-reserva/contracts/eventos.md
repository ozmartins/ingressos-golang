# Contrato de eventos — Servico-Catalogo

Este serviço publica três fatos, o ciclo de vida de uma sessão, e não consome
nenhum.

O intermediário é o RabbitMQ, no exchange compartilhado `cinema.eventos`, do tipo
`topic` e durável — o mesmo que os quatro serviços usam. O catálogo declara o
exchange na largada, e nada além dele: fila é de quem consome, e criar a fila do
vizinho é decidir por ele quando ela existe e como ela é.

A entrega é **ao menos uma vez** (princípio VI). Nenhum consumidor pode supor que
um fato chega uma vez só, nem que dois fatos chegam na ordem em que foram
produzidos.

---

## Publicado — `sessao.criada`

| | |
|---|---|
| Routing key | `sessao.criada` |
| Exchange | `cinema.eventos` |
| Chave de idempotência | `sessao_id`, repetida no `message_id` da mensagem |
| Produzido quando | uma sessão é criada com sucesso por `POST /api/v1/sessoes` |
| Versão | 1 |

### Corpo

```json
{
  "evento": "SESSAO_CRIADA",
  "versao": 1,
  "ocorrido_em": "2026-09-06T18:00:00Z",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456",
  "sala_id": "d1b2c3d4-0000-4000-8000-000000000002",
  "poltronas": [
    { "fileira": "A", "numero": 1, "tipo": "NORMAL" },
    { "fileira": "A", "numero": 2, "tipo": "NORMAL" },
    { "fileira": "F", "numero": 7, "tipo": "PCD" }
  ]
}
```

| Campo | Regra |
|---|---|
| `evento` | Sempre `SESSAO_CRIADA` |
| `versao` | Inteiro. `1` nesta versão do contrato |
| `ocorrido_em` | RFC 3339 em UTC — o instante da criação, não o da publicação |
| `sessao_id` | UUID v4 da sessão criada |
| `sala_id` | UUID v4 da sala em que ela ocorre |
| `poltronas` | A planta da sala expandida assento a assento. Nunca vazia |

Cada poltrona traz `fileira` (de 1 a 5 letras de A a Z, maiúsculas), `numero`
(inteiro ≥ 1, reiniciando em 1 a cada fileira) e `tipo`
(`NORMAL` \| `PCD` \| `NAMORADEIRA`). O par `(fileira, numero)` é único no fato.
A lista vem ordenada por fileira e, dentro dela, por número.

A planta é a da sala **no instante da criação da sessão**. Redesenhar a sala
depois não reemite o fato nem altera o que já foi anunciado: a matriz de uma
sessão é a que valia quando ela foi marcada.

### Cabeçalhos

O contexto de rastreamento W3C capturado na requisição que criou a sessão viaja
nos headers da mensagem (`traceparent`, e `tracestate` quando houver). O fato é
publicado fora do caminho da requisição, e sem isso o span do consumidor nasceria
órfão.

`message_id` carrega o `sessao_id`. `content_type` é `application/json` e o modo
de entrega é persistente.

### Entrega

O fato é gravado numa caixa de saída (`catalogo.outbox_eventos`) **na mesma
transação** que insere a sessão: ou as duas coisas acontecem, ou nenhuma. Um
processo à parte drena a caixa e republica até o broker confirmar, o que significa
que a mesma mensagem pode chegar mais de uma vez. A resposta do `POST` não espera
pela publicação — uma sessão é criada com sucesso mesmo com o broker fora do ar,
e o fato sai quando ele voltar.

O consumidor deve descartar a repetição pelo `sessao_id`.

---

## Publicado — `sessao.alterada`

| | |
|---|---|
| Routing key | `sessao.alterada` |
| Exchange | `cinema.eventos` |
| Chave de idempotência | o `message_id` da mensagem, próprio de cada alteração |
| Produzido quando | uma sessão é alterada com sucesso por `PUT /api/v1/sessoes/{id}` |
| Versão | 1 |

```json
{
  "evento": "SESSAO_ALTERADA",
  "versao": 1,
  "ocorrido_em": "2026-09-06T18:10:00Z",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456",
  "sala_id": "d1b2c3d4-0000-4000-8000-000000000002",
  "data_hora_inicio": "2026-12-01T22:00:00Z",
  "idioma": "DUBLADO",
  "preco_base": "38.00"
}
```

O corpo traz o estado final da sessão, como o `PUT` a redesenhou.

**Este fato nunca invalida a matriz de poltronas**, e é de propósito: a sala não
pode mudar. `PUT` com outra sala responde `409` — a sala é do cadastro da sessão,
não do estado que a substituição redesenha, pela mesma razão que o `cinema_id` de
uma sala não muda. Quem precisa de outra sala cancela a sessão e cria outra.
`sala_id` viaja aqui de todo modo, para que quem consome não precise guardar de
qual sala era só para concluir que não mudou.

Diferente dos outros dois, o `message_id` **não** é derivado do `sessao_id`: a
mesma sessão pode ser alterada muitas vezes, e uma chave estável faria a segunda
alteração colidir com a primeira na caixa de saída e ser descartada em silêncio.
Cada alteração tem identificador próprio, e a deduplicação é por ele.

---

## Publicado — `sessao.cancelada`

| | |
|---|---|
| Routing key | `sessao.cancelada` |
| Exchange | `cinema.eventos` |
| Chave de idempotência | `sessao_id`, repetida no `message_id` como `<sessao_id>:cancelada` |
| Produzido quando | uma sessão é retirada da grade por `DELETE /api/v1/sessoes/{id}` |
| Versão | 1 |

```json
{
  "evento": "SESSAO_CANCELADA",
  "versao": 1,
  "ocorrido_em": "2026-09-06T18:20:00Z",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456"
}
```

A remoção é lógica: a sessão passa a `CANCELADA` e sai da grade, mas a linha
permanece. O fato existe porque quem tem estado preso à sessão precisa soltá-lo —
no `Servico-Estoque`, as reservas pendentes das poltronas dela, cujo prazo
continuaria correndo para uma sessão que ninguém mais pode comprar.

O cancelamento é terminal e acontece uma vez, então a chave é estável e dá
idempotência de graça.

---

## Por que a planta da sala não precisa de fato

`sessao.criada` carrega a planta do instante em que a sessão foi criada, e nada
depois disso a altera — porque a planta de uma sala não pode ser redesenhada.
`PUT /api/v1/salas/{id}` com outras fileiras responde `409`, pela mesma razão que
uma sessão não muda de sala: quem já reservou perderia o assento, e uma poltrona
vendida numa fileira removida não teria para onde ir.

Assim a matriz provisionada e a planta cadastrada não têm como divergir, e não há
fato de redesenho a publicar. Quem precisa de outra planta desativa a sala e
cadastra outra; as sessões da sala antiga seguem válidas, com a matriz que
sempre tiveram.
