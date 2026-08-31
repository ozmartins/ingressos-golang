# Feature Specification: Processamento Assíncrono de Pagamentos (Servico-Pagamento)

**Feature Branch**: `001-pagamento-assincrono`

**Created**: 2026-08-30

**Status**: Draft

**Input**: User description: "Sigas as instruções do arquico ers-pagamento.md" (ERS do microsserviço `Servico-Pagamento`, arquivo `pagamento/erp-pagamentp.md`)

## Clarifications

### Session 2026-08-30

- Q: Qual o escopo da integração com o meio de pagamento nesta feature? → A: Apenas adquirente simulado, atrás de uma porta de domínio, com comportamento controlável nos testes; a integração com adquirente real fica fora do escopo
- Q: O que acontece quando o meio de pagamento não responde dentro do prazo? → A: A transação vai para um estado próprio de verificação pendente, nenhum resultado é anunciado, nenhuma nova cobrança é tentada e a intenção vai para a quarentena inspecionável
- Q: Como o anúncio do resultado é garantido se o serviço falhar entre gravar a transação e publicar? → A: A transação registra se o resultado já foi anunciado; a intenção só é confirmada depois da publicação, e uma reentrega cujo anúncio esteja pendente republica o resultado em vez de ignorá-la, sem nova cobrança
- Q: Quem pode consultar o pagamento de uma reserva? → A: Somente a pessoa dona da reserva; qualquer outra recebe a mesma resposta de inexistente, sem perfil administrativo
- Q: O que fazer quando a reserva já expirou no momento em que a cobrança sairia da fila? → A: Verificar o prazo antes de cobrar; se passou, não cobra, a transação é cancelada com motivo de expiração e o sistema anuncia pagamento recusado com esse motivo

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cobrar automaticamente uma reserva anunciada (Priority: P1)

Assim que uma pessoa reserva poltronas, o serviço de estoque anuncia a reserva criada. O serviço de pagamento reage a esse anúncio por conta própria, sem que ninguém espere por ele: registra a intenção de cobrança como em processamento, executa a cobrança na forma de pagamento escolhida e anuncia o resultado — aprovado ou recusado — para quem precisar reagir. A pessoa continua navegando enquanto isso acontece.

**Why this priority**: É a razão de existir do serviço. Sem essa reação automática, uma reserva nunca vira venda: o estoque não recebe a confirmação, a reserva expira e a poltrona volta a ficar livre. Entrega valor sozinha, mesmo antes de existir qualquer consulta de status.

**Independent Test**: Anunciar uma reserva criada e verificar que uma transação é registrada em processamento, que a cobrança é tentada uma única vez, que a transação termina em estado final e que exatamente um anúncio de resultado é publicado com os dados da transação.

**Acceptance Scenarios**:

1. **Given** um anúncio de reserva criada com reserva, pessoa, valor e forma de pagamento válidos, **When** o serviço o recebe, **Then** ele registra uma transação em processamento antes de qualquer tentativa de cobrança.
2. **Given** uma transação em processamento cuja cobrança é aprovada pelo meio de pagamento, **When** o resultado chega, **Then** a transação passa a paga, guarda a referência devolvida pelo meio de pagamento e o instante do pagamento, e o sistema anuncia o pagamento aprovado contendo transação, reserva, pessoa, valor e instante do pagamento.
3. **Given** uma transação em processamento cuja cobrança é recusada pelo meio de pagamento, **When** o resultado chega, **Then** a transação passa a recusada, guarda o motivo da recusa, e o sistema anuncia o pagamento recusado contendo transação, reserva, pessoa e motivo.
4. **Given** um anúncio de reserva criada em que falta um dado obrigatório, o valor não é positivo ou a forma de pagamento não é reconhecida, **When** ele é recebido, **Then** nenhuma cobrança é tentada, o anúncio é retirado de circulação para inspeção humana e a ocorrência fica registrada de forma auditável.
5. **Given** um anúncio de reserva criada cujo prazo de expiração já passou no momento em que a cobrança seria executada, **When** ele é processado, **Then** nenhuma cobrança é tentada, a transação é registrada como cancelada com o motivo de reserva expirada e o sistema anuncia o pagamento recusado com esse motivo.
6. **Given** uma cobrança em curso, **When** o meio de pagamento não responde dentro do prazo aceitável, **Then** o sistema encerra a espera, marca a transação como pendente de verificação, não anuncia resultado nenhum, não tenta nova cobrança e encaminha a intenção para a quarentena inspecionável.

---

### User Story 2 - Nunca cobrar a mesma reserva duas vezes (Priority: P1)

O mesmo anúncio de reserva criada pode chegar mais de uma vez — por reentrega do canal de eventos, por reprocessamento após uma falha ou por um reenvio manual. A pessoa MUST ser cobrada uma única vez por reserva, aconteça o que acontecer com a entrega das mensagens.

**Why this priority**: Cobrança duplicada é o defeito mais caro do domínio: envolve dinheiro real de terceiros, é percebida imediatamente pela pessoa e não se conserta sozinha. É indissociável do P1 de cobrança — entregar a cobrança sem essa garantia é entregar um defeito.

**Independent Test**: Anunciar a mesma reserva criada várias vezes, em sequência e em paralelo, e verificar que existe exatamente uma transação para aquela reserva, que a cobrança foi tentada uma única vez e que os anúncios repetidos são considerados processados sem efeito adicional.

**Acceptance Scenarios**:

1. **Given** uma reserva já processada até um estado final, **When** o anúncio da mesma reserva chega de novo, **Then** nenhuma nova cobrança é tentada, a transação existente permanece inalterada e o anúncio é considerado processado com sucesso.
2. **Given** uma reserva cuja transação ainda está em processamento, **When** o anúncio da mesma reserva chega de novo, **Then** nenhuma cobrança concorrente é iniciada e a segunda entrega não cria uma segunda transação.
3. **Given** duas entregas simultâneas do anúncio da mesma reserva, **When** ambas são processadas ao mesmo tempo, **Then** exatamente uma cobrança é tentada e apenas uma transação passa a existir para aquela reserva.
4. **Given** uma reserva já processada cujo resultado já foi anunciado, **When** o anúncio repetido chega, **Then** o sistema não republica o anúncio de resultado, evitando que os demais serviços reajam duas vezes ao mesmo pagamento.
5. **Given** uma reserva já processada até um estado final cujo resultado ainda não chegou a ser anunciado, **When** o anúncio repetido chega, **Then** o sistema publica o resultado a partir da transação já gravada, sem tentar nova cobrança.

---

### User Story 3 - Consultar o andamento do pagamento de uma reserva (Priority: P2)

Depois de reservar, a pessoa fica no aplicativo esperando saber se o pagamento passou. O aplicativo consulta o andamento do pagamento daquela reserva quantas vezes precisar e recebe o estado atual — em processamento, pago, recusado ou cancelado — junto com os dados da transação.

**Why this priority**: É o que fecha a experiência para a pessoa; sem isso, ela reserva e fica sem retorno. Mas depende da existência da cobrança (P1) para ter algo a informar, e o valor de negócio já existe sem ela — o estoque e a notificação reagem por evento, não por consulta.

**Independent Test**: Processar uma reserva até cada um dos estados possíveis e verificar que a consulta por reserva devolve o estado corrente correto, e que uma reserva desconhecida ou pertencente a outra pessoa não devolve dados.

**Acceptance Scenarios**:

1. **Given** uma reserva com transação registrada e uma pessoa autenticada que é a dona daquela reserva, **When** ela consulta o pagamento pela reserva, **Then** o sistema devolve transação, reserva, estado atual, valor, forma de pagamento e instante de criação.
2. **Given** uma transação ainda em processamento, **When** o pagamento é consultado, **Then** o sistema devolve o estado de processamento em vez de erro ou ausência.
3. **Given** uma reserva para a qual nenhuma transação existe, **When** o pagamento é consultado, **Then** o sistema informa que não há pagamento para aquela reserva, de forma distinguível de uma falha do serviço.
4. **Given** uma pessoa autenticada que não é a dona da reserva consultada, **When** ela consulta o pagamento, **Then** o sistema recusa o acesso e não revela nenhum dado da transação.
5. **Given** uma requisição sem credencial válida ou com credencial expirada, **When** o pagamento é consultado, **Then** o sistema recusa a requisição antes de qualquer leitura de dados.
6. **Given** um identificador de reserva mal formado, **When** o pagamento é consultado, **Then** o sistema recusa a requisição como inválida, de forma distinguível de reserva inexistente.

---

### User Story 4 - Absorver picos e sobreviver a falhas de infraestrutura (Priority: P2)

Nos horários de estreia, milhares de reservas são criadas em poucos minutos, enquanto o meio de pagamento aceita um número limitado de cobranças por vez. O serviço acumula as intenções pendentes e as processa no ritmo que o meio de pagamento suporta, sem perder nenhuma e sem derrubar o restante do sistema. Quando algo de infraestrutura falha no meio do caminho, a intenção volta para a fila em vez de sumir.

**Why this priority**: Nivelar a carga é o motivo declarado de o pagamento ser assíncrono. Sem isso, o pico derruba a integração e reservas pagas se perdem em silêncio. Fica atrás dos P1 porque só se manifesta sob volume: a cobrança correta de uma reserva já entrega valor antes de existir controle de vazão.

**Independent Test**: Anunciar uma rajada de reservas muito acima da capacidade de cobrança e verificar que nenhuma se perde, que o número de cobranças simultâneas nunca ultrapassa o limite vigente, e que intenções cujo processamento falha por infraestrutura voltam a ser processadas depois.

**Acceptance Scenarios**:

1. **Given** uma rajada de anúncios de reserva criada muito acima da vazão de cobrança, **When** eles chegam, **Then** todos acabam processados, nenhum é perdido e o número de cobranças em andamento ao mesmo tempo nunca ultrapassa o limite configurado.
2. **Given** uma falha de persistência durante o processamento de um anúncio, **When** ela ocorre, **Then** o anúncio é devolvido para a fila para nova tentativa e nenhum resultado é anunciado para aquela reserva.
3. **Given** um anúncio cujo processamento falha repetidamente, **When** o limite de tentativas é atingido, **Then** ele deixa de ser reentregue, é encaminhado para uma área de quarentena inspecionável e a ocorrência fica registrada de forma auditável.
4. **Given** o canal de eventos indisponível no momento de anunciar um resultado, **When** o serviço tenta publicá-lo, **Then** a intenção não é confirmada como processada, a transação já gravada permanece com o anúncio pendente, e o resultado é publicado na reentrega seguinte sem nova cobrança.
5. **Given** o serviço reiniciado no meio de uma rajada, **When** ele volta, **Then** ele retoma o consumo das intenções pendentes sem intervenção manual.

---

### Edge Cases

- O anúncio de reserva criada traz uma forma de pagamento válida, mas o meio de pagamento correspondente está fora do ar: a intenção não pode ser silenciosamente perdida nem cobrada duas vezes quando o meio voltar.
- A cobrança é aprovada pelo meio de pagamento, mas o serviço falha antes de registrar o resultado: a transação fica em processamento sem que ninguém saiba se houve cobrança. O sistema não cobra de novo; a intenção volta para a fila e, persistindo, vai para a quarentena — é dinheiro possivelmente debitado sem poltrona confirmada, e por isso precisa chegar a um humano.
- O anúncio de resultado é publicado, mas o serviço falha antes de registrar que o publicou: a republicação no reprocessamento é aceitável e esperada, desde que os consumidores possam tratá-la como repetição da mesma transação.
- Uma reserva expira no estoque enquanto a cobrança está em curso e é aprovada logo depois: o pagamento fica aprovado para poltronas já liberadas, e essa divergência precisa ser visível.
- O valor anunciado diverge do valor esperado para a reserva (por exemplo, valor zero ou negativo).
- A publicação do resultado falha repetidamente e a intenção esgota o limite de tentativas: ela vai para a quarentena com a transação já gravada e o anúncio pendente, e a divergência só se resolve por inspeção — nenhum consumidor reage àquele pagamento até lá.
- A mesma pessoa tem várias reservas em processamento ao mesmo tempo; a consulta por reserva não pode misturar transações.

## Requirements *(mandatory)*

### Functional Requirements

**Consumo da intenção de compra**

- **FR-001**: O sistema MUST reagir ao anúncio de reserva criada iniciando o processamento de cobrança, sem que o originador do anúncio precise esperar pelo resultado.
- **FR-002**: O sistema MUST registrar a transação em estado de processamento, de forma durável, antes de tentar qualquer cobrança.
- **FR-003**: O sistema MUST recusar como inválido, sem tentar cobrança, todo anúncio a que falte reserva, pessoa, valor ou forma de pagamento, cujo valor não seja positivo, ou cuja forma de pagamento não seja uma das reconhecidas.
- **FR-004**: O sistema MUST reconhecer as formas de pagamento de chave instantânea e de cartão de crédito, e MUST tratar qualquer outra como inválida.
- **FR-005**: O sistema MUST tratar como não cobrável a reserva cujo prazo de expiração já tenha passado no momento em que a cobrança seria executada, registrando-a como cancelada com o motivo de expiração.

**Idempotência**

- **FR-006**: O sistema MUST garantir no máximo uma transação por reserva, mesmo diante de entregas repetidas ou simultâneas do mesmo anúncio.
- **FR-007**: O sistema MUST verificar se a reserva já foi processada antes de executar qualquer cobrança, e MUST considerar o anúncio repetido como processado com sucesso, sem nova cobrança e sem alterar a transação existente. O resultado MUST NOT ser republicado, exceto quando ainda constar como não anunciado (FR-014).
- **FR-008**: O sistema MUST NOT emitir uma segunda cobrança para a mesma reserva em nenhuma circunstância de reprocessamento. Quando uma reentrega encontrar a transação ainda em processamento — situação em que não se sabe se a cobrança chegou a ser emitida —, o sistema MUST NOT chamar o meio de pagamento de novo, MUST devolver a intenção para nova tentativa, e MUST deixá-la seguir para a quarentena se a situação persistir até o limite de entregas.

**Resultado e anúncio**

- **FR-009**: O sistema MUST levar toda transação a um estado final — paga, recusada, cancelada ou pendente de verificação — e MUST NOT deixá-la em processamento indefinidamente.
- **FR-010**: O sistema MUST anunciar exatamente um resultado por transação que chegue a estado final anunciável: pagamento aprovado, com transação, reserva, pessoa, valor e instante do pagamento; ou pagamento recusado, com transação, reserva, pessoa e motivo. A transação pendente de verificação NÃO é anunciável e MUST NOT gerar anúncio de resultado.
- **FR-011**: O sistema MUST registrar, na transação recusada, um motivo legível que permita distinguir recusa do meio de pagamento, dado inválido e expiração da reserva.
- **FR-012**: O sistema MUST registrar, na transação paga, a referência devolvida pelo meio de pagamento e o instante em que o pagamento foi confirmado.
- **FR-013**: O sistema MUST publicar os anúncios de resultado de forma que qualquer serviço interessado possa consumi-los sem que o pagamento conheça seus consumidores.
- **FR-014**: O sistema MUST registrar, na própria transação, se o resultado dela já foi anunciado, e MUST NOT confirmar a intenção como processada antes de o anúncio ter sido publicado. Ao reprocessar uma intenção cuja transação já esteja em estado final anunciável com anúncio pendente, o sistema MUST publicar o resultado a partir da transação gravada, sem tentar nova cobrança.

**Consulta de andamento**

- **FR-015**: Uma pessoa autenticada MUST poder consultar, pelo identificador da reserva, o andamento do pagamento correspondente, obtendo transação, reserva, estado atual, valor, forma de pagamento e instante de criação.
- **FR-016**: O sistema MUST recusar toda consulta sem credencial válida, antes de qualquer leitura de dados de transação.
- **FR-017**: O sistema MUST impedir que uma pessoa obtenha dados de transação de reserva que não lhe pertence, e MUST NOT revelar pela resposta se tal reserva existe: a resposta para reserva de terceiro MUST ser indistinguível da resposta para reserva inexistente. O serviço MUST NOT reconhecer perfil administrativo ou de suporte que contorne essa regra.
- **FR-018**: O sistema MUST distinguir, na consulta, reserva sem transação, identificador mal formado e falha do próprio serviço.

**Vazão, tolerância a falha e observabilidade**

- **FR-019**: O sistema MUST limitar o número de cobranças processadas ao mesmo tempo a um teto configurável, para respeitar o limite de taxa do meio de pagamento.
- **FR-020**: O sistema MUST devolver o anúncio para nova tentativa, sem perdê-lo, quando o processamento falhar por indisponibilidade de infraestrutura.
- **FR-021**: O sistema MUST parar de reentregar um anúncio após um número configurável de tentativas malsucedidas, com padrão de três, e MUST encaminhá-lo para uma área de quarentena inspecionável em vez de descartá-lo.
- **FR-022**: O sistema MUST encerrar a espera por resposta do meio de pagamento após um prazo configurável e, nesse caso, MUST marcar a transação como pendente de verificação, MUST NOT anunciar resultado, MUST NOT tentar nova cobrança para aquela reserva e MUST encaminhar a intenção para a quarentena inspecionável.
- **FR-023**: O sistema MUST preservar, em cada transação, o instante de criação e o instante da última alteração, atualizando o segundo a cada escrita, e MUST manter a transação disponível para consulta depois de atingida a condição final. Nenhuma operação de listagem de histórico é exposta nesta feature.
- **FR-024**: O sistema MUST registrar de forma auditável, sem expor dados sensíveis de meio de pagamento, toda recusa, toda quarentena e toda divergência detectada.
- **FR-025**: O sistema MUST expor uma verificação de saúde que distinga o serviço no ar do serviço capaz de processar cobranças.

### Key Entities

- **Transação de Pagamento**: a tentativa de cobrança de uma reserva. Tem identidade própria, referencia exatamente uma reserva e uma pessoa, e guarda valor total, forma de pagamento, estado atual, referência devolvida pelo meio de pagamento, motivo da falha quando houver, o indicador de que o resultado já foi anunciado, e os instantes de criação e de última alteração. A reserva é única entre todas as transações — é ela que sustenta a garantia de cobrança única.
- **Estado da Transação**: em processamento (registrada, cobrança ainda não concluída), paga (cobrança aprovada), recusada (cobrança negada ou dado inválido), cancelada (não cobrável, por exemplo por reserva expirada) e pendente de verificação (o meio de pagamento não respondeu a tempo e não se sabe se a cobrança foi efetivada). Os quatro últimos são finais; os três primeiros desses são anunciáveis, e o pendente de verificação nunca gera anúncio, exigindo inspeção humana.
- **Intenção de Compra**: o anúncio de reserva criada consumido pelo serviço. Traz reserva, sessão, pessoa, poltronas, valor total, forma de pagamento e prazo de expiração da reserva.
- **Resultado de Pagamento**: o anúncio publicado ao fim do processamento, em duas variantes — aprovado e recusado —, cada uma com o conteúdo mínimo que os demais serviços precisam para reagir sem consultar o pagamento.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% das reservas anunciadas terminam em estado final. Toda transação em estado anunciável tem seu resultado anunciado, e nenhuma transação permanece em processamento por mais de um minuto após a resposta do meio de pagamento.
- **SC-002**: Zero cobranças duplicadas: reenviando o mesmo anúncio de reserva 100 vezes, em sequência e em paralelo, existe exatamente uma transação e uma única cobrança para aquela reserva.
- **SC-003**: Zero anúncios de resultado duplicados para uma mesma transação em condições normais de operação, e zero resultados perdidos: matando o serviço entre a gravação da transação e a publicação, em 100% dos casos o resultado é anunciado no reprocessamento, sem nova cobrança.
- **SC-004**: Uma rajada de 1.000 reservas criadas em um minuto é integralmente processada sem perda, mantendo o número de cobranças simultâneas dentro do teto configurado durante todo o pico.
- **SC-005**: 95% das consultas de andamento respondem em menos de um segundo sob a carga de pico descrita em SC-004.
- **SC-006**: Zero intenções perdidas por falha de infraestrutura: toda intenção cujo processamento falhar é reprocessada ou fica retida em quarentena inspecionável, nunca descartada em silêncio.
- **SC-007**: Nenhuma consulta devolve dados de transação a quem não é dono da reserva, verificado por teste automatizado para cada estado possível da transação.
- **SC-008**: Uma pessoa descobre o desfecho do seu pagamento em menos de 30 segundos após a criação da reserva, em operação normal.
- **SC-009**: Zero transações pendentes de verificação sem registro correspondente na quarentena: toda ausência de resposta do meio de pagamento é inspecionável, e nenhuma delas gera anúncio de resultado ou segunda cobrança.

## Assumptions

- O meio de pagamento externo (adquirente/gateway) é uma dependência de terceiros acessada por trás de um contrato próprio do domínio. Conforme decidido nas clarificações, esta entrega o atende apenas com uma implementação simulada, cujo comportamento observável — aprovação, recusa, demora e indisponibilidade — é controlável nos testes. A integração com adquirente real está fora do escopo desta feature e, quando entrar, não deve alterar nenhum requisito desta especificação.
- A identidade da pessoa vem de um provedor de identidade já existente no ecossistema, e o pagamento apenas valida a credencial apresentada, sem manter sessão nem cadastro próprio de pessoas.
- O anúncio de reserva criada é produzido pelo serviço de estoque, com o conteúdo descrito na ERS, e é a única entrada que dispara cobrança. O pagamento não expõe operação para iniciar cobrança sob demanda.
- Os anúncios de resultado são consumidos pelo estoque (para confirmar ou liberar as poltronas) e pela notificação (para avisar a pessoa). O pagamento não conhece esses consumidores nem depende deles.
- O valor a cobrar é o que vem no anúncio. O pagamento não recalcula preço nem consulta o catálogo; validar o valor contra a sessão é responsabilidade de quem originou a reserva.
- Estorno, cancelamento a pedido da pessoa, reembolso e pagamento parcial estão fora do escopo desta feature. O estado de cancelada é usado apenas para reserva não cobrável.
- A resolução de uma transação pendente de verificação é operacional e manual, a partir da quarentena; automatizar essa reconciliação (consultando o meio de pagamento pelo estado da cobrança) está fora do escopo desta feature.
- Uma transação por reserva: a ERS não prevê nova tentativa de pagamento sobre a mesma reserva após uma recusa. Uma nova tentativa exigiria uma nova reserva.
- Dados sensíveis de meio de pagamento (número de cartão e equivalentes) não trafegam nem são armazenados por este serviço; ele lida apenas com valor, forma de pagamento e referências devolvidas pelo meio de pagamento.
- O teto padrão de cobranças simultâneas é 10, e o de tentativas antes da quarentena é 3, conforme a ERS; ambos são configuráveis por ambiente.
