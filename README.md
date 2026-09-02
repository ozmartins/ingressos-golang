# Ingressos

Plataforma de venda de ingressos de cinema em quatro microsserviços Go, com
PostgreSQL, Redis, RabbitMQ e Keycloak, orquestrados por um único
[`docker-compose.yml`](docker-compose.yml) na raiz.

## Serviços

| Serviço | Papel | Superfícies |
|---|---|---|
| [`catalogo/`](catalogo/README.md) | Navegação por filmes, cinemas, salas e sessões; inicia a reserva | REST (JWT), cliente gRPC do estoque |
| [`estoque/`](estoque/README.md) | Dono da disponibilidade das poltronas; bloqueia e confirma assentos | gRPC (mTLS), REST (JWT), consumidor/produtor AMQP |
| [`pagamento/`](pagamento/README.md) | Cobra de forma assíncrona a reserva criada | REST (JWT), consumidor/produtor AMQP |
| [`notificacao/`](notificacao/README.md) | Emite o ingresso digital e valida na portaria | REST (JWT e `X-API-Key`), consumidor AMQP |

O fluxo assíncrono é `reserva.criada` → `pagamento.sucesso` / `pagamento.falhou`
→ emissão do ingresso.

## Subir tudo

```bash
docker compose up --build
```

As portas do host e os endereços de cada serviço, incluindo os `/docs` (Swagger
UI), os health checks, o painel do RabbitMQ e o console do Keycloak, estão em
[`URLS.txt`](URLS.txt). Todas as portas são configuráveis pelas variáveis
`PORTA_*` do compose.
