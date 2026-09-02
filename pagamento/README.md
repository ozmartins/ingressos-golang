# Servico-Pagamento

Processa de forma assíncrona as cobranças do sistema de cinema. Consome o fato
`reserva.criada`, cobra por trás de uma porta de adquirente e anuncia o desfecho
em `pagamento.sucesso` ou `pagamento.falhou`. Expõe uma única operação síncrona:
a consulta do andamento pelo identificador da reserva.

Especificação, plano e decisões: [`specs/001-pagamento-assincrono/`](specs/001-pagamento-assincrono/).

## ⚠ Dependência de integração aberta

O `Servico-Estoque` publica `reserva.criada` **sem** `valor_total` e sem
`forma_pagamento` — ver `estoque/internal/usecase/bloquear_poltronas.go`
(`EventoReservaCriada`) e `estoque/proto/estoque.proto` (`SolicitacaoBloqueio`,
que sequer recebe esses dados do catálogo).

Este serviço exige os dois campos (FR-003). **Enquanto o estoque não os propagar,
todo evento real vindo dele é inválido aqui e vai para a fila morta.** A validação
ponta a ponta usa o publicador manual `cmd/publicar`.

A divergência foi decidida com o mantenedor em 2026-08-30 e está registrada em
`specs/001-pagamento-assincrono/research.md` (D1) e na caixa de aviso de
`contracts/eventos.md` §1. Fechá-la exige, no estoque: os dois campos em
`SolicitacaoBloqueio`, na tabela `reservas` e em `reserva.criada` v2.

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
   (`PENDENTE_VERIFICACAO`), não uma recusa. É terminal, nunca anunciado, e a
   mensagem vai para a quarentena. É o único silêncio deliberado do serviço.
4. **O direito de cobrar é reivindicado atomicamente** no banco antes de falar com
   o adquirente, e devolvido se ele responder com erro. É o que permite a uma
   falha transitória se completar (FR-020) sem jamais recobrar (FR-008).

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
