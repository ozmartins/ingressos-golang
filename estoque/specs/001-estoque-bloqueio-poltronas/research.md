# Fase 0 — Pesquisa e Decisões Técnicas

**Feature**: Bloqueio, Confirmação e Liberação de Poltronas (`Servico-Estoque`)
**Data**: 2026-08-29

Cada decisão registra o que foi escolhido, por quê, e o que foi rejeitado.
Referências `FR-xxx`/`SC-xxx` apontam para `spec.md`.

---

## D1 — Linguagem, versão e arquitetura

**Decisão**: Go 1.25 com arquitetura hexagonal — núcleo (`internal/domain`,
`internal/usecase`) sem dependência de infraestrutura, cercado por adaptadores
de entrada (servidor gRPC, consumidores AMQP, agendador de expiração) e de saída
(PostgreSQL, Redis, publicador AMQP).

**Justificativa**: a ERS fixa Go 1.22+ e Clean/Hexagonal. O serviço irmão
`Servico-Catalogo` já roda Go 1.25 com esse desenho; manter a mesma estrutura
reduz custo cognitivo e permite reaproveitar convenções (`internal/platform`,
`test/contract`, `test/integration`). O núcleo sem infraestrutura é o que torna
as regras de concorrência e de máquina de estados testáveis sem banco.

**Alternativas rejeitadas**: camadas por tipo técnico (`handlers/`, `models/`) —
não isola a regra de negócio e torna o teste de transição de estado dependente
de banco.

---

## D2 — Mecanismo de exclusividade: PostgreSQL autoritativo, Redis para prazo

**Decisão**: a exclusividade sobre poltronas (FR-005, FR-006) é garantida por
**transação PostgreSQL com `SELECT ... FOR UPDATE`**, travando as linhas das
poltronas solicitadas **em ordem determinística de rótulo** dentro da mesma
transação que grava reserva, vínculos e caixa de saída. O Redis **não participa
da correção do bloqueio**: ele carrega o índice de expiração (uma chave por
reserva com TTL de 10 minutos), que dispara a liberação em segundos.

**Justificativa**: a ERS admite explicitamente as duas alternativas ("Redis
Distributed Lock **ou** transações com `SELECT FOR UPDATE`"). A transação vence
por uma razão de correção, não de gosto: o cadeado e a mudança de estado são o
**mesmo** ato atômico. Com cadeado no Redis, a exclusividade depende da
durabilidade do Redis — um reinício do cache abre uma janela real de
*double-booking*, exatamente o defeito que o serviço existe para evitar. Além
disso, um cadeado externo introduz escrita dupla (Redis + Postgres) com falha
parcial possível entre as duas.

A ordenação determinística das linhas travadas elimina *deadlock* entre
solicitações que disputam conjuntos que se cruzam (A1,A2 vs A2,A1).

O Redis permanece no desenho — como a ERS prevê — com um papel que o banco não
faz bem: um gatilho de expiração de baixa latência via notificação de chave
expirada, em vez de depender só da varredura periódica. A varredura no
PostgreSQL continua existindo como rede de segurança autoritativa (D4), então
**perder o Redis inteiro degrada a pontualidade da liberação, nunca a correção**.

**Alternativas rejeitadas**:
- *Cadeado distribuído no Redis (`SETNX`) como autoridade*: janela de
  *double-booking* em reinício ou *failover* do Redis; escrita dupla.
- *Apenas varredura periódica, sem Redis*: seria suficiente para SC-003 (30 s) e
  mais simples, mas contraria a stack fixada pela ERS sem ganho de correção.
  Registrado em Complexity Tracking do `plan.md`.
- *Serialização por sessão em memória do processo*: quebra com mais de uma
  instância.

---

## D3 — Publicação de `reserva.criada`: caixa de saída transacional

**Decisão**: a mesma transação que concede o bloqueio grava uma linha em
`outbox_eventos`. Uma rotina publicadora lê as pendentes, publica em
`cinema.eventos` com *publisher confirms* e marca como publicadas. A resposta
gRPC não espera a publicação (FR-009).

**Justificativa**: FR-018 exige persistir antes de anunciar e reenviar em caso
de falha, sem duplicar reserva e sem invalidar o bloqueio. Publicar direto no
caminho da requisição viola o orçamento de 100 ms (SC-001) e perde o evento se o
broker estiver fora no instante da concessão (SC-005). A caixa de saída é o
padrão que resolve os dois de uma vez, ao custo de uma tabela e uma rotina.

**Consequência aceita**: entrega **ao menos uma vez** — declarada no contrato
(`contracts/eventos.md`), com `message_id = reserva_id` para o consumidor
deduplicar. FR-017 fala de "exatamente uma vez do ponto de vista de efeito
observável", que é o que a deduplicação por chave entrega.

**Alternativas rejeitadas**: publicação síncrona antes do commit (evento sem
reserva se o commit falhar); publicação após o commit sem outbox (evento perdido
se o processo morrer no meio).

---

## D4 — Expiração: notificação do Redis + varredura autoritativa

**Decisão**: duas fontes de disparo para o mesmo procedimento idempotente
`expirarReserva(reserva_id)`:
1. **Pronta**: chave `reserva:{id}` no Redis com TTL de 10 min e notificação de
   expiração (`notify-keyspace-events Ex`) consumida pelo serviço.
2. **Autoritativa**: varredura periódica (padrão 10 s) executando
   `UPDATE reservas SET status='EXPIRADA' WHERE status='PENDENTE' AND expira_em < now() RETURNING id`,
   protegida por *advisory lock* do PostgreSQL para que só uma instância varra
   por vez.

**Justificativa**: a notificação do Redis é *best effort* — não é entregue se
ninguém estiver escutando no instante, e some em reinício. Sozinha, ela violaria
FR-013 e SC-008 (reservas vencidas durante uma parada precisam ser invalidadas
no retorno). A varredura sozinha atende tudo, mas com latência de até um
intervalo. Juntas: liberação em segundos no caso comum, garantia no caso ruim.
O procedimento é idempotente e guardado por `status='PENDENTE'`, então os dois
gatilhos disparando para a mesma reserva não causam efeito duplo.

**Alternativas rejeitadas**: `pg_cron` (dependência de extensão no banco);
agendador externo (mais uma peça operacional para um laço de 10 s).

---

## D5 — Idempotência do consumo de eventos

**Decisão**: duas camadas. (a) Tabela `mensagens_processadas (fila, message_id)`
com chave primária composta, gravada **na mesma transação** do efeito; conflito
significa reentrega e a mensagem é confirmada sem reexecutar. (b) Guarda de
máquina de estados: toda transição parte de `PENDENTE` com `UPDATE ... WHERE
status='PENDENTE'` e checa as linhas afetadas.

**Justificativa**: FR-021 exige idempotência por `reserva_id`; FR-022 exige
ignorar sem alterar estado o desfecho dirigido a reserva já finalizada. A camada
(b) sozinha já torna duplicata e ordem invertida inofensivas — é ela que resolve
"chegou aprovação depois de recusa". A camada (a) evita reexecutar efeitos
colaterais não idempotentes (publicações futuras) e dá rastro auditável. O ack
depois do commit (FR-024) fecha o ciclo: nunca se confirma o que não se aplicou.

**Alternativas rejeitadas**: deduplicação só em memória (some no reinício);
confiar apenas no broker (RabbitMQ não oferece exatamente-uma-vez).

---

## D6 — Identidade da poltrona: rótulo determinístico + id derivado

**Decisão**: a identidade de negócio é o **rótulo** `fileira || numero` (ex.:
`"A1"`), único no escopo da sessão e usado no contrato gRPC e nos eventos
(FR-027). A chave primária `poltronas.id` da ERS é preenchida com um
**UUID v5 determinístico** de `sessao_id | fileira | numero` — 36 caracteres,
cabe no `VARCHAR(36)` da ERS, que já previa "hash de sessao_id + fileira +
numero".

**Justificativa**: reconcilia a contradição da ERS (tabela com UUID, evento com
`["A1","A2"]`) sem inventar um terceiro identificador. O cliente monta o rótulo
a partir do layout que já conhece, sem ida e volta ao estoque antes de reservar;
o banco mantém uma chave estável e o provisionamento vira naturalmente
idempotente (reprocessar `sessao.criada` recalcula o mesmo id e colide na chave).

**Alternativas rejeitadas**: UUID v4 aleatório (exigiria consulta prévia do
cliente e quebraria a idempotência do provisionamento); chave composta
`(sessao_id, fileira, numero)` como PK (rejeitada só por divergir da DDL da ERS;
tecnicamente equivalente).

---

## D7 — Segurança do canal síncrono: mTLS

**Decisão**: servidor gRPC com `tls.Config{ClientAuth: RequireAndVerifyClientCert}`,
CA de confiança e par certificado/chave lidos de caminhos informados por
ambiente (FR-037, FR-040). A identidade da pessoa usuária continua sendo
parâmetro confiável (FR-038). Recusa de handshake é registrada com o *subject*
apresentado, nunca com material criptográfico (FR-039).

**Justificativa**: sem verificação de chamador, qualquer processo com acesso à
rede reserva poltronas em nome de terceiros, já que `usuario_id` é apenas um
campo. mTLS resolve no transporte, sem lógica de negócio nova e sem colocar o
estoque para validar JWT (o catálogo já faz isso).

**Dependência de integração**: o `Servico-Catalogo` hoje disca em texto claro
(`ESTOQUE_GRPC_ADDR`, sem credencial). Ele precisa passar a apresentar
certificado de cliente. Enquanto isso não acontece, o modo é controlado por
configuração (`TLS_CLIENT_AUTH=require|off`), com `require` sendo o padrão em
produção e `off` permitido apenas em desenvolvimento — registrado como risco no
`plan.md`.

**Alternativas rejeitadas**: segredo compartilhado em metadado (segredo em log e
em variável de ambiente de todo cliente); confiar na rede (o que se quer evitar).

---

## D8 — Dependências de infraestrutura

| Peça | Escolha | Motivo |
|---|---|---|
| Driver PostgreSQL | `github.com/jackc/pgx/v5` | mesmo do catálogo; SQL à mão, `FOR UPDATE` explícito, pool nativo |
| Cliente AMQP | `github.com/rabbitmq/amqp091-go` | biblioteca oficial mantida; ack manual, confirms, prefetch |
| Cliente Redis | `github.com/redis/go-redis/v9` | padrão de fato; suporte a *pub/sub* de keyspace |
| gRPC | `google.golang.org/grpc` + `protobuf` | fixado pela ERS; contrato em `contracts/estoque.proto` |
| Migrações | `github.com/golang-migrate/migrate/v4` | mesmo do catálogo; migrações versionadas em `migrations/` |
| Observabilidade | `go.opentelemetry.io/otel` + `log/slog` | mesmo do catálogo; correlação ponta a ponta (FR-040) |
| Testes | `go test` + `testcontainers-go` | Postgres, Redis e RabbitMQ reais nos testes de integração |

**Justificativa**: alinhamento deliberado com o serviço irmão. Nenhuma
biblioteca nova é introduzida sem que a ERS a exija.

---

## D9 — Estratégia de testes

**Decisão**: quatro camadas.
1. **Unitários de domínio**: máquina de estados de reserva e poltrona, validação
   de solicitação, cálculo de expiração — com relógio injetado, sem infra.
2. **Casos de uso com portas falsas**: bloqueio, confirmação, cancelamento,
   expiração e provisionamento contra implementações em memória.
3. **Contrato gRPC**: servidor real sobre `bufconn`, verificando status e corpo
   de cada categoria de `contracts/erros.md`, inclusive recusa por mTLS.
4. **Integração**: PostgreSQL, Redis e RabbitMQ via Testcontainers — incluindo
   o **teste de concorrência de SC-002** (100 solicitações paralelas sobre o
   mesmo conjunto, exatamente uma concedida), a **reentrega duplicada** (SC-004),
   a **expiração após parada simulada** (SC-008) e o **ciclo aleatório de 1.000
   reservas** (SC-006).

**Justificativa**: os requisitos mais caros de errar são de concorrência e de
idempotência, e nenhum dos dois é verificável com dublês — precisam do banco e
do broker reais. O resto fica rápido e sem infra.

---

## D10 — Configuração e largada

**Decisão**: toda configuração por variável de ambiente, lida e validada uma vez
na inicialização; o processo recusa subir com variável obrigatória ausente ou
malformada. Chaves: `DATABASE_URL`, `REDIS_URL`, `RABBITMQ_URL`, `GRPC_ADDR`,
`TLS_CERT_FILE`, `TLS_KEY_FILE`, `TLS_CLIENT_CA_FILE`, `TLS_CLIENT_AUTH`,
`RESERVA_TTL` (padrão `10m`), `POLTRONAS_MAX_POR_BLOQUEIO` (padrão `10`),
`VARREDURA_EXPIRACAO_INTERVALO` (padrão `10s`), `AMQP_PREFETCH` (padrão `32`),
`OTEL_EXPORTER_OTLP_ENDPOINT`, `LOG_LEVEL`.

**Justificativa**: FR-041 e SC-010. Falhar na largada é barato; descobrir um
endereço vazio no primeiro bloqueio do dia é caro.

---

## D11 — Saúde e prontidão

**Decisão**: dois indicadores. *Liveness* responde enquanto o processo está vivo.
*Readiness* verifica banco, Redis e conexão AMQP e reprova a instância quando o
**PostgreSQL** está fora (sem ele não há bloqueio correto — FR-006). Redis fora
degrada apenas a pontualidade (D4) e é reportado como degradado, não como
inapto. Exposto por endpoint HTTP separado (porta de administração), fora do
canal gRPC de negócio.

**Justificativa**: FR-045. Amarrar prontidão ao Redis tiraria de operação um
serviço que ainda funciona corretamente.

---

## Questões em aberto que o plano não fecha

- **Volume esperado** (sessões simultâneas, poltronas por sala, pico de
  bloqueios/s) permanece não quantificado — item Outstanding do `/speckit-clarify`.
  Afeta dimensionamento de pool e prefetch, não o desenho. O teste de carga de
  SC-001 usa uma sala de 500 lugares e 100 solicitações concorrentes como piso.
- **Publicação de `sessao.criada` pelo catálogo** (D-integração de
  `contracts/eventos.md`) depende de uma mudança no serviço irmão.
- **Certificado de cliente no catálogo** (D7) idem.
