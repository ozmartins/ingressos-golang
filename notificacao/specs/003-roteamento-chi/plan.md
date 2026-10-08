# Implementation Plan: Roteamento HTTP da Notificação via chi

**Branch**: _não aplicável — commit direto na `master` (regra do mantenedor)_ | **Date**: 2026-10-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-roteamento-chi/spec.md`

## Summary

Trocar o `http.ServeMux` usado em `API.Rotas()` (`internal/adapter/http/handlers.go`) pelo
roteador `github.com/go-chi/chi/v5`, preservando 100% do contrato HTTP: mesmas rotas, status,
corpos, autenticação (feita dentro dos tratadores, não em middleware) e comportamento de
404/405/HEAD. A mudança é de uma função mais `go.mod`; handlers, casos de uso, persistência e
mensageria não mudam. As diferenças sutis entre os dois roteadores (HEAD, `Allow` no 405, o
padrão `/docs/` como subárvore) estão em [research.md](research.md) e são cobertas por testes.

## Technical Context

**Language/Version**: Go 1.25 (`go.mod`)

**Primary Dependencies**: `github.com/go-chi/chi/v5` v5.3.2 (nova; já presente no cache de módulos); `net/http` permanece para tipos/servidor

**Storage**: N/A (inalterado)

**Testing**: `go test -race ./...`; testes HTTP existentes em `internal/adapter/http/*_test.go` via `httptest.NewServer(api.Rotas())`

**Target Platform**: Linux server (container)

**Project Type**: web-service (microsserviço hexagonal)

**Performance Goals**: sem regressão perceptível; roteamento não é gargalo

**Constraints**: contrato OpenAPI inalterado; `Rotas() http.Handler` mantém a assinatura; mudança restrita ao adaptador HTTP

**Scale/Scope**: 6 rotas (+ `/docs/` subárvore); 1 arquivo de produção alterado

## Constitution Check

Constituição aplicável: a do workspace (`../.specify/memory/constitution.md`, v1.0.0); a do serviço ainda é template.

| Princípio | Veredito |
|---|---|
| I. Complexidade só se necessária/pedida | OK — a troca foi pedida; sem middlewares extras do chi além do necessário para equivalência (`GetHead`) e um handler 405 enxuto |
| II. Domínio e API têm teste automatizado | OK — testes de equivalência de 404/405/HEAD/`/docs/` serão acrescentados; suíte existente roda intacta |
| III. Código é a fonte da verdade | OK — o comportamento de referência foi lido de `handlers.go` e dos testes, não da spec 001 |
| IV. Divergência código/spec é pergunta | OK — nenhuma divergência encontrada; se o contrato do `Allow` for considerado dispensável, é decisão do mantenedor (ver Riscos) |

Re-avaliação pós-design: sem violações.

## Project Structure

### Documentation (this feature)

```text
specs/003-roteamento-chi/
├── plan.md
├── research.md
├── data-model.md        # sem mudança de dados (registro)
├── quickstart.md
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

Sem `contracts/`: o contrato externo (`openapi.yaml`) não muda.

### Source Code

```text
notificacao/
├── internal/adapter/http/
│   ├── handlers.go            # Rotas(): chi.NewRouter() + GetHead + NotFound/MethodNotAllowed
│   ├── roteamento_test.go     # NOVO — 404, 405+Allow, HEAD, /docs e /docs/*, rotas públicas sem auth
│   └── (demais testes)        # inalterados
├── go.mod / go.sum            # + github.com/go-chi/chi/v5
└── internal/domain, usecase, platform, adapter/{postgres,amqp}  # NÃO mudam
```

**Structure Decision**: mesma estrutura hexagonal; tudo no adaptador `internal/adapter/http`.

## Riscos

- `Allow` no 405: o `ServeMux` o emite, o chi não. Plano: handler 405 próprio que o recompõe consultando o roteador (ver research D3). Se o mantenedor preferir aceitar a perda, remover esse handler.
- Corpos de 404/405 de texto puro: o chi usa `http.NotFound` / `"Method Not Allowed"`-equivalentes; teste compara com o comportamento anterior capturado.

## Complexity Tracking

Sem violações a justificar.
