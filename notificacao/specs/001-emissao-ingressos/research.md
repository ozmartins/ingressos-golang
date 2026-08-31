# Research: Emissão e Validação de Ingressos Digitais (Servico-Notificacao)

**Fase 0** | **Data**: 2026-08-30 | **Spec**: [spec.md](./spec.md)

Cada decisão abaixo resolve uma incógnita do Technical Context ou fixa um ponto que
a spec deixou para o plano. O formato é fixo: o que foi decidido, por que, e o que
foi rejeitado — o registro que o princípio I da constituição exige quando a solução
mais simples é a escolhida.

**Verificação prévia (princípio III)**: as afirmações deste documento sobre o que
os outros serviços fazem foram lidas no código, não nas specs deles. Os arquivos
consultados estão citados por caminho e o que foi lido está transcrito.

---

## D1 — O contrato real de `pagamento.sucesso`

**Decisão**: consumir o fato exatamente como o `Servico-Pagamento` o publica hoje,
tolerando campos que não interessam a este serviço.

**Verificado no código**, não na ERS: `pagamento/internal/usecase/fatos.go` monta
`FatoPagamentoSucesso` com oito campos — `evento`, `versao`, `ocorrido_em`,
`transacao_id`, `reserva_id`, `usuario_id`, `valor_total`, `pago_em` — e publica com
a chave de roteamento `pagamento.sucesso` (constante `RoutingKeySucesso`), no
exchange `cinema.eventos` (`pagamento/internal/platform/config/config.go`, padrão de
`AMQP_EXCHANGE`).

A ERS deste serviço descreve o payload com seis campos: os oito menos `versao` e
`ocorrido_em`. **O que o código publica é um superconjunto do que a ERS espera** —
logo não há divergência a levar ao princípio IV. O que a ERS descreve chega; chegam
também dois campos a mais, que este serviço ignora.

Dos seis campos que a ERS descreve, este serviço usa **quatro**: `reserva_id`,
`usuario_id`, `transacao_id` e `pago_em`. `valor_total` e `evento` chegam e não são
guardados — nenhum requisito depende deles, e o modelo de dados não tem onde pô-los.

**Rationale**: o consumidor precisa apenas do que usa. Exigir presença de campo que
não se usa transformaria uma mudança inofensiva do produtor em quarentena em massa.

**Alternativas rejeitadas**:
- *Validar o payload inteiro contra um esquema fechado, recusando campo desconhecido*:
  rejeitada porque quebra o serviço a cada campo novo que o pagamento acrescente, sem
  proteger nada — os campos que importam continuam sendo validados um a um (FR-002).
- *Exigir `versao == 1`*: rejeitada por antecipação de requisito. Não há segunda versão,
  e o dia em que houver é o dia de decidir o que fazer com ela.

---

## D2 — Emissão única: a restrição de unicidade é o mecanismo

**Decisão**: a garantia de um ingresso por reserva (FR-004) mora na restrição
`UNIQUE (reserva_id)` da tabela. A emissão é um
`INSERT ... ON CONFLICT (reserva_id) DO NOTHING RETURNING *`; devolver zero linhas
significa "outra entrega chegou primeiro", e essa entrega não emite nem avisa.

**Rationale**: a linha do ingresso **é** o registro de "esta reserva já foi
processada". Não existe janela entre verificar e inserir, porque não há verificação
separada: quem decide é o banco, atomicamente, e isso vale igualmente para reentrega
sequencial e para duas entregas simultâneas (FR-004, cenários 2 e 3 da User Story 1).
É o mesmo mecanismo já em produção no `Servico-Pagamento`
(`pagamento/internal/usecase/ports.go`, `Repositorio.CriarSeAusente`), lido no código.

**Alternativas rejeitadas**:
- *Tabela de mensagens já processadas*: rejeitada porque duplica em uma segunda tabela
  a informação que a primeira já carrega, e cria a necessidade de mantê-las coerentes.
- *`SELECT` antes do `INSERT`*: rejeitada por ser incorreta sob concorrência — é
  exatamente a janela que o cenário 3 da User Story 1 exercita.
- *Trava distribuída por reserva*: rejeitada por acrescentar um componente (e um modo
  de falha) para resolver um problema que uma restrição de coluna já resolve.

---

## D3 — Forma do código de acesso: assinatura sobre o identificador do ingresso

**Decisão**: `codigo_qr` tem três partes separadas por ponto:

```
CIN1.<id do ingresso em base64url>.<HMAC-SHA256 em base64url>
```

O HMAC é calculado sobre as duas primeiras partes juntas (`CIN1.<id>`), com um segredo
lido do ambiente (`INGRESSO_QR_SEGREDO`). A comparação da assinatura é feita em tempo
constante. O resultado tem cerca de 71 caracteres e cabe folgado no `VARCHAR(255)` da ERS.

**Rationale**, requisito a requisito:
- **FR-005** (nada sensível em texto legível): a única coisa transportada é o
  identificador do próprio ingresso, que é opaco e gerado aleatoriamente. Não vai ali
  nem a pessoa, nem a reserva, nem o valor, nem a sessão.
- **FR-005** (não adivinhável): sem o segredo, produzir uma assinatura válida para um
  identificador qualquer é inviável — é a propriedade do HMAC, não uma escolha de
  formato.
- **FR-006 / FR-010** (autenticidade verificável, e recusa barata do malformado): o
  prefixo fixo `CIN1` e a forma de três partes permitem descartar lixo antes de
  qualquer trabalho criptográfico, e a assinatura permite recusar um código forjado
  **sem consultar o banco**. Só um código com assinatura válida chega a virar consulta.

O prefixo `CIN1` é marcador de formato, e está aqui por esse motivo — **não** é um
mecanismo de rotação de chave. Rotação está declarada fora de escopo na spec, e nada
neste desenho a implementa.

**Alternativas rejeitadas**:
- *Código aleatório opaco, sem assinatura, verificado só por busca no banco*: mais
  simples de gerar, e foi seriamente considerada. Rejeitada porque a spec pede
  verificação de autenticidade sem consulta ao acervo (FR-006 e a suposição
  correspondente), e porque transforma toda tentativa de força bruta em carga no banco.
- *JWT como conteúdo do QR*: rejeitada por tamanho (centenas de caracteres, pior para
  ler em tela de celular arranhada) e por trazer junto um vocabulário de validade,
  emissor e público que este código não tem e não deveria sugerir que tem.
- *Assinar a reserva em vez do ingresso*: rejeitada porque coloca no código um dado do
  domínio da compra, contra a FR-005.
- *Guardar apenas o HMAC e derivar o resto*: rejeitada porque a ERS e a FR-013 exigem
  devolver o código na listagem; guardá-lo inteiro é o que torna isso possível sem
  recalcular.

---

## D4 — A baixa do ingresso é uma escrita condicionada, não uma leitura seguida de escrita

**Decisão**: validar é
`UPDATE ingressos_emitidos SET status='UTILIZADO', utilizado_em=$1 WHERE id=$2 AND status='VALIDO'`.
Uma linha afetada é a autorização; zero linhas afetadas significa que o ingresso não
estava válido, e só então se lê a linha para dizer **por quê** (já utilizado, cancelado
ou inexistente).

**Rationale**: é o que satisfaz a FR-011 sem trava explícita — duas leituras
simultâneas do mesmo código disputam a mesma linha, e o banco garante que só uma vê a
transição acontecer. A leitura posterior existe apenas para escolher a mensagem, e
acontece fora do caminho de sucesso.

**Alternativas rejeitadas**:
- *`SELECT` e depois `UPDATE`*: rejeitada por ser a própria condição de corrida que o
  cenário 5 da User Story 2 exercita — as duas leituras veriam `VALIDO`.
- *`SELECT ... FOR UPDATE`*: correta, mas segura a linha por mais tempo e acrescenta
  uma ida ao banco para resolver o que uma escrita condicionada já resolve.
- *Serializar as validações em uma fila*: rejeitada de saída; a portaria precisa de
  resposta em menos de um segundo (SC-003), e a concorrência aqui é o caso normal.

---

## D5 — Reentrega e quarentena são do broker, não do serviço

**Decisão**: fila do tipo `quorum` com `x-delivery-limit`, ligada a um exchange de
mensagens mortas. A classificação da FR-022 vira dois gestos no consumidor:

| Situação | Gesto | Efeito |
|---|---|---|
| Anúncio ilegível ou malformado (FR-002) | `Nack(requeue=false)` | vai direto para a quarentena, sem retentativa |
| Falha transitória (banco fora, timeout) | `Nack(requeue=true)` após breve espera | volta para a fila; o broker conta a entrega |
| Sucesso (inclusive reentrega idempotente) | `Ack` | some da fila |

Esgotado o limite de entregas, o próprio broker encaminha para a fila morta — o
serviço não conta tentativa nenhuma.

**Tradução de vocabulário, medida e não suposta**: o `Servico-Pagamento` já enfrentou
isto e deixou o achado registrado em `pagamento/internal/adapter/amqp/topologia.go`: o
RabbitMQ trata `x-delivery-limit` como número de **reentregas**, entregando `limite+1`
vezes. Como a FR-022 fala em **tentativas**, o valor declarado é
`AMQP_LIMITE_ENTREGAS - 1`. Reaproveito a tradução e o teste que a sustenta.

**Rationale**: contar tentativa no serviço exige guardar contador em algum lugar, e
esse lugar seria uma tabela nova ou um cabeçalho reescrito a cada volta. O broker já
conta, de forma durável e sem estado do lado de cá.

**Alternativas rejeitadas**:
- *Contador em coluna própria*: rejeitada por acrescentar escrita no caminho de falha,
  justamente quando o banco pode ser a coisa que está falhando.
- *Reentrega indefinida (opção C da clarificação 3)*: descartada pelo mantenedor.
- *Retentativa com espera crescente dentro do processo*: rejeitada porque segura a
  entrega e o prefetch enquanto espera, reduzindo a vazão para defender uma dependência
  que o broker já protege ao devolver a mensagem.

---

## D6 — O aviso sai no mesmo processamento e a falha morre ali

**Decisão** (clarificação 5): ordem fixa dentro do consumo — gravar o ingresso →
disparar o aviso → gravar o registro do aviso → confirmar a mensagem. O erro do canal é
capturado, vira um registro com estado `FALHA`, e **não** é devolvido ao consumidor:
para o broker, a entrega foi um sucesso (FR-025).

Quando a emissão não aconteceu por já existir ingresso para a reserva (D2), o aviso
**não** é disparado de novo. A reentrega é inteiramente inerte.

**Rationale**: um único caminho de execução, sem segundo mecanismo. A garantia da
FR-018 vira uma regra local — "este erro não sobe" — que se testa com uma porta falsa
que devolve erro, em teste sem rede.

O caso da queda entre gravar o ingresso e gravar o registro está tratado na spec, em
Edge Cases: o ingresso permanece, o registro ausente é a evidência de quem ficou sem
aviso, e a reentrega não repara isso. É comportamento aceito, não descuido.

**Alternativas rejeitadas**:
- *Rotina de fundo varrendo ingressos sem registro de aviso (opção B da clarificação 5)*:
  descartada pelo mantenedor. Seria um segundo mecanismo de execução — mais partes
  móveis para uma garantia que a spec não pede.
- *Caixa de saída transacional*: rejeitada pelo mesmo motivo, com custo ainda maior.
- *Disparar antes de gravar (opção C)*: descartada pelo mantenedor; avisaria sobre um
  ingresso que pode não chegar a existir.

---

## D7 — Duas credenciais, cada rota aceitando só a sua

**Decisão**: dois esquemas independentes, sem hierarquia entre eles.

- `GET /api/v1/ingressos/meus-ingressos` — JWT do Keycloak validado localmente por
  JWKS (assinatura, emissor, público, validade). A identidade é o `sub`, e é ela que
  recorta a listagem (FR-014).
- `POST /api/v1/ingressos/validar` — cabeçalho `X-API-Key` comparado em tempo constante
  com o segredo do ambiente. Sem token, sem `sub`, sem noção de pessoa.

Apresentar a credencial errada na rota errada é recusa, não promoção: a rota de
validação não olha `Authorization`, e a de listagem não olha `X-API-Key`. É o caso de
borda que a spec descreve.

**Rationale**: a validação de JWT por JWKS já existe no workspace, lida em
`pagamento/internal/adapter/http/auth.go` — mesma biblioteca, mesmas travas
(`WithExpirationRequired`, emissor e público exigidos, métodos restritos). Reaproveitar
o desenho verificado é mais barato e mais seguro do que inventar outro.

A comparação da chave da portaria é em tempo constante porque comparação ingênua de
string vaza o prefixo correto pelo tempo de resposta — e essa é uma chave estática,
que ninguém vai rotacionar tão cedo (rotação está fora de escopo).

**Alternativas rejeitadas**:
- *Um escopo dentro do próprio JWT para a portaria*: alinhado com o que a ERS chama de
  "chave de API / escopo específico", mas exigiria emitir e renovar token de máquina
  para cada catraca. Rejeitada por complexidade não pedida; a ERS aceita a chave de API,
  e o contrato dela é `X-API-Key`.
- *TLS mútuo por dispositivo*: rejeitada por ser gestão de certificado por catraca —
  fora de escopo por decisão explícita da spec.

---

## D8 — Listagem: tudo de uma vez, ordenada, com filtro opcional

**Decisão** (clarificação 4): `WHERE usuario_id = $1` mais, se informado,
`AND status = $2`; ordenação `criado_em DESC, id DESC`. Sem paginação. Estado não
reconhecido é `400`, não filtro ignorado (FR-024).

O `id DESC` existe só como desempate: dois ingressos podem nascer no mesmo instante, e
sem desempate a ordem entre eles varia entre execuções — o que tornaria o teste de
ordenação intermitente.

**Índice**: `(usuario_id, criado_em DESC)`. É o único índice além das chaves e das
restrições de unicidade, e serve exatamente a esta consulta, que é a única de leitura
por pessoa.

**Rationale**: o volume por pessoa é pequeno — SC-009 fixa o alvo verificável em 200
ingressos no histórico. Paginação para 200 linhas é maquinário sem carga.

**Alternativas rejeitadas**:
- *Paginação por cursor (opção C da clarificação)*: descartada pelo mantenedor.
- *Ordenar no cliente (opção A)*: descartada; a ordem passa a ser um requisito
  (FR-023), e requisito verificável mora no servidor.
- *Índice adicional por `status`*: rejeitado; com o recorte por pessoa já feito, o
  filtro por estado atua sobre dezenas de linhas.

---

## D9 — Relógio e gerador de identidade são portas

**Decisão**: `Relogio` e `GeradorID` como interfaces do pacote de casos de uso,
implementadas por adaptadores triviais sobre `time.Now` e `uuid.NewString`.

**Rationale**: `utilizado_em` e `criado_em` são observáveis do contrato (aparecem na
resposta da validação e na listagem) e são objeto de asserção nos testes. Sem relógio
injetado, a asserção vira tolerância a intervalo — que passa quando não deveria. É o
mesmo desenho já verificado em `pagamento/internal/usecase/ports.go`.

**Alternativa rejeitada**: *`time.Now()` direto no domínio* — rejeitada porque
contamina o núcleo com o relógio do sistema e viola a exigência do princípio II de que
o teste de domínio rode sem espera real.

---

## D10 — Duas tabelas, exatamente as da ERS

**Decisão**: `ingressos_emitidos` e `registros_notificacao`, com o DDL da ERS mais as
restrições `CHECK` que materializam invariantes do domínio no banco
(ver [data-model.md](./data-model.md) §1 e §5). Nenhuma coluna nova: ao contrário do
`Servico-Pagamento`, que precisou de `resultado_anunciado` para fechar a janela entre
gravar e publicar, aqui não há publicação a garantir — este serviço não emite fato
nenhum.

**Rationale**: o serviço é a ponta da cadeia. Nada reage a ele por evento, então não há
anúncio a rastrear, e a coluna que o pagamento precisou aqui não teria função.

**Alternativas rejeitadas**:
- *Publicar um fato `ingresso.emitido`*: rejeitada por não ter consumidor. Antecipação
  de requisito futuro não é necessidade demonstrada (princípio I).
- *Uma tabela só, com o aviso em colunas do ingresso*: rejeitada porque a ERS prevê
  vários registros de aviso por ingresso, e achatar isso perderia a trilha de tentativas.

---

## D11 — Configuração validada na largada, processo recusa subir se faltar

**Decisão**: mesmo desenho de `pagamento/internal/platform/config/config.go`, lido no
código: uma função que reúne **todas** as chaves problemáticas e falha uma vez, listando
todas. Obrigatórias sem padrão: `DATABASE_URL`, `AMQP_URL`, `JWKS_URL`, `JWT_ISSUER`,
`JWT_AUDIENCE`, `INGRESSO_QR_SEGREDO`, `PORTARIA_API_KEY`.

**Rationale**: subir sem o segredo do QR significa emitir ingressos que a portaria não
vai conseguir validar — dano silencioso e difícil de desfazer, porque os códigos já
foram para as mãos das pessoas. Falhar na largada é a alternativa barata.

**Alternativa rejeitada**: *gerar um segredo aleatório quando a variável faltar* —
rejeitada com veemência: cada reinício invalidaria todos os ingressos já emitidos.

---

## D12 — Quatro camadas de teste, e o que cada uma pode falhar

O princípio II exige teste de domínio sem banco/rede/servidor e teste de cada operação
exposta em todas as categorias de erro do contrato. As camadas:

1. **Domínio puro** (`internal/domain/ingresso`) — transições (FR-019), rejeição de
   saída de estado terminal, invariante de `utilizado_em`. Sem banco, sem rede, relógio
   injetado.
2. **Casos de uso com portas falsas** — idempotência (FR-004), classificação
   quarentena/reentrega (FR-022), e o teste que sustenta a FR-025: notificador que
   devolve erro, e a asserção de que o desfecho continua sendo `Ack`.
3. **Contrato HTTP sobre `httptest`** — as duas rotas, cada categoria de erro de
   `contracts/erros.md`: 400, 401, 403, 404, 409, 422. Inclui a asserção de que a rota
   errada recusa a credencial errada.
4. **Integração com Testcontainers** (Postgres e RabbitMQ reais) — o que só falha de
   verdade: duas entregas simultâneas da mesma reserva (SC-001), duas validações
   simultâneas do mesmo código (SC-004), esgotamento do limite até a fila morta
   (FR-022), e o percurso completo do anúncio ao ingresso listável.

**O que não é testado por regra**: adaptadores, fiação de composição e o gerador de
UUID. Testá-los é decisão de custo-benefício, como a constituição diz — não obrigação.
Percentual de linhas cobertas não é portão em lugar nenhum deste plano.

---

## D13 — Observabilidade: o código de acesso nunca vai para o log

**Decisão**: `log/slog` em JSON e OpenTelemetry, com o `traceparent` extraído dos
cabeçalhos AMQP para que o processamento continue o rastro do pagamento — mesmo
desenho de `pagamento/internal/adapter/amqp/consumidor.go`. A FR-021 acrescenta uma
regra dura: **`codigo_qr` não aparece em log, em atributo de rastro nem em mensagem de
erro**. O que identifica a operação no log é o `ingresso_id`.

**Rationale**: log é copiado, agregado e lido por muita gente. Um código de acesso em
log é um ingresso utilizável em log, e a assinatura não protege contra quem simplesmente
copia o código inteiro.

**Alternativa rejeitada**: *registrar um prefixo do código para facilitar depuração* —
rejeitada porque prefixo de código válido é exatamente o que um ataque de força bruta
quer, e a depuração já tem o `ingresso_id`.
