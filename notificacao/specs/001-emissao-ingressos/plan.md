# Implementation Plan: Emissão e Validação de Ingressos Digitais (Servico-Notificacao)

**Branch**: `master` (nenhuma branch de feature — não há hook de git configurado) | **Date**: 2026-08-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-emissao-ingressos/spec.md`

## Summary

O `Servico-Notificacao` consome o fato `pagamento.sucesso`, emite o ingresso digital da
reserva com um código de acesso assinado, avisa a pessoa e registra o aviso. Expõe duas
operações síncronas: a listagem dos ingressos da pessoa autenticada e a validação do
código na portaria, que dá baixa no ingresso no mesmo ato.

É o serviço mais simples da plataforma até aqui, e o plano protege essa simplicidade.
Arquitetura hexagonal em Go, duas tabelas, **nenhum fato publicado**, sobre quatro
decisões que moldam o desenho:

1. **A emissão única mora na restrição `UNIQUE (reserva_id)`** (research D2). Um
   `INSERT ... ON CONFLICT DO NOTHING RETURNING *` decide, sob entregas simultâneas,
   qual delas emite. A linha do ingresso **é** o registro de "já processei" — sem
   tabela de mensagens processadas, sem trava distribuída.
2. **A baixa na portaria é uma escrita condicionada, não uma leitura seguida de
   escrita** (D4). `UPDATE ... WHERE id = $1 AND status = 'VALIDO'`: uma linha afetada
   é a autorização. É o que satisfaz a FR-011 sem travar linha, e o motivo da recusa
   é uma segunda pergunta, feita só quando a primeira já disse não.
3. **O código de acesso é uma assinatura HMAC sobre o identificador opaco do
   ingresso** (D3). Verificável sem consultar o banco, sem carregar dado da pessoa,
   da reserva ou da compra. Código forjado é recusado antes de virar consulta.
4. **O aviso sai no mesmo processamento e a falha dele morre ali** (D6, clarificação 5).
   Um único caminho de execução; a garantia da FR-018 vira uma regra local — "este erro
   não sobe" — testável com uma porta falsa, sem rede.

A quinta decisão é sobre o que **não** existe: este serviço não publica evento algum
(D10). É a ponta da cadeia, e a coluna `resultado_anunciado` que o `Servico-Pagamento`
precisou para fechar a janela entre gravar e publicar aqui não teria função.

## Technical Context

**Language/Version**: Go 1.27.0 — verificado no ambiente (`go version`: go1.27.0) e em
`pagamento/go.mod`. A ERS pede 1.22+. Catálogo e estoque ainda estão em 1.25.0; adoto
1.27.0 por ser o toolchain instalado e o do serviço mais recente.

**Primary Dependencies** (todas já em uso no workspace, verificadas em `pagamento/go.mod`):
- `github.com/jackc/pgx/v5` — driver e pool PostgreSQL; SQL à mão, para manter o
  `ON CONFLICT` e o `UPDATE` condicionado visíveis no código
- `github.com/rabbitmq/amqp091-go` — consumo AMQP com ack manual
- `github.com/golang-migrate/migrate/v4` — migrações versionadas
- `github.com/google/uuid` — identidade do ingresso e do registro de aviso
- `github.com/MicahParks/keyfunc/v3` + `github.com/golang-jwt/jwt/v5` — validação
  stateless do token do Keycloak por JWKS
- `crypto/hmac`, `crypto/sha256`, `encoding/base64`, `crypto/subtle` (stdlib) — geração
  e verificação do código de acesso, comparação em tempo constante
- `go.opentelemetry.io/otel` (+ SDK, OTLP) — rastreamento, com `traceparent` vindo do AMQP
- `log/slog`, `net/http` (stdlib) — logs JSON e servidor. Duas rotas de negócio não
  pagam um roteador de terceiro
- `github.com/testcontainers/testcontainers-go` — Postgres e RabbitMQ reais nos testes

**Storage**: PostgreSQL 16. Duas tabelas, exatamente as da ERS —
`ingressos_emitidos` e `registros_notificacao` — mais as restrições `CHECK` que trazem
invariantes do domínio para o banco ([data-model.md](./data-model.md) §1 e §3). Um
índice: `(usuario_id, criado_em DESC)`, para a única consulta de leitura. Sem Redis,
sem cache.

**Testing**: `go test` em quatro camadas (research D12) — domínio puro com relógio
injetado; casos de uso com portas falsas; contrato HTTP sobre `httptest` cobrindo cada
categoria de `contracts/erros.md`; integração com Testcontainers para o que só falha de
verdade: emissão sob entregas simultâneas (SC-001), dupla leitura do mesmo código
(SC-004) e esgotamento até a fila morta (FR-022).

**Target Platform**: contêiner Linux na rede interna do cluster. A listagem é exposta ao
aplicativo; a validação, aos dispositivos de portaria; o consumo é interno ao barramento.

**Project Type**: microsserviço — um consumidor AMQP, um servidor HTTP com duas rotas de
negócio e duas de saúde, nenhuma rotina de fundo, nenhuma publicação. Módulo Go único.

**Performance Goals**: veredito da portaria em < 1 s em 99% das leituras (SC-003);
ingresso disponível em < 5 s do pagamento confirmado em 95% dos casos (SC-002);
listagem completa em < 2 s em 95% das consultas, para até 200 ingressos por pessoa
(SC-009).

**Constraints**: limite de entregas antes da quarentena configurável, padrão 3
(FR-022); `codigo_qr` nunca em log, atributo de rastro ou mensagem de erro (FR-021,
D13); código malformado, assinatura inválida e ingresso inexistente indistinguíveis na
resposta (FR-010); chave da portaria comparada em tempo constante; processo recusa subir
sem `INGRESSO_QR_SEGREDO` (D11).

**Configuração**: obrigatórias sem padrão — `DATABASE_URL`, `AMQP_URL`, `JWKS_URL`,
`JWT_ISSUER`, `JWT_AUDIENCE`, `INGRESSO_QR_SEGREDO`, `PORTARIA_API_KEY`. Com padrão —
`PORTA_HTTP` (8080), `AMQP_EXCHANGE` (`cinema.eventos`), `AMQP_EXCHANGE_DLX`
(`cinema.eventos.dlx`), `AMQP_FILA_PAGAMENTO_SUCESSO` (`notificacao.pagamento-sucesso`),
`AMQP_PREFETCH` (10), `AMQP_LIMITE_ENTREGAS` (3), `NOTIFICADOR_MODO` (`enviar`),
`NIVEL_LOG` (`info`). Valor inválido em chave com padrão é erro de largada, não queda
silenciosa no padrão (D11).

**Scale/Scope**: 1 fila consumida, 0 fatos publicados, 2 tabelas, 4 rotas HTTP,
3 casos de uso (emitir, validar, listar).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Avaliado contra `../.specify/memory/constitution.md` — a constituição do **workspace**,
v1.0.0, princípios I a IV.

**Sobre a constituição deste serviço**: `notificacao/.specify/memory/constitution.md`
está com os placeholders do template, nunca ratificada. Isso **não** é uma lacuna a
tapar: a constituição do workspace determina que "serviço novo MUST NOT recopiar estes
princípios — herda-os deste arquivo". Catálogo e estoque têm constituições próprias com
princípios técnicos adicionais; elas **não** governam este serviço. Onde este plano
adota práticas equivalentes às deles, é escolha de projeto registrada em `research.md`,
não portão constitucional — e está dito assim de propósito, para não fingir uma
obrigação que não existe. O mantenedor foi avisado de que pode ratificar uma
constituição de serviço com `/speckit-constitution`, e optou por seguir sem ela.

| Princípio | Veredito | Evidência no desenho |
|---|---|---|
| **I. Complexidade só entra se for necessária ou pedida** | **PASS** | Duas tabelas, um índice, nenhuma publicação de evento (D10), nenhuma rotina de fundo (D6), nenhum contador de tentativa próprio (D5), nenhuma caixa de saída, nenhum roteador HTTP de terceiro, nenhuma paginação (D8). Em cada ponto a alternativa mais elaborada está registrada com o motivo da rejeição — inclusive as que o `Servico-Pagamento` legitimamente adotou e que aqui não se pagam. Escopo não foi ampliado: gatilho de cancelamento, reenvio de aviso, rotação de segredo e integração com provedor real ficaram fora, exatamente como a spec delimita. |
| **II. Domínio e API têm teste automatizado** | **PASS** | D12 fixa quatro camadas. As camadas 1 e 2 cobrem transições, invariante de `utilizado_em`, idempotência e classificação quarentena/reentrega sem banco, sem rede e sem servidor. Toda operação exposta tem teste de sucesso e de **cada** categoria de erro declarada em `contracts/erros.md` — as duas rotas HTTP e também o consumo, que é interface exposta ainda que por evento. Percentual de linhas cobertas não é portão em lugar nenhum deste plano, e está dito nos artefatos. |
| **III. O código é a fonte da verdade** | **PASS** | A afirmação central deste plano sobre comportamento existente — o formato real do `pagamento.sucesso` — foi lida em `pagamento/internal/usecase/fatos.go` e `pagamento/internal/platform/config/config.go`, não na spec do pagamento. A tradução de `x-delivery-limit` (D5) é reaproveitada com a origem citada, e será reverificada por teste próprio em vez de aceita de segunda mão. Nenhum artefato desta feature afirma que algo funciona: o `quickstart.md` abre declarando que nada ali foi executado. |
| **IV. Divergência entre código e spec é pergunta, não decisão** | **PASS** | Nenhuma divergência bloqueante encontrada. O caso que poderia ser uma — o payload da ERS ter seis campos e o código publicar oito — foi verificado e **não** é divergência: o publicado é superconjunto do esperado (D1). Está registrado assim, com o que cada lado diz, em vez de silenciado. Uma incoerência menor foi encontrada **fora** desta feature e está reportada ao mantenedor sem ser corrigida por conta própria (ver abaixo). |

**Divergência encontrada fora do escopo, reportada e não decidida**: o `plan.md` do
`Servico-Pagamento` declara "Go 1.25", mas `pagamento/go.mod` diz `go 1.27.0`. É
incoerência entre artefato de spec e código, e pelo princípio IV a escolha entre
corrigir o artefato ou o `go.mod` é do mantenedor. Não toquei em nenhum dos dois: não é
matéria desta feature, não bloqueia nada aqui, e este plano usa a versão que o código e o
ambiente confirmam.

**Portões de qualidade** (seção "Fluxo de Desenvolvimento"): a feature seguiu
`/speckit-specify` → `/speckit-clarify` → `/speckit-plan`. A verificação por princípio
está nesta seção, como exigido. Nenhuma violação precisou ser registrada em Complexity
Tracking. A exigência de aferir conclusão no código, e não na marcação de tarefas, passa
a valer no `/speckit-implement`.

**Re-avaliação pós-Fase 1**: as decisões de design não introduziram violação. O desenho
ficou **mais** simples que o ponto de partida em três pontos — sem publicação de evento
(D10), sem coluna de controle de anúncio (D10) e sem contador de tentativas próprio
(D5) — e nenhum artefato da Fase 1 acrescentou componente que a Fase 0 não justificasse.

## Project Structure

### Documentation (this feature)

```text
specs/001-emissao-ingressos/
├── plan.md              # Este arquivo
├── spec.md              # Especificação da feature (5 clarificações integradas)
├── research.md          # Fase 0 — 13 decisões técnicas e alternativas rejeitadas
├── data-model.md        # Fase 1 — tabelas, invariantes, máquina de estados, portas
├── quickstart.md        # Fase 1 — roteiro de validação ponta a ponta
├── contracts/
│   ├── eventos.md       # Fato consumido + topologia RabbitMQ (nada é publicado)
│   ├── openapi.yaml     # Listagem, validação e saúde
│   └── erros.md         # Categorias de erro da API e do consumo
├── checklists/
│   └── requirements.md  # Checklist de qualidade da spec
└── tasks.md             # Fase 2 — gerado por /speckit-tasks
```

### Source Code (repository root)

```text
notificacao/
├── cmd/
│   ├── notificacao/main.go              # Composição: config → adaptadores → casos de uso → servidor e consumidor
│   └── publicar/main.go                 # Publicador manual de pagamento.sucesso, para o quickstart
├── internal/
│   ├── domain/
│   │   ├── ingresso/                    # Entidade, estados, transições, invariante de utilizado_em
│   │   └── aviso/                       # Registro de notificação: canal, desfecho, exigência de detalhe na falha
│   ├── usecase/
│   │   ├── ports.go                     # Ingressos, Avisos, Notificador, Relogio, GeradorID
│   │   ├── emitir_ingresso.go           # Fluxo do consumo: idempotência → emissão → aviso → registro
│   │   ├── validar_ingresso.go          # Baixa condicionada e classificação da recusa
│   │   └── listar_ingressos.go          # Listagem recortada pelo dono, ordenada, com filtro
│   ├── adapter/
│   │   ├── amqp/                        # Consumidor (ack manual) e topologia (quorum + DLX)
│   │   ├── postgres/                    # ON CONFLICT, UPDATE condicionado, listagem ordenada
│   │   ├── notificador/simulado/        # Único adaptador de aviso desta entrega (D6)
│   │   ├── codigo/                      # Geração e verificação do código assinado (D3)
│   │   └── http/                        # Duas rotas, saúde, JWT e chave de portaria
│   └── platform/
│       ├── config/                      # Leitura e validação do ambiente na largada
│       ├── health/                      # Vivacidade e prontidão
│       └── observability/               # slog JSON + OTel, propagação de traceparent
├── migrations/                          # 000001_criar_ingressos.{up,down}.sql
├── test/integration/                    # Testcontainers: emissão concorrente, dupla leitura, fila morta
├── scripts/token-teste.sh               # Token assinado por chave de teste, para o quickstart
├── docker-compose.yml                   # Postgres + RabbitMQ
├── Dockerfile
└── Makefile
```

**Structure Decision**: mesmo desenho hexagonal de `catalogo/`, `estoque/` e
`pagamento/`, no mesmo nível do workspace, como módulo Go independente. Não é cópia por
inércia: o domínio deste serviço tem regra própria — transições do ingresso e a
invariante que amarra `utilizado_em` ao estado — que precisa ser testável sem banco e
sem broker, como o princípio II exige. E são as portas `Notificador` e `Ingressos` que
tornam possível testar a FR-025 (falha de aviso não vira reprocessamento) sem rede.

Duas pastas do `pagamento/` **não** aparecem aqui, e a ausência é deliberada: não há
`adquirente/` porque não há terceiro a chamar no caminho crítico, e o publicador AMQP
não existe porque este serviço não publica (D10). A pasta `codigo/` é a única novidade
em relação aos outros serviços, e existe porque a geração do código assinado é
adaptador — depende de `crypto/hmac` — e não pode morar no domínio.

## Complexity Tracking

> Preenchido apenas se o Constitution Check tiver violações a justificar.

Nenhuma violação. Os quatro princípios passaram sem concessão.

As alternativas mais elaboradas que foram rejeitadas estão em `research.md` — D2
(tabela de mensagens processadas, trava distribuída), D3 (JWT no QR), D4
(`SELECT FOR UPDATE`), D5 (contador próprio de tentativas), D6 (rotina de fundo, caixa
de saída), D8 (paginação), D10 (publicar `ingresso.emitido`) — cada uma com a
necessidade que não se demonstrou e o motivo da rejeição. Esse é o registro que o
princípio I pede quando a solução mais simples **é** a escolhida.
