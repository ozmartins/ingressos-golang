# Implementation Plan: Roteamento do Catálogo via chi

**Branch**: `master` (nenhuma branch de feature — regra do mantenedor) | **Date**: 2026-10-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/003-roteamento-chi/spec.md`

## Summary

Trocar o roteador HTTP do `Servico-Catalogo` do `http.ServeMux` para o chi v5, sem alterar
nada que um consumidor observe. A troca cabe em um arquivo de produção
(`internal/adapter/http/router.go`) mais `go.mod`, uma regra no `.golangci.yml` e testes novos.

Decisões que moldam o desenho (detalhe em [research.md](./research.md)):

1. **Os handlers não mudam.** chi v5.3.2 chama `r.SetPathValue`; os 11 `r.PathValue("id")`
   seguem valendo (D1). A tabela `rotas` e `Rotas()` também não mudam (D2).
2. **Middlewares passam a `r.Use`**, na mesma ordem. Fora do chi, o middleware de log leria
   `r.Pattern` vazio e as métricas ganhariam o identificador como rótulo (D3 — achado
   verificado por execução).
3. **Paridade com o ServeMux, por decisão do mantenedor**: `r.Pattern` mantido como
   `"MÉTODO /caminho"`, HEAD atendido pela rota GET, e 405 com `Allow` completo e corpo
   `Method Not Allowed` (D4).
4. **Divergências residuais** (limpeza de caminho `//`/`..` e `%2F`) não são preserváveis a
   custo razoável e vão ao mantenedor (D5).

Nenhum contrato muda (REST, gRPC, AMQP), então esta feature **não tem `contracts/`**.

## Technical Context

**Language/Version**: Go 1.25 (`go.mod`)

**Primary Dependencies**: acrescenta `github.com/go-chi/chi/v5` v5.3.2 (já no cache de
módulos local, sem dependências transitivas). Nada sai.

**Storage**: N/A (nenhuma mudança).

**Testing**: `go test -race ./...` (inclui `docs_test.go`, paridade com o OpenAPI),
`go test ./test/contract/...` (contrato HTTP) e `make test-integration` (`reservar_test.go`
também monta o `NovoRouter`).

**Target Platform**: contêiner Linux distroless (sem mudança no Dockerfile; o
`go mod download` resolve o módulo novo).

**Project Type**: microsserviço (REST + gRPC cliente + publicador AMQP).

**Performance Goals**: sem mudança perceptível; o roteamento é desprezível frente ao banco (SC-006).

**Constraints**: mesma ordem de middlewares; 404/405 também passam por telemetria,
recuperação e log; rótulo `rota` de log/métrica com o método.

**Scale/Scope**: 1 arquivo de produção alterado (`router.go`), `go.mod`/`go.sum`,
`.golangci.yml`, 1 arquivo de teste novo, atualização documental da spec 001 e do README.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Avaliado contra `.specify/memory/constitution.md` **v1.2.0**.

| Princípio | Veredito | Evidência no desenho |
|---|---|---|
| I. Dependências Apontam Para Dentro | **PASS** | chi só em `internal/adapter/http`. `github.com/go-chi` entra na regra `nucleo-sem-adaptadores` do depguard (`make lint`) |
| II. Configuração Externa, Falha na Largada | **PASS** | Nenhuma configuração nova |
| III. Fronteira de Estado Explícita | **PASS** | Sem estado; o roteador não guarda nada |
| IV. Erro é Contrato | **PASS** | 404/405 continuam como hoje (texto simples do stdlib, sem detalhe interno); recuperação de pânico continua devolvendo o `problem+json` de erro interno, inclusive em 404/405 |
| V. Integração Síncrona Tem Orçamento | **PASS** | Sem chamada externa nova |
| VI. Entrega de Fato é Ao Menos Uma Vez | **PASS** | Não tocado |
| VII. Complexidade Só Entra Se For Necessária ou Pedida | **PASS** | A troca foi pedida. Só `GetHead` do `chi/middleware` e ~15 linhas de 405; handlers e tabela intactos. Alternativas rejeitadas em D1/D4 |
| VIII. Domínio e API Têm Teste Automatizado | **PASS** | Teste novo percorre a tabela de rotas e cobre HEAD, 405, 404 e o rótulo visto pelo `Log` (D7); suítes existentes são critério de aceite |
| IX. O Código é a Fonte da Verdade | **PASS** | Comportamento atual medido por execução (ServeMux × chi), não lido da spec 001 |
| X. Divergência Entre Código e Spec é Pergunta | **ATENÇÃO** | Ver abaixo |

**Re-avaliação pós-Fase 1**: o desenho não introduziu violação. Nenhuma entrada em
Complexity Tracking.

### Divergências (princípio X)

1. **405/HEAD/Pattern entre ServeMux e chi** — respondida em 2026-10-07: **preservar** (D4).
2. **Spec 001 × código após a troca** — `001/research.md` (decisão "ServeMux", com `chi`
   listado como alternativa rejeitada), `001/plan.md` (linhas 15 e 102) e `001/tasks.md`
   (T020) descrevem o ServeMux. Na 002 o mantenedor decidiu **atualizar a 001** para a
   persistência; a mesma decisão é aplicada aqui como tarefa (T012), **sujeita a
   confirmação** se preferirem outro caminho.
3. **Divergências residuais** (D5: `//`/`..` e `%2F`) — **em aberto**, a confirmar pelo
   mantenedor; nenhuma foi aceita ou corrigida por conta própria.

## Project Structure

### Documentation (this feature)

```text
specs/003-roteamento-chi/
├── plan.md              # Este arquivo
├── spec.md
├── research.md          # Fase 0 — decisões D1..D7
├── data-model.md        # Fase 1 — sem mudança de dados
├── quickstart.md        # Fase 1 — como validar a equivalência
├── checklists/requirements.md
└── tasks.md             # Fase 2 — /speckit-tasks
```

Sem `contracts/`: nenhum contrato externo muda.

### Source Code (catalogo/)

```text
catalogo/
├── internal/adapter/http/
│   ├── router.go                  # NovoRouter: chi.NewRouter, r.Use(...), r.Method(...), 405 próprio
│   ├── router_test.go             # NOVO: tabela de rotas, HEAD, 405, 404, Pattern visto pelo Log
│   ├── handlers.go                # INTOCADO (r.PathValue continua valendo)
│   └── middleware/                # INTOCADO (Encadear deixa de ser usada por NovoRouter)
├── .golangci.yml                  # + "github.com/go-chi" na regra nucleo-sem-adaptadores
├── go.mod / go.sum                # + github.com/go-chi/chi/v5 v5.3.2
├── README.md                      # revisar menção ao roteador, se houver
├── specs/001-catalogo-sessoes-reserva/{research,plan,tasks}.md   # atualizados (T012)
└── internal/domain, internal/usecase, cmd/catalogo               # INTOCADOS
```

**Structure Decision**: sem pacote novo; o chi fica em `internal/adapter/http`. `main.go` e
os harnesses de teste não mudam, porque a assinatura de `NovoRouter(Dependencias) http.Handler`
é preservada.

## Complexity Tracking

Nenhuma violação a justificar.
