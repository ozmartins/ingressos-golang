# Implementation Plan: Roteamento do Estoque via chi

**Branch**: `master` (nenhuma branch de feature — regra do mantenedor) | **Date**: 2026-10-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/003-roteamento-chi/spec.md`

## Summary

Trocar o `http.ServeMux` das duas superfícies HTTP do `Servico-Estoque` pelo chi, sem
alterar nada que um consumidor observe. A troca é pequena: `Rotas()` em
`internal/adapter/http/handlers.go`, `Handler()` em `internal/platform/health/health.go`
e uma dependência nova em `go.mod`. `main.go` não muda: continua recebendo
`http.Handler`.

O trabalho real está nas **diferenças de comportamento** entre os dois roteadores, que
foram medidas por experimento (ver [research.md](./research.md), D2): 405 sem corpo e com
`Allow` diferente, `HEAD` deixando de ser atendido, `/docs/` deixando de ser subárvore,
parâmetro codificado chegando cru e o 307 de caminhos não canônicos. Todas, menos o 307,
são compensadas no adaptador; o 307 é levado ao mantenedor (FR-005).

Nenhum contrato muda — esta feature **não tem `contracts/`**.

## Technical Context

**Language/Version**: Go 1.25 (`go.mod`)

**Primary Dependencies**: acrescenta `github.com/go-chi/chi/v5` v5.3.2 (já no cache de
módulos; exige Go 1.23). Nenhuma remoção.

**Storage**: N/A (nada de persistência é tocado)

**Testing**: `go test -race ./...` (inclui `test/arquitetura_test.go`),
`make lint` (depguard), `make test-integration` (já exercita `/health/ready`). Novo:
teste de caracterização de roteamento (D8) para API REST e saúde.

**Target Platform**: contêiner Linux (Dockerfile inalterado; `go mod download` resolve o
módulo novo)

**Project Type**: microsserviço (gRPC + REST + consumidores AMQP)

**Performance Goals**: mesma ordem de grandeza de hoje (SC-006); o chi roteia por árvore
radix sem alocação por requisição no caso comum.

**Constraints**: comportamento observável idêntico (FR-002); config/portas inalteradas
(FR-009); núcleo sem importar o chi (FR-008).

**Scale/Scope**: 2 rotas de API de negócio + 3 de documentação + 2 de saúde.

## Constitution Check

*GATE: passa antes da Fase 0; reavaliado após a Fase 1.*

| Princípio | Veredito |
|-----------|----------|
| I. Dependências apontam para dentro | **Passa.** O chi vive só em `internal/adapter/http` e `internal/platform/health`. `arquitetura_test.go` e o depguard recebem `github.com/go-chi` na lista de proibidos ao núcleo (verificação mecânica). |
| II. Configuração externa | N/A — nenhuma configuração nova. |
| III. Fronteira de estado | N/A. |
| IV. Erro é contrato | **Passa.** Respostas 401/4xx continuam em `escreverProblem`; 404/405 preservam o formato atual (D3). |
| V. Orçamento de integração | N/A. |
| VI. Entrega ao menos uma vez | N/A. |
| VII. Complexidade só se necessária | **Passa, com registro.** Só `Get/Post/Handle`, `MethodNotAllowed` e `GetHead`; sem `Route/Group/Mount`. O único artefato "extra" é o construtor comum de roteador (D9), justificado por evitar duas cópias de lógica sutil. Rejeitado por complexidade: replicar o 307 (D7). |
| VIII. Domínio e API têm teste | **Passa.** Teste de caracterização cobre as duas superfícies HTTP, inclusive a saúde (hoje sem teste); testes existentes ficam inalterados. |
| IX. Código é a fonte da verdade | **Passa.** Comportamento atual verificado no código e por experimento. Ao escrever o rascunho da spec, duas afirmações estavam erradas (autenticação do mapa; existência de teste de rotas × OpenAPI) e foram corrigidas contra o código antes do plano. |
| X. Divergência é pergunta | **Resolvida:** o 307 de caminhos não canônicos (D7) deixa de existir (passa a 404); diferença aceita pelo mantenedor em 2026-10-07. |

Pós-Fase 1: sem mudança de veredito.

## Project Structure

### Documentation (this feature)

```text
specs/003-roteamento-chi/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code

```text
estoque/
├── go.mod / go.sum                              # + github.com/go-chi/chi/v5
├── .golangci.yml                                # depguard: + github.com/go-chi
├── test/arquitetura_test.go                     # proibidos: + github.com/go-chi
├── internal/adapter/http/
│   ├── roteador.go                              # NOVO: NovoRoteador() (GetHead + 405 compatível)
│   ├── roteador_test.go                         # NOVO: teste de caracterização (D8)
│   └── handlers.go                              # Rotas() em chi; helper sessaoID (D6)
└── internal/platform/health/
    ├── health.go                                # Handler() usa NovoRoteador()
    └── health_test.go                           # NOVO: caracterização da saúde
```

**Structure Decision**: sem pacote novo; mudança confinada a dois pacotes de
infraestrutura já existentes.

## Complexity Tracking

Sem violações. Itens com justificativa registrada: construtor comum de roteador (D9) e
`Allow` por sondagem (D3).
