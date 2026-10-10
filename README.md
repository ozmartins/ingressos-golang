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

## Persistência: um banco, um schema por serviço

Os quatro serviços usam a mesma instância PostgreSQL e o mesmo banco (`cinema`),
mas **nenhum compartilha tabelas com outro**. Foi uma escolha para manter a
infraestrutura local simples (um único contêiner de banco); o isolamento é
feito por schema e por papel:

| Serviço | Schema | Papel (login) |
|---|---|---|
| `catalogo` | `catalogo` | `catalogo` |
| `estoque` | `estoque` | `estoque` |
| `pagamento` | `pagamento` | `pagamento` |
| `notificacao` | `notificacao` | `notificacao` |

- [`infra/postgres/init/01-papeis-e-schemas.sql`](infra/postgres/init/01-papeis-e-schemas.sql)
  cria cada schema com `AUTHORIZATION` do próprio papel, sem nenhum `GRANT`
  cruzado, e revoga `CREATE` no schema `public`. Um serviço não tem permissão
  para ler nem escrever no schema de outro.
- Cada serviço fixa o `search_path` no seu schema, tanto em runtime
  (`internal/adapter/postgres`) quanto nas migrations (a tabela
  `schema_migrations` também fica no schema do serviço).
- Nenhuma query ou migration referencia o schema de outro serviço. A troca de
  dados entre serviços acontece só por gRPC/REST e por mensagens AMQP.
- Separar em bancos físicos distintos exigiria apenas trocar a `DATABASE_URL` de
  cada serviço no `docker-compose.yml`; o código não muda.

## Subir tudo

```bash
docker compose up --build
```

As portas do host e os endereços de cada serviço, incluindo os `/docs` (Swagger
UI), os health checks, o painel do RabbitMQ e o console do Keycloak, estão em
[`urls.txt`](urls.txt). Todas as portas são configuráveis pelas variáveis
`PORTA_*` do compose.
