# Quickstart — Servico-Pagamento

**Fase 1** do `/speckit-plan` · Roteiro de validação ponta a ponta.
Prova, com o serviço rodando de verdade, os desfechos que a spec exige.

## Pré-requisitos

- Go 1.25+, Docker e Docker Compose, `migrate` (golang-migrate) e `jq`
- Portas livres: 8080 (HTTP), 5432 (Postgres), 5672/15672 (RabbitMQ).
  Se a máquina já tiver PostgreSQL local, exporte `PORTA_POSTGRES=55432` (e
  ajuste a `DATABASE_URL` abaixo) antes de subir o compose.

## 1. Subir infraestrutura e o serviço

```bash
cd pagamento
docker compose up -d postgres rabbitmq       # espera os health checks
export DATABASE_URL='postgres://pagamento:pagamento@localhost:5432/pagamento?sslmode=disable'
make migrate-up
make run                                     # declara a topologia e começa a consumir
```

Pronto quando `GET /api/v1/health/ready` devolver 200:

```bash
curl -s -o /dev/null -w '%{http_code}\n' localhost:8080/api/v1/health/ready
```

> **Por que o evento vem de um publicador manual.** O `Servico-Estoque` ainda não
> inclui `valor_total` e `forma_pagamento` no `reserva.criada` — ver a caixa de
> dependência de integração em [contracts/eventos.md](./contracts/eventos.md) §1.
> Até que ele passe a incluí-los, todo evento vindo dele é inválido para este
> serviço e vai para a fila morta. `cmd/publicar` gera o evento no formato que
> este serviço exige, e é com ele que o roteiro abaixo funciona.

## 2. Caminho feliz — cobrança aprovada e anunciada

```bash
RESERVA=$(uuidgen)
make publicar-reserva RESERVA=$RESERVA VALOR=84.00 FORMA=PIX
```

**Esperado**: em segundos, a transação chega a `PAGO` e `pagamento.sucesso` é
publicado com `transacao_id`, `reserva_id`, `usuario_id`, `valor_total` e `pago_em`.

```bash
make espiar-evento ROUTING=pagamento.sucesso      # lê da fila de inspeção
psql "$DATABASE_URL" -c \
  "select status, resultado_anunciado, pago_em from transacoes_pagamento where reserva_id='$RESERVA'"
# → PAGO | t | <instante>
```

## 3. Consulta do andamento e as três recusas de acesso

```bash
TOKEN=$(./scripts/token-teste.sh "$USUARIO")          # assinado pela chave de teste
curl -s -H "Authorization: Bearer $TOKEN" \
  localhost:8080/api/v1/pagamentos/reserva/$RESERVA | jq
```

| Verificação | Comando | Esperado |
|---|---|---|
| Dona da reserva | token com `sub` = `usuario_id` | 200 com `status: PAGO` |
| Terceiro | token com outro `sub` | **404** `PAGAMENTO_NAO_ENCONTRADO` — idêntico ao de reserva inexistente (FR-017) |
| Reserva sem transação | UUID aleatório | 404, mesmo corpo do caso acima |
| Sem token | sem cabeçalho | 401 `CREDENCIAL_INVALIDA` |
| UUID malformado | `.../reserva/abc` | 400 `RESERVA_ID_INVALIDO` |

Compare os corpos dos dois 404 byte a byte: precisam ser iguais.

## 4. Cobrança única sob entrega repetida (SC-002)

```bash
RESERVA=$(uuidgen)
for i in $(seq 1 20); do make publicar-reserva RESERVA=$RESERVA VALOR=50.00 FORMA=PIX & done; wait
psql "$DATABASE_URL" -c "select count(*) from transacoes_pagamento where reserva_id='$RESERVA'"
# → 1
make espiar-evento ROUTING=pagamento.sucesso | grep -c "$RESERVA"
# → 1  (um único anúncio; ver §4 de contracts/eventos.md)
```

## 5. Recusa do adquirente

O adquirente simulado decide pelos centavos do valor (research.md D7):
`.13` recusa por cartão, `.66` por saldo insuficiente, `.99` demora além do prazo,
qualquer outro aprova.

```bash
make publicar-reserva RESERVA=$(uuidgen) VALOR=13.13 FORMA=CARTAO_CREDITO
```

**Esperado**: `RECUSADO` com `motivo_falha` preenchido e `pagamento.falhou`
publicado com o mesmo motivo.

## 6. Reserva já expirada (FR-005)

```bash
make publicar-reserva RESERVA=$(uuidgen) VALOR=84.00 FORMA=PIX EXPIRA_EM=-5m
```

**Esperado**: **nenhuma** chamada ao adquirente; transação `CANCELADO`;
`pagamento.falhou` com `motivo: RESERVA_EXPIRADA`.

## 7. Adquirente sem resposta — o único silêncio deliberado (FR-022)

```bash
make publicar-reserva RESERVA=$(uuidgen) VALOR=99.99 FORMA=CARTAO_CREDITO   # centavos .99 = demora
```

**Esperado**, e é o desfecho mais importante de conferir:

- transação em `PENDENTE_VERIFICACAO`, `resultado_anunciado = false`
- **nenhuma** mensagem em `pagamento.sucesso` nem em `pagamento.falhou`
- a mensagem original em `pagamento.reserva-criada.dlq`, esperando inspeção humana

```bash
curl -su guest:guest localhost:15672/api/queues/%2f/pagamento.reserva-criada.dlq | jq .messages
# → 1
```

## 8. Anúncio garantido após queda entre gravar e publicar (SC-003)

O caso que a coluna `resultado_anunciado` existe para cobrir: o serviço grava o
estado final e morre antes de publicar. Reproduzir isso à mão exigiria injetar
uma falha no meio do fluxo, então ele é verificado por teste automatizado, que
monta exatamente esse estado e observa a reentrega:

```bash
go test -tags=integration -v -run TestReentregaPublicaResultadoPendenteSemRecobrar ./test/integration/
```

**Esperado**: `pagamento.sucesso` publicado a partir da transação já gravada,
`resultado_anunciado` passa a verdadeiro, e **nenhuma segunda cobrança** chega ao
adquirente.

Para ver o estado intermediário à mão, dá para forjá-lo no banco:

```bash
psql "$DATABASE_URL" -c \
  "update transacoes_pagamento set resultado_anunciado=false where reserva_id='$RESERVA'"
make publicar-reserva RESERVA=$RESERVA VALOR=84.00 FORMA=PIX
```

## 9. Vazão sob rajada (SC-004)

```bash
make publicar-rajada N=1000
```

**Esperado**: as 1.000 processadas sem perda; o medidor de cobranças em andamento
nunca passa do teto configurado (padrão 10); as consultas continuam respondendo
durante o pico.

## 10. Suíte automatizada

```bash
make test                 # domínio, casos de uso e contrato HTTP — sem Docker
make test-integration     # Testcontainers: Postgres e RabbitMQ reais
make lint
```

`make test-integration` é onde os cenários 4, 7, 8 e 9 acima viram teste
automatizado — o roteiro manual serve para ver acontecer, não substitui a suíte.

## Limites conhecidos deste roteiro

- O passo 2 usa `cmd/publicar`, não o estoque real (ver caixa na §1). Ponta a ponta
  com o estoque só depois que ele publicar os dois campos.
- O adquirente é simulado nesta entrega, por decisão registrada em research.md D7.
- Resolver uma transação `PENDENTE_VERIFICACAO` é trabalho manual a partir da fila
  morta; automatizar essa reconciliação está fora do escopo (spec, Assumptions).
