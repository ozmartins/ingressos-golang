# Implementation Plan: Bloqueio, Confirmação e Liberação de Poltronas (Servico-Estoque)

**Branch**: `master` (nenhuma branch de feature criada — não há hook de git configurado) | **Date**: 2026-08-29 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-estoque-bloqueio-poltronas/spec.md`

## Summary

O `Servico-Estoque` é o dono do estado de disponibilidade de poltronas do
sistema. Ele expõe um canal gRPC com duas operações — bloquear poltronas de forma
atômica e consultar o mapa de uma sessão — e reage a três fatos do barramento:
`sessao.criada` (provisiona a matriz), `pagamento.sucesso` (confirma) e
`pagamento.falhou` (libera). Reservas não pagas expiram sozinhas em 10 minutos.

A abordagem técnica é uma arquitetura hexagonal em Go, com três decisões que
moldam todo o desenho:

1. **A exclusividade mora no PostgreSQL** (`SELECT ... FOR UPDATE` em ordem
   determinística, na mesma transação que grava reserva, vínculos e evento). O
   Redis, que a ERS pede, fica com o índice de prazo — perder o Redis inteiro
   degrada a pontualidade da liberação, nunca a correção do bloqueio (research D2).
2. **Caixa de saída transacional** para `reserva.criada`: persistir antes de
   anunciar, reenviar até conseguir, sem prender a resposta de 100 ms (D3).
3. **Idempotência em duas camadas** no consumo — registro de mensagem processada
   e guarda de máquina de estados a partir de `PENDENTE` — o que torna duplicata
   e ordem invertida inofensivas por construção (D5).

## Technical Context

**Language/Version**: Go 1.25 (a ERS fixa 1.22+; alinhado ao `Servico-Catalogo`)

**Primary Dependencies**:
- `google.golang.org/grpc` + `google.golang.org/protobuf` — servidor gRPC (contrato próprio)
- `github.com/jackc/pgx/v5` — driver e pool PostgreSQL, SQL à mão com `FOR UPDATE` explícito
- `github.com/rabbitmq/amqp091-go` — consumo e publicação AMQP com ack manual e confirms
- `github.com/redis/go-redis/v9` — índice de prazo e notificação de chave expirada
- `github.com/golang-migrate/migrate/v4` — migrações versionadas
- `go.opentelemetry.io/otel` (+ SDK, OTLP, `otelgrpc`) — métricas e rastreamento
- `log/slog` (stdlib) — logs estruturados em JSON
- `github.com/testcontainers/testcontainers-go` — Postgres, Redis e RabbitMQ reais nos testes

**Storage**: PostgreSQL 16 — `poltronas`, `reservas`, `reserva_poltronas` (DDL da ERS,
com `rotulo` e `finalizado_em` acrescentados e justificados em `data-model.md`), mais
`outbox_eventos` e `mensagens_processadas`. Redis 7 — chaves de prazo, sem estado autoritativo.

**Testing**: `go test` em quatro camadas — domínio puro com relógio injetado; casos de
uso com portas falsas; contrato gRPC sobre `bufconn`; integração com Testcontainers,
incluindo concorrência (SC-002), reentrega (SC-004) e recuperação após parada (SC-008).

**Target Platform**: contêiner Linux, rede interna do cluster; canal gRPC protegido por
autenticação mútua de transporte.

**Project Type**: microsserviço — servidor gRPC + consumidores AMQP + rotinas de fundo
(publicador da caixa de saída, varredura de expiração), módulo Go único.

**Performance Goals**: p99 do bloqueio < 100 ms (SC-001); consulta do mapa de sala de
500 lugares < 200 ms p99 (SC-013); liberação por expiração em até 30 s do vencimento
(SC-003); reservas vencidas durante parada invalidadas em até 1 min do retorno (SC-008).

**Constraints**: nenhuma concessão sem garantia de exclusividade — banco fora significa
recusa, não bloqueio otimista (FR-006, SC-012); resposta síncrona não espera publicação
(FR-009); toda transição de reserva e poltronas é indivisível (FR-015); toda configuração
externa, com recusa de subir se faltar (FR-041); limite de 10 poltronas por bloqueio,
configurável (FR-004).

**Scale/Scope**: 2 operações gRPC, 3 filas consumidas, 1 evento publicado, 5 tabelas,
2 rotinas de fundo. Volume alvo não quantificado pelo cliente (item Outstanding do
`/speckit-clarify`); o piso verificado é sala de 500 lugares com 100 solicitações
concorrentes.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Avaliado contra `.specify/memory/constitution.md` **v1.0.0** (ratificada em 2026-08-29).
Os princípios I a V são idênticos aos da constituição do `Servico-Catalogo`; o VI é
próprio deste serviço.

| Princípio | Veredito | Evidência no desenho |
|---|---|---|
| I. Dependências Apontam Para Dentro | **PASS** | `internal/domain` e `internal/usecase` sem import de adaptador; portas em `usecase/ports.go` (T039); composição única em `cmd/estoque/main.go` (T024). Verificação mecânica pelo analisador de imports (T003), com teste negativo deliberado (T098) — o princípio exige verificação automatizada, não revisão |
| II. Configuração Externa, Falha na Largada | **PASS** | Todas as chaves de research D10 lidas e validadas uma vez (T012); processo recusa subir com variável ausente ou malformada (T013); material criptográfico vem de caminhos externos (T009, FR-040); subida só com ambiente verificada em T094 |
| III. Fronteira de Estado Explícita | **PASS** | Este serviço é o **dono** do estado de disponibilidade e não replica o cadastro de sessões — recebe o layout pelo fato `sessao.criada`. A segunda cláusula do princípio é o que fixa D2: a correção do bloqueio não depende do Redis, e perdê-lo inteiro não muda nenhuma resposta, apenas a pontualidade da liberação (verificado em T069) |
| IV. Erro é Contrato | **PASS** | `contracts/erros.md` fixa a categoria no status gRPC e em `ErrorInfo.reason`, versionada; `sucesso=false` reservado a indisponibilidade de poltrona; nenhum detalhe interno em resposta (T019, verificado por T031) |
| V. Integração Síncrona Tem Orçamento | **PASS** | O caminho síncrono não chama serviço externo; o travamento usa `NOWAIT` para recusar rápido em vez de esperar (data-model §8) e o banco indisponível vira recusa imediata, nunca concessão sem garantia (T035). Redis fora não degrada bloqueio nem consulta (T023, T069) |
| VI. Entrega de Fato é Ao Menos Uma Vez | **PASS** | Caixa de saída transacional (T044) com reenvio até confirmação (T045); resposta não espera publicação (T040, FR-009); consumo idempotente por chave declarada no contrato (T022, T052, T081), ack após commit (T021), DLQ para erro definitivo (T053). `contracts/eventos.md` declara entrega ao menos uma vez, sem prometer exatamente-uma-vez |

**Restrições técnicas**: observabilidade atendida por FR-042..FR-044 (T016, T021, T048, T093), incluindo a propagação do contexto de rastreamento **pelos fatos publicados** — exigência explícita da constituição, sustentada pela coluna `trace_context` da caixa de saída (T044/T045); contrato antes da implementação atendido por `contracts/` gerado na Fase 1, antes de qualquer código; mudança incompatível como versão nova é o que torna as adições ao `.proto` estritamente aditivas; segredos e material criptográfico nunca registrados (T018, T095).

**Portões de qualidade**: esta feature passou por `/speckit-specify` → `/speckit-clarify` → `/speckit-plan` → `/speckit-tasks` → `/speckit-analyze`, na ordem exigida. A análise apontou uma lacuna de propagação de rastreamento pela mensageria e cinco itens menores, todos corrigidos nos artefatos antes da implementação. A exigência de teste automatizado com infraestrutura real para invariante de concorrência e idempotência é atendida por T032, T052, T081 e T090.

**Re-avaliação pós-Fase 1**: as decisões de design não introduziram violação. A complexidade adicional está registrada em Complexity Tracking, com a alternativa mais simples e o motivo da rejeição, como a governança exige.

## Project Structure

### Documentation (this feature)

```text
specs/001-estoque-bloqueio-poltronas/
├── plan.md              # Este arquivo
├── spec.md              # Especificação da feature
├── research.md          # Fase 0 — decisões técnicas e alternativas
├── data-model.md        # Fase 1 — tabelas, invariantes, máquinas de estado, SQL do bloqueio
├── quickstart.md        # Fase 1 — como subir e validar ponta a ponta
├── contracts/
│   ├── estoque.proto    # Contrato gRPC servido (dono)
│   ├── eventos.md       # Fatos publicados e consumidos + topologia RabbitMQ
│   └── erros.md         # Categorias de erro por status gRPC
├── checklists/
│   └── requirements.md  # Checklist de qualidade da spec
└── tasks.md             # Fase 2 — gerado por /speckit-tasks
```

### Source Code (repository root)

```text
estoque/
├── cmd/
│   └── estoque/
│       └── main.go                       # Composição: config → adaptadores → casos de uso → servidor e rotinas
├── internal/
│   ├── domain/                           # Núcleo: sem import de infraestrutura
│   │   ├── poltrona/                     # Poltrona, Rotulo, Tipo, Status e transições válidas
│   │   ├── reserva/                      # Reserva, máquina de estados, prazo, invariantes
│   │   └── shared/                       # Erros de domínio, relógio (porta)
│   ├── usecase/
│   │   ├── ports.go                      # Repositórios, publicador, cadeado de prazo, relógio
│   │   ├── bloquear_poltronas.go         # FR-001..FR-009
│   │   ├── consultar_mapa.go             # FR-033..FR-035
│   │   ├── confirmar_reserva.go          # FR-019, FR-021..FR-024
│   │   ├── cancelar_reserva.go           # FR-020..FR-024
│   │   ├── expirar_reservas.go           # FR-012..FR-013
│   │   └── provisionar_sessao.go         # FR-029..FR-031
│   ├── adapter/
│   │   ├── grpc/                         # Servidor: mapeamento de erros, mTLS, interceptores
│   │   ├── postgres/                     # Repositórios, transação do bloqueio, varredura, outbox
│   │   ├── amqp/                         # Consumidores (3 filas), publicador, topologia, DLQ
│   │   └── redis/                        # Índice de prazo e escuta de expiração
│   └── platform/
│       ├── config/                       # Leitura e validação na largada
│       ├── observability/                # slog, métricas e rastreamento OTel
│       └── health/                       # Liveness e readiness na porta de administração
├── gen/pb/estoque/                       # Código gerado do .proto (buf)
├── migrations/                           # 000001_criar_esquema, 000002_criar_indices, ...
├── proto/                                # Cópia servida do contrato (fonte: specs/.../contracts)
├── test/
│   ├── contract/                         # Servidor real sobre bufconn, por categoria de erro
│   └── integration/                      # Testcontainers: concorrência, reentrega, expiração, mTLS
├── certs/                                # Material de desenvolvimento gerado por `make certs` (não versionado)
├── docker-compose.yml                    # postgres, migrate, redis, rabbitmq, estoque
├── Dockerfile
├── Makefile
├── buf.yaml / buf.gen.yaml
└── go.mod
```

**Structure Decision**: microsserviço único em módulo Go próprio (`estoque/`), espelhando
a estrutura já adotada em `catalogo/` — mesma separação `domain`/`usecase`/`adapter`/
`platform`, mesmos diretórios de teste e mesmo fluxo de migrações. A diferença de forma
em relação ao irmão é a ausência de adaptador HTTP de negócio (o canal de negócio é gRPC;
o HTTP existe só para saúde e métricas) e a presença de rotinas de fundo (publicador da
caixa de saída e varredura de expiração), compostas no mesmo `main.go`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Redis mantido no desenho, com papel reduzido a índice de prazo | A ERS fixa Redis na stack do serviço; ele entrega liberação em segundos por notificação de chave expirada, em vez de esperar o próximo ciclo de varredura | Só a varredura periódica (10 s) atenderia SC-003 com menos uma dependência, mas divergiria da stack exigida pela ERS. O papel foi reduzido ao mínimo: nenhuma decisão de correção depende do Redis (D2, D4) |
| Tabela `outbox_eventos` e rotina publicadora | FR-018 e SC-005 exigem que o evento sobreviva a um broker indisponível no instante da concessão, sem duplicar reserva nem prender a resposta de 100 ms | Publicar direto no caminho da requisição é mais simples, mas perde o evento se o broker estiver fora e coloca a latência do broker dentro do orçamento de SC-001 (D3) |
| Tabela `mensagens_processadas` além da guarda de estado | Dá rastro auditável de reentrega e protege efeitos colaterais não idempotentes futuros | Só a guarda `WHERE status='PENDENTE'` já torna duplicata inofensiva hoje; foi mantida como camada primária. A tabela é a segunda camada, e sua remoção não quebraria FR-021 (D5) |
| Coluna `rotulo` derivada de `fileira`+`numero` | Sem ela, o bloqueio decomporia `"A1"` em SQL e não usaria índice, ameaçando SC-001 | Chave composta `(sessao_id, fileira, numero)` no contrato seria equivalente, mas exigiria mudar a forma da mensagem já consumida pelo catálogo (D6) |

## Riscos e dependências externas

| Risco | Impacto | Encaminhamento |
|---|---|---|
| `Servico-Catalogo` ainda não publica `sessao.criada` | Sem ele, a matriz de poltronas não é provisionada automaticamente | Contrato proposto em `contracts/eventos.md`; até a adoção, carga administrativa documentada no `quickstart.md` |
| `Servico-Catalogo` disca em texto claro, sem certificado de cliente | Ativar mTLS obrigatório quebraria a integração hoje | `TLS_CLIENT_AUTH=require\|off` por ambiente, `require` como padrão de produção; a mudança no catálogo é pré-requisito de implantação (D7) |
| Cliente do catálogo mapeia todo erro não-`Unavailable` como falha genérica | `INVALID_ARGUMENT` chegaria ao usuário como 5xx | Documentado em `contracts/erros.md`; comportamento é seguro, apenas menos informativo, até o catálogo estender o mapeamento |
| Volume real não quantificado | Dimensionamento de pool, prefetch e intervalo de varredura ficam por estimativa | Valores configuráveis (D10); piso verificado por teste de carga em SC-001 |

## Próxima fase

`/speckit-tasks` gera `tasks.md` a partir destes artefatos. Este comando não o cria.
