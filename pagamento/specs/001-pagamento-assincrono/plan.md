# Implementation Plan: Processamento Assíncrono de Pagamentos (Servico-Pagamento)

**Branch**: `master` (nenhuma branch de feature criada — não há hook de git configurado) | **Date**: 2026-08-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-pagamento-assincrono/spec.md`

## Summary

O `Servico-Pagamento` consome o fato `reserva.criada`, cobra a reserva por trás de
uma porta de adquirente e anuncia o desfecho em `pagamento.sucesso` ou
`pagamento.falhou`. Expõe uma única operação síncrona — a consulta do andamento
pelo identificador da reserva — protegida por JWT.

A abordagem é arquitetura hexagonal em Go, sobre **uma única tabela**, com quatro
decisões que moldam todo o desenho:

1. **A cobrança única mora na restrição `UNIQUE (reserva_id)`** (research D2).
   `INSERT ... ON CONFLICT DO NOTHING RETURNING *` é o que decide, sob entregas
   simultâneas, qual delas cobra. Sem tabela de mensagens processadas, sem lock
   distribuído: a linha da transação **é** o registro de "já processei".
2. **O anúncio é garantido por uma coluna booleana, não por caixa de saída**
   (D3, FR-014). Ordem invariável: gravar estado final → publicar → marcar →
   confirmar a mensagem. A fila é o mecanismo de reenvio; a coluna fecha a janela
   entre gravar e publicar.
3. **Ausência de resposta do adquirente é um estado do domínio**, não uma recusa
   (D4, clarificação Q2). `PENDENTE_VERIFICACAO` é terminal, nunca anunciado, e a
   mensagem vai para a quarentena. É o único silêncio deliberado do serviço.
4. **O contrato de entrada é o da ERS, e a lacuna do estoque é dependência de
   integração declarada** (D1) — decisão do mantenedor sobre a divergência entre
   spec e código, tomada em 2026-08-30 pelo princípio IV.

## Technical Context

**Language/Version**: Go 1.25 (a ERS fixa 1.22+; alinhado a catálogo e estoque)

**Primary Dependencies**:
- `github.com/jackc/pgx/v5` — driver e pool PostgreSQL, SQL à mão para manter o `ON CONFLICT` visível
- `github.com/rabbitmq/amqp091-go` — consumo e publicação AMQP, ack manual e publisher confirms
- `github.com/golang-migrate/migrate/v4` — migrações versionadas
- `github.com/google/uuid` — identidade da transação
- biblioteca de JOSE/JWKS para validação stateless do token do Keycloak
- `go.opentelemetry.io/otel` (+ SDK, OTLP) — métricas e rastreamento
- `log/slog` (stdlib) — logs estruturados em JSON
- `net/http` (stdlib) — a API tem uma rota; roteador de terceiro não se paga aqui
- `github.com/testcontainers/testcontainers-go` — Postgres e RabbitMQ reais nos testes

**Storage**: PostgreSQL 16, tabela única `transacoes_pagamento` — DDL da ERS mais
`resultado_anunciado`, `pago_em` e o estado `PENDENTE_VERIFICACAO`, cada um
justificado em [data-model.md](./data-model.md) §1. Sem Redis, sem cache: nada no
caminho crítico se beneficia deles.

**Testing**: `go test` em quatro camadas (research D12) — domínio puro com relógio
injetado; casos de uso com portas falsas; contrato HTTP sobre `httptest`; integração
com Testcontainers para concorrência (SC-002), reentrega após queda (SC-003),
esgotamento até a fila morta (SC-006) e rajada com teto de concorrência (SC-004).

**Target Platform**: contêiner Linux, rede interna do cluster. A API é exposta ao
aplicativo; o consumo é interno ao barramento.

**Project Type**: microsserviço — um consumidor AMQP, um servidor HTTP com uma rota
de negócio e duas de saúde, nenhuma rotina de fundo. Módulo Go único.

**Performance Goals**: 95% das consultas < 1 s sob a rajada de SC-004; 1.000
reservas em um minuto processadas sem perda (SC-004); desfecho conhecido em < 30 s
da criação da reserva em operação normal (SC-008).

**Constraints**: teto de cobranças simultâneas configurável, padrão 10 (FR-019);
limite de entregas antes da quarentena configurável, padrão 3 (FR-021); prazo do
adquirente configurável (FR-022); nenhum dado sensível de meio de pagamento
trafega ou é registrado; resposta de reserva de terceiro indistinguível da de
reserva inexistente (FR-017).

**Scale/Scope**: 1 fila consumida, 2 fatos publicados, 1 tabela, 3 rotas HTTP,
2 casos de uso. Volume alvo verificado: rajada de 1.000 reservas por minuto.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Avaliado contra `../.specify/memory/constitution.md` — a constituição do
**workspace**, v1.0.0, princípios I a IV. **Este serviço não tem constituição
própria, por decisão explícita**: a do workspace determina que "serviço novo MUST
NOT recopiar estes princípios — herda-os deste arquivo". Catálogo e estoque têm
constituições próprias com princípios técnicos adicionais (dependências apontam
para dentro, erro é contrato, entrega ao menos uma vez); elas **não** governam este
serviço. Onde este plano adota práticas equivalentes, é escolha de projeto
registrada em `research.md`, não portão constitucional — e está dito assim de
propósito, para não fingir uma obrigação que não existe.

| Princípio | Veredito | Evidência no desenho |
|---|---|---|
| **I. Complexidade só entra se for necessária ou pedida** | **PASS** | Uma tabela, sem caixa de saída (D3), sem tabela de mensagens processadas (D2), sem Redis, sem rotina de fundo, sem roteador HTTP de terceiro, sem índice além do que a unicidade já dá (data-model §1). Em cada um desses pontos a alternativa mais elaborada — que o estoque legitimamente usa — foi rejeitada por escrito, com o motivo. Escopo não foi ampliado: estorno, nova tentativa, papel administrativo e reconciliação automática de `PENDENTE_VERIFICACAO` ficaram fora, como a spec delimita. |
| **II. Domínio e API têm teste automatizado** | **PASS** | Camadas 1 a 3 da D12 cobrem a máquina de estados, a regra de expiração e a classificação de motivo sem banco, sem rede e sem servidor. Toda operação exposta tem teste de sucesso e de cada categoria de erro declarada: a consulta contra `contracts/erros.md` §"API de consulta"; o consumo — que é interface exposta, ainda que por evento — contra §"consumo de eventos", incluindo os desfechos que **não** publicam nada. Percentual de cobertura não é portão em lugar nenhum deste plano. |
| **III. O código é a fonte da verdade** | **PASS** | A afirmação central deste plano sobre comportamento existente — o formato do `reserva.criada` real — foi verificada no código (`estoque/internal/usecase/bloquear_poltronas.go:15`) e no `.proto`, não na spec do estoque, e é exatamente onde a spec do estoque e o código **coincidem** mas a ERS do pagamento diverge. A compatibilidade dos fatos publicados com o consumidor real foi conferida do mesmo jeito. Nenhum artefato desta feature afirma que algo funciona: o quickstart declara o que ainda não foi executado. |
| **IV. Divergência entre código e spec é pergunta, não decisão** | **PASS** | A divergência do `reserva.criada` foi apresentada ao mantenedor com o que a spec diz, o que o código faz e o caminho mais curto para cada resolução, **antes** de qualquer artefato ser escrito. A resolução escolhida (opção A) está registrada em research D1 e na caixa de `contracts/eventos.md` §1. Nenhum lado foi "consertado" por conta própria e a divergência não foi silenciada — ela está declarada como dependência de integração aberta. |

**Portões de qualidade** (seção "Fluxo de Desenvolvimento" da constituição): a
feature seguiu `/speckit-specify` → `/speckit-clarify` → `/speckit-plan`. A
verificação por princípio está nesta seção, como exigido. Nenhuma violação
precisou ser registrada em Complexity Tracking. A exigência de que o estado de
conclusão seja aferido no código, e não na marcação de tarefas, vale a partir do
`/speckit-implement`.

**Re-avaliação pós-Fase 1**: as decisões de design não introduziram violação. O
desenho ficou **mais** simples do que o de partida em três pontos (D2, D3, D6), e
nenhum artefato da Fase 1 acrescentou componente que a Fase 0 não justificasse.

## Project Structure

### Documentation (this feature)

```text
specs/001-pagamento-assincrono/
├── plan.md              # Este arquivo
├── spec.md              # Especificação da feature
├── research.md          # Fase 0 — 12 decisões técnicas e alternativas rejeitadas
├── data-model.md        # Fase 1 — tabela, invariantes, máquina de estados, porta de repositório
├── quickstart.md        # Fase 1 — roteiro de validação ponta a ponta
├── contracts/
│   ├── eventos.md       # Fato consumido e fatos publicados + topologia RabbitMQ
│   ├── openapi.yaml     # API de consulta e saúde (contrato servido)
│   └── erros.md         # Categorias de erro, da API e do consumo
├── checklists/
│   └── requirements.md  # Checklist de qualidade da spec
└── tasks.md             # Fase 2 — gerado por /speckit-tasks
```

### Source Code (repository root)

```text
pagamento/
├── cmd/
│   ├── pagamento/main.go                 # Composição: config → adaptadores → casos de uso → servidor e consumidor
│   └── publicar/main.go                  # Publicador manual de reserva.criada, para o quickstart
├── internal/
│   ├── domain/
│   │   └── transacao/                    # Entidade, estados, transições, motivos, regra de expiração — sem import de adaptador
│   ├── usecase/
│   │   ├── ports.go                      # Repositorio, Adquirente, Publicador, Relogio, GeradorID
│   │   ├── processar_pagamento.go        # Fluxo do consumo: idempotência → expiração → cobrança → anúncio
│   │   └── consultar_pagamento.go        # Consulta por reserva, com a guarda de dono
│   ├── adapter/
│   │   ├── amqp/                         # Consumidor (ack manual), publicador (confirms), topologia
│   │   ├── postgres/                     # Repositório: ON CONFLICT, finalização condicionada, marca de anúncio
│   │   ├── adquirente/simulado/          # Único adaptador de adquirente desta entrega (D7)
│   │   └── http/                         # Rota de consulta, saúde, middleware de JWT
│   └── platform/
│       ├── config/                       # Leitura e validação do ambiente na largada
│       ├── health/                       # Vivacidade e prontidão
│       └── observability/                # slog JSON + OTel, propagação de traceparent
├── migrations/                           # 000001_criar_transacoes.{up,down}.sql
├── test/integration/                     # Testcontainers: concorrência, reentrega, fila morta, rajada
├── scripts/token-teste.sh                # Token assinado por chave de teste, para o quickstart
├── docker-compose.yml                    # Postgres + RabbitMQ
├── Dockerfile
└── Makefile
```

**Structure Decision**: mesmo desenho hexagonal de `catalogo/` e `estoque/`, no
mesmo nível do workspace, como módulo Go independente. A escolha não é cópia por
inércia: o domínio deste serviço tem regra própria de verdade (transições, motivos,
expiração) que precisa ser testável sem banco e sem broker, e é a porta `Adquirente`
que torna possível testar o desfecho indeterminado da D4 sem rede. As pastas acima
são só as que esta feature precisa — não há `grpc/`, `redis/` nem rotina de fundo,
porque nada aqui os exige.

## Complexity Tracking

> Preenchido apenas se o Constitution Check tiver violações a justificar.

Nenhuma violação. Todos os quatro princípios passaram sem concessão, e as
alternativas mais elaboradas que foram rejeitadas estão em `research.md` (D2, D3,
D5, D6, D10) com a necessidade que não se demonstrou e o motivo da rejeição — que
é o registro que o princípio I pede quando a solução mais simples **é** a escolhida.
