# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Cinema ticketing platform made of four Go microservices (`catalogo`, `estoque`, `pagamento`, `notificacao`) plus a Vite + React 19 + Tailwind 4 frontend (`web/`, served by nginx in Docker). Each service is a **separate Go module** (own `go.mod`, `Dockerfile`, `Makefile`; no `go.work`), so `cd` into the service before running Go tooling. Docs, specs and comments are in Portuguese.

All services share one PostgreSQL database (`cinema`) with one schema per service, plus Redis, RabbitMQ and Keycloak (realm `cinema`, imported from `catalogo/keycloak`). A single root `docker-compose.yml` runs everything, including one-shot `migrate-<svc>` containers:

```
docker compose up --build
```

Host ports and URLs (Swagger at `/docs`, health checks, RabbitMQ panel, Keycloak console) are listed in `urls.txt`; ports are overridable via `PORTA_*` variables.

Frontend (`web/`, not Go) — run inside `web/`: `npm run dev`, `npm run build` (`tsc -b && vite build`), `npm run typecheck`. No test or lint scripts. Infra seed SQL (roles/schemas) lives in `infra/postgres/`.

## Async flow

`catalogo` starts a reservation (gRPC call to `estoque`) → `estoque` locks seats and publishes `reserva.criada` → `pagamento` charges and publishes `pagamento.sucesso` / `pagamento.falhou` → `estoque` confirms or releases the seats, and `notificacao` issues the digital ticket. `estoque` also consumes `sessao.criada` (provisions the seat matrix) and expires unpaid reservations after 10 minutes.

## Commands

Run inside a service directory (each has a `Makefile`):

```
make build              # go build -o bin/<svc> ./cmd/<svc>
make test               # go test -race ./...
go test -race ./internal/usecase/... -run TestName    # single test
make test-integration   # -tags=integration ./test/integration/... (needs Docker infra)
make lint               # golangci-lint run (.golangci.yml per service)
make migrate-up         # / migrate-down; needs DATABASE_URL
make proto              # buf generate (catalogo, estoque)
make openapi-sync       # copy contract into internal/adapter/http/openapi/
```

`estoque` also has `make certs` (mTLS for gRPC), `contrato-sync`, and `publicar-sessao` / `publicar-pagamento` (publish AMQP facts by hand for the quickstart).

## Architecture (hexagonal, same in every service)

`internal/domain` and `internal/usecase` are the core; `internal/adapter/*` (http, grpc, amqp, postgres, redis) and `internal/platform` (config/wiring) are infrastructure; `cmd/<svc>` is the entrypoint. The core must not import adapters, platform, generated protobuf, pgx, amqp or redis. `estoque` enforces this in `test/arquitetura_test.go` (via `go list -json`, so a violation fails `make test`); `catalogo` enforces it with the `depguard` rule `nucleo-sem-adaptadores` in `.golangci.yml` (`make lint`); `pagamento` and `notificacao` have neither.

Non-obvious rules:
- **Contracts are copied, not shared.** The versioned `specs/<feature>/contracts/{openapi.yaml,*.proto}` is the source of truth; `go:embed` can't reach it, so `make openapi-sync` / `make proto` copy it into `internal/...` or `proto/`. A parity test fails if copies diverge — edit the spec file and sync, never the copy.
- **Exclusivity lives in PostgreSQL** (`SELECT ... FOR UPDATE NOWAIT`, deterministic row order, same transaction as reservation + outbox row). Redis holds only the expiry index: losing it delays release but cannot cause double-selling.
- **Transactional outbox** for `reserva.criada`; consumers are idempotent (processed-message table plus `WHERE status = 'PENDENTE'` state guards).
- **`schema_migrations` is pinned to each service's schema** via `search_path` (shared DB); migration SQL qualifies every object itself.
- Auth: REST uses Keycloak JWT (`notificacao` also accepts `X-API-Key`); `estoque` gRPC (:50051) requires mTLS, with a plaintext simulated estoque on :50052.

## Governance (Spec-Kit)

Features are built with Spec-Kit; each service keeps `specs/NNN-.../` (spec, research, data-model, plan, tasks, contracts) and its own constitution adding technical rules. The workspace constitution is `.specify/memory/constitution.md` (v1.0.0), with four principles: (I) no complexity unless needed or requested; (II) domain and exposed APIs must have automated tests; (III) **the code is the source of truth, not the spec** — verify behavior claims in code; (IV) code/spec divergence is a question for the maintainer, not something to fix unilaterally.
