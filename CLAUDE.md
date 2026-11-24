# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Visão geral

Plataforma de venda de ingressos de cinema: quatro microsserviços Go (`catalogo`, `estoque`, `pagamento`, `notificacao`), dois frontends React/Vite (`front` = loja, `admin` = painel) e a infraestrutura em `infra/` (init do Postgres, Prometheus, Grafana). Tudo sobe pelo único `docker-compose.yml`; portas e URLs (Swagger em `/docs`, health, painéis) estão em `urls.txt`. O `README.md` descreve o papel de cada serviço e os fluxos. Docs, specs, comentários e mensagens estão em português.

Cada serviço Go é um **módulo separado** (`go.mod` e `Dockerfile` próprios, sem `go.work`): dê `cd` no serviço antes de rodar `go`/`make`.

```
docker compose up --build        # plataforma inteira (Postgres, Redis, RabbitMQ, Keycloak, Jaeger, Prometheus, Grafana)
```

## Comandos

Dentro de `catalogo/`, `estoque/`, `pagamento/` ou `notificacao/`:

```
make build                 # go build -o bin/<svc> ./cmd/<svc>
make test                  # go test -race ./...
go test -race ./internal/usecase/... -run TestNome   # um teste
make test-integration      # -tags=integration ./test/integration/... (precisa de Docker; timeout longo)
make lint                  # golangci-lint run (.golangci.yml por serviço)
make migrate-up / migrate-down   # exige DATABASE_URL; usa golang-migrate
make proto                 # catalogo e estoque: buf generate (gen/pb é commitado; o Dockerfile não roda buf)
```

`estoque`, `pagamento` e `notificacao` têm alvos `publicar-*` (e `espiar-evento`) que publicam fatos AMQP à mão via `cmd/publicar`, para testar um serviço isolado.

Frontends (`front/`, `admin/`): `npm run dev | build | test` (vitest; `front` também tem `npm run lint` com oxlint). Um teste: `npx vitest run caminho/arquivo.test.tsx`.

## Comunicação entre serviços

- **REST + JWT do Keycloak** (realm `cinema`, importado de `catalogo/keycloak/realm-cinema.json`) entre navegador e APIs. `notificacao` também aceita `X-API-Key` (portaria).
- **gRPC síncrono** só em `estoque → catalogo` (`CatalogoService/ObterSessao`, porta 9091) para precificar/validar a reserva, com timeout e circuit breaker (`gobreaker`) em `estoque/internal/adapter/catalogo`. O `.proto` existe em `catalogo/proto` **e** `estoque/proto` (cópia idêntica): ao mudar um, mude o outro e rode `make proto` nos dois. (O CLAUDE.md do diretório pai ainda diz REST; está desatualizado.)
- **Eventos no RabbitMQ**, exchange `topic` `cinema.eventos`: `sessao.criada/alterada/cancelada` (catalogo), `reserva.criada` (estoque), `pagamento.sucesso/falhou` (pagamento). Nenhum serviço chama `pagamento` ou `notificacao` por HTTP.

Fluxo: `POST /api/v1/sessoes/{id}/bloqueios` (estoque) → trava poltronas e publica `reserva.criada` → `pagamento` cobra e publica o resultado → `estoque` confirma/libera e `notificacao` emite o ingresso. `estoque` também consome `sessao.criada` (provisiona o mapa de poltronas) e expira reservas não pagas em 10 min.

## Arquitetura (hexagonal, igual nos quatro serviços)

`internal/domain` + `internal/usecase` formam o núcleo; `internal/adapter/*` (http, grpc, amqp, postgres, redis) e `internal/platform` (config, obs, health, wiring) são infraestrutura; `cmd/<svc>` é o ponto de entrada. O núcleo não pode importar adaptadores/plataforma/pgx/amqp/redis. Isso é imposto em `estoque/test/arquitetura_test.go` (usa `go list -json`, quebra `make test`) e, no `catalogo`, pela regra `depguard` `nucleo-sem-adaptadores` em `.golangci.yml` (`make lint`). `pagamento` e `notificacao` não têm essa trava, então respeite a regra à mão.

Regras não óbvias:
- **O contrato mora no código**: `internal/adapter/http/openapi/openapi.yaml` é embutido (`go:embed`) e servido em `/docs`; é a fonte de verdade do OpenAPI.
- **Exclusividade no PostgreSQL** (`SELECT ... FOR UPDATE NOWAIT`, ordem determinística de linhas, na mesma transação da reserva + linha da outbox). O Redis guarda só o índice de expiração: perdê-lo atrasa a liberação, mas não causa venda dupla.
- **Outbox transacional** para os eventos publicados; consumidores idempotentes (tabela de mensagens processadas + guardas `WHERE status = 'PENDENTE'`).
- **Um banco (`cinema`), um schema por serviço**. A `schema_migrations` do migrate é fixada no schema do serviço via `search_path`; o SQL das migrações qualifica todo objeto.
- **Observabilidade** (`internal/platform/obs`, praticamente idêntico nos serviços): `slog` JSON com `trace_id`/`span_id`; métricas Prometheus em `OBS_ADDR` (`:9090`, só na rede do compose) com rótulos de baixa cardinalidade (padrão da rota do chi, nunca a URL); OpenTelemetry com o traceparent atravessando HTTP, gRPC (`otelgrpc`) e AMQP (headers). Não há `X-Request-ID`: o `trace_id` é o identificador de correlação.
- Autenticação: `jwt/v5` + JWKS, só `RS256`, valida `iss`/`aud`/`exp`. **Autorização no `catalogo`**: o verificador lê `realm_access.roles` + `scope` do token; rotas de escrita REST exigem o papel `catalogo-admin` (401 sem/ com token ruim, 403 com token válido sem o papel) e o gRPC exige `catalogo-leitura-sessao` (`Unauthenticated`/`PermissionDenied`, interceptor em `adapter/grpc/autenticacao.go`). `estoque`, `pagamento` e `notificacao` ainda só extraem o `sub`.
- **Identidade serviço a serviço**: `estoque` usa o client `estoque-svc` (client credentials, `FonteDeToken` em `estoque/internal/adapter/catalogo/token.go`, token em cache e renovado) como credencial por chamada no gRPC para o catálogo. Papéis e clients vivem em `catalogo/keycloak/realm-cinema.json`; o `--import-realm` só roda com o realm ausente, então mudanças exigem `docker compose down -v` (volume `keycloak-dados`).
- **Segredos vêm do `.env`** (ignorado pelo git; modelo em `.env.example`: `cp .env.example .env` antes do `docker compose up`, que recusa subir com variável obrigatória ausente). Cobre senhas do Postgres (superusuário e um papel por serviço, criados por `infra/postgres/init/01-*.sh`), RabbitMQ, Keycloak admin, `CINEMA_M2M_SECRET` e `UFSC_USER_PASSWORD` (substituídos no `realm-cinema.json` pelo import do Keycloak), `PORTARIA_API_KEY` e `INGRESSO_QR_SEGREDO`. As senhas do Postgres só valem na criação do volume.
- **Rate limiting** (`internal/platform/ratelimit`, balde de fichas em memória por réplica): `estoque` limita `POST /bloqueios` por usuário (`sub`) e `notificacao` limita `POST /ingressos/validar` por IP de origem (sem confiar em `X-Forwarded-For`); excesso devolve 429 com `Retry-After`. Ajuste por `RATE_LIMIT_RPS`/`RATE_LIMIT_BURST`.
