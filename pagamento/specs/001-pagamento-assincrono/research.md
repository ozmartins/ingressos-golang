# Research — Processamento Assíncrono de Pagamentos (Servico-Pagamento)

**Fase 0** do `/speckit-plan` · Data: 2026-08-30 · Spec: [spec.md](./spec.md)

Cada decisão abaixo resolve um ponto que a spec deixa em aberto por ser
implementação, ou um conflito descoberto ao confrontar a spec com o código dos
serviços vizinhos. Formato: decisão, motivo, alternativas rejeitadas.

---

## D1 — O `reserva.criada` real não carrega valor nem forma de pagamento

**Divergência encontrada.** A ERS descreve o evento consumido com `valor_total` e
`forma_pagamento`. O código que efetivamente publica esse fato —
`estoque/internal/usecase/bloquear_poltronas.go:15`, `EventoReservaCriada` — emite
oito campos e nenhum dos dois. E não é omissão do publicador: o
`SolicitacaoBloqueio` de `estoque/proto/estoque.proto:30` recebe apenas
`sessao_id`, `poltronas_ids` e `usuario_id` — o estoque não tem essa informação.

Implementado como a spec pede, todo evento real cairia na FR-003 como inválido e
o serviço nunca cobraria nada.

**Decisão** (princípio IV — levada ao mantenedor em 2026-08-30, resposta: opção A):
o contrato de entrada deste serviço é o da ERS, com valor e forma de pagamento
obrigatórios, e a lacuna fica registrada como **dependência de integração** em
`contracts/eventos.md`. Enquanto o estoque não publicar os dois campos, o roteiro
ponta a ponta usa um publicador manual (`cmd/publicar`).

**Motivo**: é a única resolução que não amplia o escopo para outros serviços
(princípio I), e é o padrão que este repositório já adota para lacuna de
integração — o próprio estoque declara `sessao.criada` como proposta ao catálogo,
que ainda não o publica (`estoque/specs/.../contracts/eventos.md`, §4).

**Alternativas rejeitadas**:
- Alterar estoque e catálogo para propagar valor e forma: correto a médio prazo,
  mas mexe em dois serviços fora do escopo desta feature.
- Pagamento consultar o catálogo pelo preço de forma síncrona: cria dependência
  síncrona no caminho crítico de um serviço cuja razão de existir é ser assíncrono.
- Tornar os campos opcionais com valor padrão: cobrar valor adivinhado.

**Consequência aceita**: a validação ponta a ponta contra o estoque real fica
pendente até que ele publique os campos. Está declarado no quickstart.

---

## D2 — Idempotência pela unicidade de `reserva_id`, sem tabela de mensagens

**Decisão**: a garantia de cobrança única (FR-006, FR-007) é a restrição `UNIQUE`
sobre `reserva_id` em `transacoes_pagamento`, exercida por
`INSERT ... ON CONFLICT (reserva_id) DO NOTHING RETURNING *`. Quem insere, cobra;
quem colide, lê a transação existente e decide o que fazer sem cobrar.

**Motivo**: a chave natural de deduplicação deste domínio já é a reserva, e a
própria linha da transação é o registro de "já processei". O banco resolve a
corrida do cenário 2.3 sem lock distribuído, sem transação serializável e sem
estrutura nova.

**Alternativas rejeitadas**:
- Tabela `mensagens_processadas` por `(fila, message_id)`, como o estoque usa: lá
  ela é necessária porque um mesmo agregado recebe fatos de tipos diferentes;
  aqui seria um segundo registro dizendo o que a unicidade de `reserva_id` já diz.
  Duas fontes para o mesmo fato é exatamente o que o princípio I proíbe.
- Lock distribuído em Redis: dependência nova para um problema que o banco
  resolve com uma restrição declarativa.

---

## D3 — Garantia de anúncio: marca na transação, sem caixa de saída

**Decisão** (FR-014, clarificação Q3): a tabela ganha
`resultado_anunciado BOOLEAN NOT NULL DEFAULT FALSE`. A ordem de execução é
sempre: gravar o estado final → publicar com *publisher confirm* → marcar como
anunciado → confirmar a mensagem. A mensagem só é confirmada depois da
publicação; uma reentrega cuja transação já esteja final com anúncio pendente
publica a partir do que está gravado, sem tocar no adquirente.

**Motivo**: fecha a janela entre gravar e publicar com uma coluna booleana, sem
tabela nova nem processo publicador. Consumo e publicação usam o mesmo broker: se
ele está fora, o serviço também não está consumindo, então o cenário que a caixa
de saída protege quase não existe aqui.

**Alternativas rejeitadas**:
- Caixa de saída transacional (o que o estoque usa): tabela nova, rotina de
  drenagem nova e ordenação própria. Lá ela paga o próprio custo porque a resposta
  síncrona de 100 ms não pode esperar a publicação; aqui não há resposta síncrona
  a proteger — o consumidor já é assíncrono por natureza.
- Varredor periódico sobre transações com anúncio pendente: rotina de fundo nova
  para um caso que a reentrega da própria fila já cobre.

**Consequência aceita**: entrega **ao menos uma vez**. Uma queda entre publicar e
marcar republica o resultado. Está declarado em `contracts/eventos.md`, e o
estoque já deduplica por `reserva_id`.

---

## D4 — Ausência de resposta do adquirente vira estado próprio, não recusa

**Decisão** (FR-022, clarificação Q2): estourado o prazo, a transação vai para
`PENDENTE_VERIFICACAO`, nenhum resultado é publicado, nenhuma nova cobrança é
tentada, e a mensagem é rejeitada **sem reenfileirar** (`nack`, `requeue=false`),
caindo direto na fila morta. `PENDENTE_VERIFICACAO` é terminal e não anunciável.

**Motivo**: numa demora não se sabe se a cobrança foi efetivada. Recusar arrisca
negar poltrona já paga; reprocessar arrisca cobrar duas vezes. O estado próprio
não escolhe entre os dois defeitos e deixa o caso visível para inspeção.

**Alternativas rejeitadas**:
- Tratar como recusa: pode recusar cobrança que passou.
- Reenfileirar contando com a referência do adquirente para reconhecer a cobrança:
  só funciona se o adquirente devolver a referência **antes** do estouro, o que é
  exatamente o que não acontece num timeout.
- Consultar o adquirente pelo estado da cobrança antes de decidir: é a resolução
  automática que a spec deixou explicitamente fora de escopo.

---

## D5 — Contagem de tentativas: fila quórum com `x-delivery-limit`

**Decisão** (FR-021): `pagamento.reserva-criada` é uma fila **quórum** com
`x-delivery-limit: 3` (configurável) e `x-dead-letter-exchange` apontando para
`cinema.eventos.dlx`. Esgotado o limite, o broker encaminha para
`pagamento.reserva-criada.dlq` sozinho.

**Motivo**: é o único mecanismo que conta entregas sem estado do lado da
aplicação. Uma linha de argumento na declaração da fila substitui contador,
cabeçalho e exchange de repetição.

**Alternativas rejeitadas**:
- Contar `x-death` e reencaminhar por exchange de retentativa com TTL: três
  recursos AMQP a mais para reimplementar o que o broker já faz.
- Contador em coluna da transação: não cobre a falha que acontece *antes* de a
  transação existir (JSON inválido, banco fora no `INSERT`).

---

## D6 — Vazão: `prefetch` igual ao teto de cobranças simultâneas

**Decisão** (FR-019): `basic.qos(prefetch_count = N)`, padrão 10, com um conjunto
de N rotinas consumindo do mesmo canal. O teto de mensagens não confirmadas é o
próprio teto de cobranças em andamento.

**Motivo**: o `prefetch` já é o limitador de concorrência quando a rotina só
confirma ao terminar; um semáforo separado seria um segundo teto dizendo o mesmo.

**Alternativas rejeitadas**: limitador de taxa por janela de tempo — a ERS pede
teto de concorrência, não de taxa, e os dois não são a mesma coisa.

---

## D7 — Adquirente simulado atrás de uma porta do domínio

**Decisão** (clarificação Q1): a porta `Adquirente` declara
`Cobrar(ctx, cobrança) (resultado, erro)` com três desfechos distinguíveis —
aprovada com referência, recusada com motivo, e indeterminada. O único adaptador
desta entrega é `internal/adapter/adquirente/simulado`, cujo comportamento vem de
regras de configuração (por forma de pagamento e por faixa de valor) para o
roteiro manual, e é injetado diretamente nos testes.

**Motivo**: o desfecho indeterminado precisa ser um caso do contrato, não um erro
genérico — é o que sustenta a D4. Com a porta desenhada assim, trocar o simulado
por um adquirente real não altera domínio, caso de uso nem banco.

**Alternativas rejeitadas**: chamar o adquirente direto do caso de uso, sem porta
— tornaria a D4 impossível de testar sem rede.

---

## D8 — Autorização da consulta: `sub` do token contra o dono da transação

**Decisão** (FR-016, FR-017, clarificação Q4): validação stateless do JWT do
Keycloak por JWKS (assinatura, emissor, público e validade). O `sub` do token é
comparado ao `usuario_id` da transação; se diferirem, a resposta é a **mesma** de
reserva inexistente (404), sem corpo distinguível. Nenhum papel administrativo é
reconhecido.

**Motivo**: responder 403 para reserva de terceiro confirma que ela existe, o que
a FR-017 proíbe. Não reconhecer papel administrativo é o princípio I: ninguém pediu.

**Alternativas rejeitadas**: introspecção do token no Keycloak a cada requisição —
chamada de rede no caminho de leitura para obter o que a assinatura já garante.

---

## D9 — Expiração conferida contra relógio injetado, sem folga

**Decisão** (FR-005, clarificação Q5): antes de chamar o adquirente, o caso de uso
compara `expira_em` com o relógio injetado. Já vencida, a transação nasce e morre
`CANCELADO` com motivo `RESERVA_EXPIRADA`, e o sistema publica `pagamento.falhou`.
Sem margem de tolerância.

**Motivo**: relógio injetado é o que torna o caso testável sem espera real. A
margem foi oferecida ao mantenedor e recusada (Q5, opção A): cobrar dentro de uma
folga é cobrar por poltrona que o estoque já liberou.

---

## D10 — Persistência e migrações

**Decisão**: PostgreSQL 16 com `pgx/v5` e SQL escrito à mão; migrações versionadas
com `golang-migrate`. Uma única tabela, `transacoes_pagamento`, com o DDL da ERS
mais três colunas justificadas em `data-model.md`.

**Motivo**: alinhado ao que catálogo e estoque já usam no repositório; SQL à mão
mantém visível o `ON CONFLICT` de que a D2 depende, que é justamente o que um ORM
esconderia.

**Alternativas rejeitadas**: ORM — indireção sobre a única instrução SQL que
carrega a garantia central do serviço.

---

## D11 — Observabilidade e configuração

**Decisão**: `log/slog` em JSON, com `reserva_id` e `transacao_id` em todo registro
do fluxo; OpenTelemetry para métricas e rastreamento, extraindo `traceparent` dos
cabeçalhos AMQP e reinjetando-o nos fatos publicados, de modo que bloqueio e
desfecho de pagamento fiquem no mesmo rastro. Toda configuração vem do ambiente e
é validada uma vez na largada; falta ou valor malformado impede o processo de subir.

**Motivo**: o estoque já propaga `traceparent` pelos fatos; não fazer o mesmo aqui
quebraria o rastro exatamente no meio da jornada de compra. Nenhum dado sensível
de meio de pagamento é registrado — o serviço não os recebe (spec, Assumptions).

---

## D12 — Estratégia de teste em quatro camadas

**Decisão**:
1. **Domínio puro** — máquina de estados da transação, regra de expiração,
   classificação de motivo. Sem banco, sem rede, relógio injetado.
2. **Caso de uso com portas falsas** — cobrança única, ordem gravar→publicar→marcar,
   desfecho indeterminado, republicação de anúncio pendente.
3. **Contrato HTTP** — a consulta sobre `httptest`, com token forjado por chave de
   teste: 200, 400, 401 e 404 para terceiro e para inexistente.
4. **Integração com Testcontainers** — Postgres e RabbitMQ reais: corrida de
   entregas simultâneas (SC-002), reentrega após queda entre gravar e publicar
   (SC-003), esgotamento de entregas até a fila morta (SC-006), rajada com teto de
   concorrência (SC-004).

**Motivo**: o princípio II obriga teste automatizado do domínio e de cada operação
exposta — incluindo as expostas por evento. As camadas 1 a 3 rodam sem
infraestrutura; a 4 é a única que precisa de Docker, e é onde as garantias de
concorrência e reentrega só podem ser provadas de verdade.
