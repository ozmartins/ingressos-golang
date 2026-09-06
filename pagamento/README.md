# Servico-Pagamento

Processa de forma assíncrona as cobranças do sistema de cinema. Consome o fato
`reserva.criada`, cobra por trás de uma porta de adquirente e anuncia o desfecho
em `pagamento.sucesso` ou `pagamento.falhou`. Expõe duas operações síncronas: a
escolha da forma de pagamento de uma reserva e a consulta do andamento dela.

Especificação, plano e decisões: [`specs/001-pagamento-assincrono/`](specs/001-pagamento-assincrono/).

## Reserva e cobrança são separadas

Reservar uma poltrona e escolher como pagar são decisões distintas, e o fluxo as
trata assim:

1. o `Servico-Estoque` publica `reserva.criada` com `valor_total` — o preço vem
   do catálogo, dono do cadastro da sessão;
2. o consumo do fato cria a transação em **`AGUARDANDO_FORMA`**: ela já sabe
   quanto cobrar, e ainda não como;
3. quem paga escolhe a forma em `POST /api/v1/pagamentos/reserva/{reserva_id}`,
   que responde **`202`** — a cobrança acontece fora da requisição, então ninguém
   fica esperando o adquirente;
4. uma varredura periódica cobra as escolhidas, desiste das que venceram sem
   escolha, e republica anúncio que não saiu.

Até 2026-09-06 este serviço exigia `forma_pagamento` dentro do fato, e o estoque
nunca a enviou — porque ninguém no sistema a pedia. O resultado é que **todo**
anúncio vindo dele era inválido e ia para a fila morta. O histórico da decisão
está na caixa de `contracts/eventos.md` §1.

## Como subir

```bash
docker compose -f ../docker-compose.yml up -d rabbitmq
export DATABASE_URL='postgres://pagamento:pagamento@localhost:5434/cinema?sslmode=disable'
make migrate-up
make run
```

A `DATABASE_URL` precisa da query string: `make migrate-up` anexa
`&search_path=pagamento` a ela.

As tabelas do pagamento vivem no schema `pagamento`, não em `public`: a migração
cria o schema e qualifica cada objeto, e o serviço fixa o `search_path` no pool
de conexões. Para inspecionar o banco com `psql`, aponte o `search_path` antes
(`SET search_path TO pagamento;`) ou qualifique a tabela. A tabela de
controle do golang-migrate (`schema_migrations`) mora no mesmo schema: em
`public` os quatro serviços disputariam uma só.

## Documentação da API

O contrato é servido pelo próprio serviço: `/openapi.yaml` devolve o documento e
`/docs` abre o Swagger UI sobre ele — http://localhost:8083/docs no compose. Nenhum dos dois exige credencial:
pedir token para ler o contrato barraria justamente quem ainda vai integrar.

O contrato versionado em `specs/001-pagamento-assincrono/contracts/openapi.yaml` é a fonte da verdade; a cópia embutida no
binário é gerada por `make openapi-sync`, e um teste de paridade falha quando as
duas divergem.

Roteiro completo de validação: [`specs/001-pagamento-assincrono/quickstart.md`](specs/001-pagamento-assincrono/quickstart.md).

## Configuração

Tudo vem do ambiente e é validado uma vez na largada; variável ausente ou
malformada impede o processo de subir.

| Variável | Obrigatória | Padrão | O que é |
|---|---|---|---|
| `DATABASE_URL` | sim | — | conexão PostgreSQL (o `search_path` é fixado no código, no schema `pagamento`) |
| `AMQP_URL` | sim | — | conexão RabbitMQ |
| `JWKS_URL` | sim | — | conjunto de chaves do Keycloak |
| `JWT_ISSUER` | sim | — | emissor aceito |
| `JWT_AUDIENCE` | sim | — | público aceito |
| `PORTA_HTTP` | não | `8080` | porta da API |
| `AMQP_EXCHANGE` | não | `cinema.eventos` | exchange do barramento |
| `AMQP_FILA_RESERVA_CRIADA` | não | `pagamento.reserva-criada` | fila consumida |
| `AMQP_PREFETCH` | não | `10` | teto de cobranças simultâneas (FR-019) |
| `AMQP_LIMITE_ENTREGAS` | não | `3` | tentativas antes da quarentena (FR-021) |
| `ADQUIRENTE_TIMEOUT` | não | `10s` | prazo de resposta do adquirente (FR-022) |
| `VARREDURA_INTERVALO` | não | `2s` | intervalo da varredura; é o atraso entre a escolha da forma e o início da cobrança |
| `VARREDURA_LOTE` | não | `50` | transações examinadas por varredura |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | não | — | destino de métricas e rastros |
| `NIVEL_LOG` | não | `info` | `debug`, `info`, `warn` ou `error` |

## Desenho

Arquitetura hexagonal. `internal/domain` e `internal/usecase` não importam
adaptador; a composição acontece só em `cmd/pagamento/main.go`.

Quatro decisões moldam o resto (detalhes em `research.md`):

1. **A cobrança única mora na restrição `UNIQUE (reserva_id)`**, exercida por
   `INSERT ... ON CONFLICT DO NOTHING RETURNING`. Não há tabela de mensagens
   processadas: a linha da transação é o registro de "já processei".
2. **O anúncio é garantido por uma coluna booleana**, não por caixa de saída.
   Ordem invariável: gravar estado final → publicar → marcar → confirmar a
   mensagem. A entrega é ao menos uma vez; consumidores deduplicam por `reserva_id`.
3. **Ausência de resposta do adquirente é um estado do domínio**
   (`PENDENTE_VERIFICACAO`), não uma recusa. É terminal e nunca anunciado — o
   único silêncio deliberado do serviço. Desde que a cobrança saiu do consumo, ele
   não descarta mais a mensagem: o anúncio da reserva foi processado com sucesso,
   e o que ficou indeterminado é a cobrança, sinalizada pelo estado da transação.
4. **O direito de cobrar é reivindicado atomicamente** no banco antes de falar com
   o adquirente, e devolvido se ele responder com erro. É o que permite a uma
   falha transitória se completar (FR-020) sem jamais recobrar (FR-008).
5. **A cobrança mora numa varredura, não no consumo do fato.** Quem paga escolheu
   a forma e não precisa esperar o adquirente; e o que retoma trabalho perdido —
   cobrança interrompida, anúncio que não saiu, espera vencida — passa a ser um
   lugar só, que roda pela passagem do tempo e não pela chegada de mensagem.

## Testes

```bash
make test              # domínio, casos de uso e contrato HTTP — sem Docker
make test-integration  # Testcontainers: PostgreSQL e RabbitMQ reais
make lint
```

A suíte de integração é onde as garantias que envolvem dinheiro são provadas:
cobrança única sob vinte entregas simultâneas, republicação após queda entre
gravar e publicar, quarentena do estado indeterminado, e rajada de mil intenções
respeitando o teto de concorrência.
