# Implementation Plan: Roteamento do Pagamento via chi

**Branch**: `master` (nenhuma branch de feature — regra do mantenedor) | **Date**: 2026-10-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/003-roteamento-chi/spec.md`

## Summary

Trocar o `http.ServeMux` por um `chi.Router` no `Servico-Pagamento`, sem alterar nada que um consumidor observe. A troca é local a `internal/adapter/http/handlers.go` (método `API.Rotas()` e a leitura do parâmetro `reserva_id`, hoje `r.PathValue`, que passa a `chi.URLParam`), mais `go.mod`/`go.sum` e a regra `depguard` em `.golangci.yml`. `cmd/pagamento/main.go` não muda: continua recebendo um `http.Handler` de `api.Rotas()`.

Decisões (detalhe em [research.md](./research.md)):

1. **chi v5 puro, sem middlewares.** Hoje o roteador não tem nenhuma camada transversal; adicionar `middleware.Logger`/`Recoverer` seria mudança de comportamento e escopo novo (D1).
2. **Respostas de 404/405 preservadas explicitamente** com `NotFound`/`MethodNotAllowed`, porque o `ServeMux` responde `404 page not found` e `405 Method Not Allowed` (texto simples, com `Allow`) e o chi tem padrões próprios (D2).
3. **`/docs` e `/docs/` registrados como rotas exatas**, pois o padrão `GET /docs/` do `ServeMux` casa subárvore e o do chi não (D3).
4. **`HEAD` preservado conforme medição (T005/T006)**: o `ServeMux` atende `HEAD` nas rotas `GET`; o chi não. A diferença é verificada e, se existir, preservada com tratamento explícito ou levada ao mantenedor (D4).
5. **Fronteira do núcleo**: `go-chi/chi` entra na lista de proibidos da regra `nucleo-sem-adaptadores` (D5).

Nenhum contrato muda (REST, AMQP), então esta feature **não tem `contracts/`**.

## Technical Context

**Language/Version**: Go 1.25 (`go.mod`)

**Primary Dependencies**: acrescenta `github.com/go-chi/chi/v5` (v1.5.5 é o que há no cache local; a versão v5 mais recente é resolvida com `go get`)

**Storage**: N/A (inalterado)

**Testing**: `go test -race ./...`; os testes HTTP existentes (`handlers_test.go`, `openapi_test.go`) exercitam `API.Rotas()` e são a rede de segurança; entram testes novos de 404/405, `/docs/`, `HEAD` e parâmetro de caminho

**Target Platform**: Linux server (container)

**Project Type**: web-service (microsserviço hexagonal)

**Performance Goals**: sem regressão perceptível (SC-006)

**Constraints**: contrato OpenAPI e comportamento observável idênticos; núcleo sem importar chi

**Scale/Scope**: 1 arquivo de produção alterado, 1 de lint, `go.mod`/`go.sum`, testes

## Constitution Check

Constituição de workspace (`../.specify/memory/constitution.md`, v1.0.0); o serviço não tem constituição própria.

- **I. Complexidade só se necessária ou pedida** — PASS. A troca foi pedida; não se adicionam middlewares, grupos de rota nem abstrações. Tratamentos de 404/405 só existem para preservar comportamento.
- **II. Domínio e API têm teste automatizado** — PASS. Testes de API existentes seguem valendo; casos novos cobrem os pontos em que roteadores divergem.
- **III. Código é a fonte da verdade** — PASS. O comportamento atual do `ServeMux` será medido por teste antes da troca (caracterização), não presumido.
- **IV. Divergência é pergunta, não decisão** — PASS, com ressalva: qualquer diferença inevitável de 404/405/`HEAD` é levada ao mantenedor.

Reavaliação pós-desenho: sem violações.

## Project Structure

### Documentation (this feature)

```text
specs/003-roteamento-chi/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code

```text
pagamento/
├── go.mod / go.sum                         # + github.com/go-chi/chi/v5
├── .golangci.yml                           # + chi na regra nucleo-sem-adaptadores
├── cmd/pagamento/main.go                   # inalterado
└── internal/adapter/http/
    ├── handlers.go                         # Rotas() com chi.NewRouter; chi.URLParam
    └── handlers_test.go / openapi_test.go  # + testes de 404/405, /docs/, HEAD
```

**Structure Decision**: mudança confinada ao adaptador HTTP; domínio, casos de uso, persistência e mensageria intocados.

## Complexity Tracking

Sem violações a justificar.
