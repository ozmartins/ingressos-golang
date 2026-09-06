# Contrato de eventos — Servico-Catalogo

Este serviço publica um fato e não consome nenhum.

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

## Não publicado, e por quê

Alterar (`PUT`) ou cancelar (`DELETE`) uma sessão **não** emite fato. Não é
esquecimento: não existe consumidor para eles, e nenhum contrato de evento os
descreve do outro lado. Enquanto for assim, mudar a sala de uma sessão já
anunciada deixa a matriz do estoque apontando para a planta antiga — a alteração
precisa ser tratada como sessão nova até que um fato de alteração seja
especificado e consumido.
