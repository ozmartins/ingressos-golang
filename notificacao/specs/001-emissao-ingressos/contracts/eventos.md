# Contrato de Eventos: Servico-Notificacao

**Fase 1** | **Data**: 2026-08-30

Este serviço **consome um fato e não publica nenhum**. É a ponta da cadeia: nada reage
a ele por evento (research D10).

---

## 1. Fato consumido: `pagamento.sucesso`

> **Verificado no código do produtor, não na spec dele** (princípio III).
> Fonte: `pagamento/internal/usecase/fatos.go`, tipo `FatoPagamentoSucesso` e constante
> `RoutingKeySucesso`. O exchange vem de `pagamento/internal/platform/config/config.go`
> (`AMQP_EXCHANGE`, padrão `cinema.eventos`).

**Exchange**: `cinema.eventos` (tipo `topic`, durável)
**Chave de roteamento**: `pagamento.sucesso`
**Fila consumida**: `notificacao.pagamento-sucesso`

### Payload publicado hoje

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

### O que este serviço exige, ignora e descarta

| Campo | Tratamento |
|---|---|
| `reserva_id` | **obrigatório**, UUID. Chave da idempotência (FR-004) |
| `usuario_id` | **obrigatório**, UUID. Dono do ingresso |
| `transacao_id` | **obrigatório**, UUID. Só para correlação em log e rastro |
| `pago_em` | **obrigatório**, RFC 3339. Validado como instante bem formado |
| `valor_total` | ignorado — nenhum requisito o usa, nenhuma coluna o guarda |
| `evento` | ignorado — a chave de roteamento já disse o que é |
| `versao`, `ocorrido_em` | ignorados; chegam a mais e são tolerados (D1) |
| qualquer campo novo | tolerado, sem erro |

Ausência ou má formação de qualquer um dos quatro obrigatórios → **quarentena
imediata, sem retentativa** (FR-002).

> **Nota sobre a ERS**: a ERS deste serviço descreve seis campos — os oito acima menos
> `versao` e `ocorrido_em`. O que o código publica é um **superconjunto** do que a ERS
> espera, então não há divergência entre spec e código a levar ao princípio IV. Os
> quatro campos que este serviço realmente usa estão descritos na ERS e são publicados
> pelo código.

---

## 2. Topologia RabbitMQ declarada por este serviço

Declarada na largada, de forma idempotente. O processo **recusa subir** se a topologia
não puder ser garantida: consumir de fila sem fila morta é perder mensagem em silêncio.

| Recurso | Nome | Configuração |
|---|---|---|
| Exchange | `cinema.eventos` | `topic`, durável (compartilhado; já declarado por outros serviços) |
| Exchange de mortas | `cinema.eventos.dlx` | `topic`, durável |
| Fila | `notificacao.pagamento-sucesso` | `quorum`, durável |
| Fila morta | `notificacao.pagamento-sucesso.dlq` | durável |
| Vínculo | `cinema.eventos` → fila | chave `pagamento.sucesso` |
| Vínculo | `cinema.eventos.dlx` → fila morta | chave `notificacao.pagamento-sucesso` |

Argumentos da fila principal:

```
x-queue-type              = quorum
x-delivery-limit          = AMQP_LIMITE_ENTREGAS - 1
x-dead-letter-exchange    = cinema.eventos.dlx
x-dead-letter-routing-key = notificacao.pagamento-sucesso
```

**O `- 1` é tradução de vocabulário, não engano** (D5): a FR-022 conta **tentativas**;
o RabbitMQ conta **reentregas** e entrega `limite + 1` vezes. Com
`AMQP_LIMITE_ENTREGAS=3`, a mensagem é processada três vezes e vai para a fila morta.
O achado é do `Servico-Pagamento`, medido contra o broker e registrado em
`pagamento/internal/adapter/amqp/topologia.go`; aqui é reaproveitado, e reverificado
pelo teste de integração de quarentena.

---

## 3. Desfechos do consumo

O consumo é interface exposta — por evento, não por HTTP — e por isso o princípio II
exige teste de sucesso e de **cada** categoria abaixo.

| Desfecho | Quando | Gesto AMQP | Ingresso | Aviso |
|---|---|---|---|---|
| **Emitido** | anúncio válido, reserva ainda sem ingresso | `Ack` | criado, `VALIDO` | disparado e registrado |
| **Emitido, aviso falhou** | idem, canal de aviso com erro | `Ack` | criado, `VALIDO` | registrado como `FALHA` (FR-018, FR-025) |
| **Inerte (idempotente)** | reserva já tem ingresso | `Ack` | inalterado | **não** disparado (D6) |
| **Quarentena** | JSON ilegível, campo obrigatório ausente ou malformado | `Nack(requeue=false)` | nenhum | nenhum |
| **Nova tentativa** | falha transitória (banco fora, timeout) | `Nack(requeue=true)` | nenhum | nenhum |
| **Quarentena por esgotamento** | limite de entregas atingido | encaminhado pelo broker | nenhum | nenhum |

Um ponto que o teste precisa fixar (FR-025): **aviso com falha continua sendo `Ack`**.
Se um dia virar `Nack`, a mensagem passa a ser reprocessada por causa de um erro de
notificação, e a garantia da FR-018 morre em silêncio.

---

## 4. Publicação

Nenhuma. Este serviço não publica fato algum.

Publicar um `ingresso.emitido` foi considerado e rejeitado: não há consumidor, e
antecipar requisito futuro não é necessidade demonstrada (princípio I, research D10).
