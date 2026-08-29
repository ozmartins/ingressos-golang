# Implementation Plan: Catálogo de Filmes, Sessões e Reserva de Poltronas

**Branch**: `001-catalogo-sessoes-reserva` | **Date**: 2026-08-29 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-catalogo-sessoes-reserva/spec.md`

## Summary

O `Servico-Catalogo` expõe uma API REST pública para navegação do catálogo (filmes, cinemas, salas e grade de sessões) e um endpoint autenticado que traduz a intenção de reserva em uma chamada gRPC síncrona ao `Servico-Estoque`. Todo o estado de disponibilidade de poltronas vive no estoque; este serviço é dono apenas dos dados estruturais do catálogo.

A abordagem técnica é uma arquitetura hexagonal em Go: um núcleo de domínio e casos de uso sem dependências de infraestrutura, cercado por adaptadores de entrada (HTTP) e de saída (PostgreSQL, cliente gRPC do estoque, provedor de identidade). As três decisões que moldam o desenho são: paginação obrigatória e uniforme em todas as coleções (FR-001..FR-005), uma política de falha explícita na integração com o estoque — uma única tentativa, orçamento de 2s, recusa rápida após falhas consecutivas (FR-027..FR-030) — e observabilidade de primeira classe com correlação ponta a ponta (FR-034..FR-036).

## Technical Context

**Language/Version**: Go 1.22+ (definido pela ERS; `net/http.ServeMux` com padrões de método e caminho exige 1.22)

**Primary Dependencies**:
- `github.com/jackc/pgx/v5` — driver e pool PostgreSQL, SQL escrito à mão nos adaptadores
- `google.golang.org/grpc` + `google.golang.org/protobuf` — cliente do `Servico-Estoque`
- `github.com/coreos/go-oidc/v3` — descoberta OIDC do Keycloak e cache de JWKS
- `github.com/sony/gobreaker/v2` — recusa rápida após falhas consecutivas
- `go.opentelemetry.io/otel` (+ SDK, exportador OTLP, instrumentações `otelhttp` e `otelgrpc`) — métricas e rastreamento
- `log/slog` (stdlib) — logs estruturados em JSON
- `github.com/golang-migrate/migrate/v4` — migrações de esquema
- `github.com/testcontainers/testcontainers-go` — PostgreSQL real nos testes de integração

**Storage**: PostgreSQL 16 — tabelas `filmes`, `cinemas`, `salas`, `sessoes` conforme DDL da ERS. Somente leitura nesta feature; a escrita pertence ao processo administrativo externo.

**Testing**: `go test` (stdlib) em três camadas — unitários no domínio e casos de uso com portas falsas; contrato HTTP com `net/http/httptest`; integração com PostgreSQL via Testcontainers e estoque simulado via `grpc/test/bufconn`.

**Target Platform**: contêiner Linux (imagem distroless), atrás de um gateway/ingress que faz a limitação de volume por origem.

**Project Type**: microsserviço web (API REST + cliente gRPC), único módulo Go.

**Performance Goals**: p95 < 1s por página de qualquer coleção com 500 filmes, 50 cinemas, 300 salas e 5.000 sessões, e < 2s com dez vezes esse volume (SC-003); desfecho conclusivo da reserva em < 2,5s (SC-004); recusa rápida em < 200 ms com o estoque fora do ar (SC-007).

**Constraints**: timeout rígido de 2s na chamada ao estoque (FR-027); nenhuma retentativa automática (FR-029); nenhum estado de reserva persistido localmente (FR-031); leitura sempre atual, sem cache que sirva dado desatualizado (FR-010); toda configuração injetada por variáveis de ambiente (FR-032).

**Scale/Scope**: 4 agregados de leitura, 6 endpoints REST, 1 método gRPC consumido; volumes de SC-003 com margem de uma ordem de grandeza.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Avaliado contra `.specify/memory/constitution.md` **v1.0.0** (ratificada em 2026-08-29).

| Princípio | Veredito | Evidência no desenho |
|---|---|---|
| I. Dependências Apontam Para Dentro | **PASS** | `internal/domain` e `internal/usecase` não importam `internal/adapter`; portas declaradas em `usecase/ports.go`; composição única em `cmd/catalogo/main.go`. Verificação mecânica pelo import-linter (T003), com teste negativo deliberado (T078) — o princípio exige verificação automatizada, não revisão |
| II. Configuração Externa, Falha na Largada | **PASS** | Configuração lida e validada uma vez em `internal/platform/config` (T008); processo recusa subir com variável ausente ou malformada (T009); nenhum segredo no artefato; subida só com ambiente verificada em T080 |
| III. Fronteira de Estado Explícita | **PASS** | Nenhuma tabela, cache ou fila de reserva: disponibilidade, bloqueio e expiração pertencem ao `Servico-Estoque` (FR-031). O resultado do estoque é repassado sem reinterpretação (FR-025). Nenhum cache de dados de domínio — FR-010 exige leitura sempre atual, verificada em T079. O único cache é o de chaves públicas do emissor, que não é dado de domínio |
| IV. Erro é Contrato | **PASS** | RFC 9457 em toda resposta de erro, com `type` estável por categoria (`contracts/errors.md`); texto humano livre para mudar; nenhum detalhe interno em `detail`, verificado em T076; `instance` carrega o identificador de rastreamento |
| V. Integração Síncrona Tem Orçamento | **PASS** | Timeout de 2s declarado (FR-027); nenhuma retentativa em operação não idempotente (FR-029); interrupção temporária após falhas consecutivas com retomada automática (FR-030); estoque fora do ar não degrada a navegação (SC-012, verificado em T049) |

**Restrições técnicas da constituição**: observabilidade atendida por FR-034..FR-036 (T018, T021, T047, T049); contrato antes da implementação atendido por `contracts/` gerado nesta fase, antes de qualquer código; mudança incompatível como versão nova é o que motiva o envelope de paginação entrar já na v1 (research.md D8); segredos nunca registrados, incluindo o token recebido (T063).

**Portões de qualidade**: esta feature passou por `/speckit-specify` → `/speckit-clarify` → `/speckit-plan` → `/speckit-tasks` → `/speckit-analyze`, na ordem que a constituição exige. A análise identificou um critério de sucesso que nenhuma implementação plausível poderia violar de forma verificável (SC-003, cláusula de invariância) e ele foi reescrito, conforme o último item do fluxo de desenvolvimento.

**Re-avaliação pós-Fase 1**: as decisões de design não introduziram violação. A complexidade adicional está registrada em Complexity Tracking, com a alternativa mais simples e o motivo da rejeição, como a governança exige.

## Project Structure

### Documentation (this feature)

```text
specs/001-catalogo-sessoes-reserva/
├── plan.md              # Este arquivo
├── spec.md              # Especificação da feature
├── research.md          # Fase 0 — decisões técnicas e alternativas
├── data-model.md        # Fase 1 — entidades, invariantes e consultas
├── quickstart.md        # Fase 1 — como subir e validar ponta a ponta
├── contracts/
│   ├── openapi.yaml     # Contrato REST público
│   ├── estoque.proto    # Contrato gRPC consumido (cliente)
│   └── errors.md        # Catálogo de erros RFC 9457
├── checklists/
│   └── requirements.md  # Checklist de qualidade da spec
└── tasks.md             # Fase 2 — gerado por /speckit-tasks
```

### Source Code (repository root)

```text
catalogo/
├── cmd/
│   └── catalogo/
│       └── main.go                     # Composição: config → adaptadores → casos de uso → servidor
├── internal/
│   ├── domain/                         # Núcleo: entidades, regras e erros de domínio
│   │   ├── catalogo/                   # Filme, Cinema, Sala, Sessao + invariantes
│   │   ├── reserva/                    # SolicitacaoReserva, ResultadoReserva
│   │   └── shared/                     # Page, PageRequest, erros sentinela
│   ├── usecase/                        # Casos de uso + portas (interfaces)
│   │   ├── listar_filmes.go
│   │   ├── listar_cinemas.go
│   │   ├── listar_salas.go
│   │   ├── consultar_sessoes.go
│   │   ├── reservar_poltronas.go
│   │   └── ports.go                    # FilmeRepository, SessaoRepository, EstoqueGateway…
│   ├── adapter/
│   │   ├── http/                       # Adaptador de entrada
│   │   │   ├── router.go               # net/http.ServeMux, rotas /api/v1/*
│   │   │   ├── handler_*.go            # Um handler por caso de uso
│   │   │   ├── dto.go                  # Envelopes de request/response
│   │   │   ├── pagination.go           # Parsing e validação de page/page_size
│   │   │   ├── problem.go              # Serialização RFC 9457
│   │   │   └── middleware/             # authn, correlação, recuperação, telemetria
│   │   ├── postgres/                   # Adaptador de saída — repositórios
│   │   │   ├── pool.go
│   │   │   ├── filme_repository.go
│   │   │   ├── cinema_repository.go
│   │   │   ├── sala_repository.go
│   │   │   └── sessao_repository.go
│   │   ├── estoque/                    # Adaptador de saída — cliente gRPC
│   │   │   ├── client.go               # Timeout de 2s, sem retentativa
│   │   │   ├── breaker.go              # Recusa rápida e retomada automática
│   │   │   └── mapper.go               # Resposta do estoque → resultado de domínio
│   │   └── identidade/
│   │       └── keycloak.go             # Verificação de token via JWKS em cache
│   └── platform/
│       ├── config/                     # Leitura e validação de variáveis de ambiente
│       ├── observability/              # slog, OTel (tracer, meter), métricas nomeadas
│       └── health/                     # Indicador de saúde
├── gen/pb/estoque/                     # Código gerado do estoque.proto (não editar)
├── migrations/                         # Migrações do esquema (golang-migrate)
├── test/
│   ├── contract/                       # Handlers HTTP contra o contrato OpenAPI
│   ├── integration/                    # Postgres real (Testcontainers) + estoque em bufconn
│   └── fixtures/                       # Dados de carga do catálogo
├── Dockerfile
├── docker-compose.yml                  # Postgres + Keycloak + estoque simulado, para desenvolvimento
└── go.mod
```

**Structure Decision**: módulo Go único com arquitetura hexagonal, conforme a ERS exige. `internal/domain` e `internal/usecase` não importam nada de `internal/adapter` nem bibliotecas de infraestrutura; a dependência é invertida por interfaces declaradas em `usecase/ports.go` e implementadas pelos adaptadores. A composição acontece apenas em `cmd/catalogo/main.go`, o que mantém os casos de uso testáveis sem banco, sem rede e sem servidor HTTP. `gen/pb` é isolado porque é código gerado a partir do contrato do estoque, e `platform` concentra o que é transversal (configuração, telemetria, saúde) sem virar um pacote-depósito.

## Requirements Coverage

Onde cada requisito da spec é resolvido no desenho. A coluna de artefato aponta o documento que carrega a decisão; a implementação virá de `tasks.md`.

| Requisitos | Como são atendidos | Artefato |
|---|---|---|
| FR-001..FR-005 | Contrato único de paginação: `page`/`page_size`, envelope com total e `tem_proxima`, ordenação com desempate por `id`, página vazia além do fim | research.md D3; data-model.md (Paginação); contracts/openapi.yaml |
| FR-006..FR-009 | `GET /api/v1/filmes` com filtro `status`, campos opcionais omitidos, filtro desconhecido recusado | contracts/openapi.yaml; contracts/errors.md |
| FR-010 | Leitura direta do PostgreSQL; nenhum cache de dados de domínio (o cache de JWKS não é dado de catálogo) | research.md D4 |
| FR-011..FR-013 | `GET /api/v1/cinemas` e `GET /api/v1/cinemas/{id}/salas`, com 404 distinguível | contracts/openapi.yaml; contracts/errors.md |
| FR-014..FR-018 | `GET /api/v1/sessoes` com junção das quatro tabelas, filtros combináveis, recorte por situação e filtro de data por intervalo | data-model.md (Sessao, Consultas); contracts/openapi.yaml |
| FR-019..FR-021 | Middleware de autenticação com verificação local via JWKS; `sub` vira `usuario_id`; ausência de `sub` é 401 | research.md D4; contracts/errors.md |
| FR-022, FR-023 | Validações no caso de uso antes de qualquer ida à rede: sessão reservável, lista não vazia e sem duplicatas | data-model.md (SolicitacaoReserva) |
| FR-024, FR-025 | Adaptador gRPC encaminha sessão, poltronas e `usuario_id`; resposta vira 201 com `reserva_id` e `expira_em` | research.md D5; contracts/estoque.proto |
| FR-026 | Tudo-ou-nada delegado ao estoque; `sucesso=false` mapeia para 409 com `type` próprio | research.md D5; contracts/errors.md |
| FR-027..FR-030 | `context.WithTimeout(2s)`, nenhuma retentativa, disjuntor com abertura e retomada automáticas, 503 uniforme | research.md D5 |
| FR-031 | Nenhuma tabela, cache ou fila de reserva no desenho; objetos de reserva são transientes | data-model.md (ResultadoReserva); Constitution Check |
| FR-032 | Configuração lida e validada na inicialização; processo falha se faltar obrigatória | research.md D10 |
| FR-033 | Registro de auditoria dedicado por solicitação de reserva, com `usuario_id`, poltronas e desfecho, sem token | research.md D6 |
| FR-034..FR-036 | `slog` JSON com `trace_id`, métricas OTel por operação, propagação W3C via `otelhttp`/`otelgrpc` | research.md D6; quickstart.md (cenário 6) |
| FR-037 | `GET /health` com verificação do banco; estoque fora do ar não derruba a saúde | contracts/openapi.yaml; quickstart.md |
| FR-038 | RFC 9457 em toda resposta de erro, com `type` estável por categoria | research.md D7; contracts/errors.md |

Os critérios de sucesso mensuráveis têm caminho de verificação em `quickstart.md`: SC-003 e SC-008 no cenário 2, SC-004 e SC-007 no cenário 5, SC-005 e SC-011 nos cenários 4 e 6, SC-006 na suíte automatizada, SC-012 no cenário 5.

## Complexity Tracking

> Preenchido porque duas decisões custam mais que a alternativa ingênua e precisam de justificativa explícita.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Recusa rápida com máquina de estados (`gobreaker`) na chamada ao estoque | SC-007 exige resposta em < 200 ms durante falha sustentada e retomada automática em até 1 minuto; só o timeout de 2s deixaria cada requisição pagar o custo total da falha e manteria pressão sobre um serviço já degradado | Apenas timeout de 2s: viola SC-007 (resposta levaria ~2s, não < 200 ms) e mantém carga sobre o estoque em recuperação |
| Paginação uniforme em coleções de baixa cardinalidade (cinemas, salas) | FR-001 e a decisão explícita do usuário: contrato uniforme desde a v1 evita mudança incompatível quando o volume crescer | Retornar cinemas e salas na íntegra: mais simples hoje, mas exigiria quebra de contrato depois, e o cliente teria dois formatos de resposta para tratar |
