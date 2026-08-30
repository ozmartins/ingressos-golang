# Feature Specification: Bloqueio, Confirmação e Liberação de Poltronas (Servico-Estoque)

**Feature Branch**: `001-estoque-bloqueio-poltronas`

**Created**: 2026-08-29

**Status**: Draft

**Input**: User description: "Sigas as sintruções que constam no arquivo ers-estoque.md" (ERS do microsserviço `Servico-Estoque`)

## Clarifications

### Session 2026-08-29

- Q: Como as poltronas de uma sessão passam a existir no serviço de estoque? → A: O serviço consome um evento de sessão criada (com o layout da sala) e provisiona as poltronas automaticamente
- Q: Que identificador de poltrona o contrato de bloqueio e o evento de reserva criada usam? → A: Identificador determinístico derivado de sessão + fileira + número (ex.: "A1" no escopo da sessão), montado pelo cliente a partir do layout que já conhece
- Q: Como o canal síncrono de bloqueio é protegido? → A: Autenticação mútua no transporte (certificados de serviço); só chamadores com identidade de serviço válida são atendidos, e a identidade da pessoa usuária segue como parâmetro confiável
- Q: Como o estado atual das poltronas de uma sessão é consultado? → A: Segunda operação no mesmo canal síncrono, devolvendo o mapa de poltronas da sessão com o estado atual de cada uma
- Q: Há limite de poltronas por solicitação de bloqueio? → A: Limite configurável, com padrão de 10 poltronas por solicitação

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Bloquear poltronas de uma sessão (Priority: P1)

Uma pessoa escolheu uma sessão e um conjunto de poltronas no aplicativo. O serviço de catálogo repassa esse pedido ao serviço de estoque, que precisa decidir em milissegundos se aquelas poltronas específicas podem ser reservadas para aquela pessoa. Se todas estiverem livres, elas passam a ficar reservadas para ela por um prazo limitado, e o serviço devolve o identificador da reserva e o instante em que ela expira. Se qualquer uma já estiver tomada, nada é reservado e o pedido é recusado com a informação de indisponibilidade.

**Why this priority**: É a razão de existir do serviço. Sem o bloqueio atômico não há venda: é ele que impede que duas pessoas comprem a mesma poltrona e que dá à pessoa o tempo necessário para concluir o pagamento. Entrega valor sozinho, mesmo antes de qualquer integração com pagamento.

**Independent Test**: Provisionar a matriz de poltronas de uma sessão, solicitar o bloqueio de um subconjunto delas e verificar que a reserva é criada com prazo de expiração, que as poltronas passam a constar como reservadas e que uma segunda solicitação sobre qualquer uma delas é recusada.

**Acceptance Scenarios**:

1. **Given** uma sessão cujas poltronas solicitadas estão todas livres, **When** o serviço de catálogo solicita o bloqueio informando sessão, poltronas e identidade da pessoa, **Then** o sistema cria uma reserva pendente, marca todas as poltronas como reservadas e responde com sucesso, o identificador da reserva e o instante de expiração.
2. **Given** uma sessão em que ao menos uma das poltronas solicitadas já está reservada ou ocupada, **When** o bloqueio é solicitado, **Then** nenhuma poltrona do pedido muda de estado, nenhuma reserva é criada e a resposta indica insucesso com a causa de indisponibilidade.
3. **Given** duas solicitações simultâneas disputando a mesma poltrona, **When** ambas são processadas, **Then** exatamente uma obtém sucesso e a outra recebe indisponibilidade, sem que a poltrona fique atribuída a duas reservas.
4. **Given** uma solicitação cuja lista de poltronas está vazia, contém rótulos repetidos ou contém poltronas que não existem na sessão informada, **When** o bloqueio é solicitado, **Then** o sistema recusa a solicitação como inválida, de forma distinguível da indisponibilidade.
5. **Given** uma solicitação sem identidade da pessoa usuária, **When** o bloqueio é solicitado, **Then** o sistema recusa a solicitação como inválida e não cria reserva.
6. **Given** uma solicitação com mais poltronas do que o limite vigente por bloqueio, **When** o bloqueio é solicitado, **Then** o sistema recusa a solicitação como inválida, informa o limite e não altera o estado de nenhuma poltrona.
7. **Given** um chamador que não apresenta identidade de serviço válida, **When** ele tenta solicitar um bloqueio, **Then** a conexão é recusada antes de qualquer alteração de estado.
8. **Given** um bloqueio concluído com sucesso, **When** a resposta é devolvida, **Then** o sistema também anuncia o evento de reserva criada para os demais serviços interessados, contendo reserva, sessão, pessoa, poltronas e instante de expiração.

---

### User Story 2 - Confirmar a reserva quando o pagamento é aprovado (Priority: P1)

Depois que a pessoa paga, o serviço de pagamento anuncia o sucesso. O serviço de estoque reage a esse anúncio tornando a posse das poltronas definitiva: a reserva passa a confirmada e as poltronas passam a ocupadas, deixando de ser candidatas a expiração ou a novo bloqueio.

**Why this priority**: Sem a confirmação, toda reserva paga acabaria expirando e liberando poltronas já vendidas — o pior defeito possível do domínio. É indissociável do P1 de bloqueio.

**Independent Test**: Criar uma reserva pendente, anunciar o sucesso de pagamento correspondente e verificar que a reserva fica confirmada, que as poltronas ficam ocupadas, que elas não são mais liberadas pela expiração e que reprocessar o mesmo anúncio não altera nada.

**Acceptance Scenarios**:

1. **Given** uma reserva pendente dentro do prazo, **When** o sistema recebe o anúncio de pagamento aprovado para ela, **Then** a reserva passa a confirmada e todas as suas poltronas passam a ocupadas de forma definitiva.
2. **Given** uma reserva já confirmada, **When** o mesmo anúncio de pagamento aprovado é recebido novamente, **Then** o estado permanece inalterado e o anúncio é considerado processado com sucesso.
3. **Given** um anúncio de pagamento aprovado referente a uma reserva inexistente, **When** ele é recebido, **Then** o sistema não altera nenhum estado, registra a ocorrência de forma auditável e não reprocessa o anúncio indefinidamente.
4. **Given** uma reserva já expirada cujas poltronas foram liberadas e eventualmente retomadas por outra pessoa, **When** chega um anúncio de pagamento aprovado para ela, **Then** o sistema não sobrescreve o estado atual das poltronas e sinaliza a divergência de forma auditável.
5. **Given** uma reserva confirmada, **When** o prazo original de expiração é atingido, **Then** as poltronas permanecem ocupadas e não são liberadas.

---

### User Story 3 - Liberar poltronas quando o pagamento falha (Priority: P1)

Se o pagamento é recusado, as poltronas não podem ficar presas até o fim do prazo: o serviço de estoque reage ao anúncio de falha cancelando a reserva e devolvendo as poltronas ao estado livre, para que outra pessoa possa comprá-las imediatamente.

**Why this priority**: Fecha o ciclo de vida da reserva e protege a receita — poltronas presas por pagamentos recusados são assentos não vendidos. Testável e implantável de forma independente da confirmação.

**Independent Test**: Criar uma reserva pendente, anunciar a falha de pagamento correspondente e verificar que a reserva fica cancelada, que as poltronas voltam a livres, que elas podem ser bloqueadas por outra pessoa em seguida e que reprocessar o mesmo anúncio não produz efeito adicional.

**Acceptance Scenarios**:

1. **Given** uma reserva pendente, **When** o sistema recebe o anúncio de pagamento recusado para ela, **Then** a reserva passa a cancelada, todas as suas poltronas voltam a livres e o bloqueio de concorrência associado é liberado.
2. **Given** uma reserva cujas poltronas foram liberadas por falha de pagamento, **When** outra pessoa solicita o bloqueio dessas mesmas poltronas, **Then** o bloqueio é concedido normalmente.
3. **Given** uma reserva já cancelada, **When** o mesmo anúncio de falha é recebido novamente, **Then** o estado permanece inalterado e o anúncio é considerado processado com sucesso.
4. **Given** uma reserva já confirmada por pagamento aprovado, **When** chega um anúncio de falha para ela, **Then** as poltronas permanecem ocupadas e a divergência é registrada de forma auditável.

---

### User Story 4 - Liberar automaticamente reservas não pagas (Priority: P2)

Quando a pessoa abandona a compra e nenhum desfecho de pagamento chega, a reserva não pode prender as poltronas para sempre. Passado o prazo de 10 minutos, o sistema invalida a reserva por conta própria e devolve as poltronas ao estado livre, sem intervenção manual.

**Why this priority**: Sem expiração automática, o abandono de carrinho retiraria assentos do estoque de forma permanente. É P2 porque o sistema já entrega valor com bloqueio e desfechos de pagamento, mas degrada rapidamente em operação real sem esta regra.

**Independent Test**: Criar uma reserva, avançar o relógio além do prazo sem anunciar nenhum desfecho de pagamento e verificar que a reserva consta como expirada e que as poltronas foram devolvidas ao estado livre e podem ser bloqueadas novamente.

**Acceptance Scenarios**:

1. **Given** uma reserva pendente cujo prazo de 10 minutos se esgotou sem desfecho de pagamento, **When** o prazo é atingido, **Then** a reserva passa a expirada e todas as suas poltronas voltam a livres, sem qualquer ação humana.
2. **Given** uma reserva expirada, **When** outra pessoa solicita o bloqueio das mesmas poltronas, **Then** o bloqueio é concedido normalmente.
3. **Given** o serviço ficou indisponível durante o prazo de várias reservas, **When** ele volta a operar, **Then** todas as reservas cujo prazo venceu durante a indisponibilidade são invalidadas e suas poltronas liberadas.
4. **Given** uma reserva cujo prazo está a poucos instantes de vencer, **When** chega o anúncio de pagamento aprovado antes do vencimento, **Then** a reserva é confirmada e não é posteriormente expirada.

---

### User Story 5 - Provisionar a matriz de poltronas ao criar a sessão (Priority: P3)

Quando uma nova sessão é criada no catálogo, o fato é anunciado com o layout da sala. O serviço de estoque reage a esse anúncio criando automaticamente a matriz de poltronas daquela sessão — quais poltronas existem, em que fileira, com que número e de que tipo (normal, PCD, namoradeira) — todas no estado livre, sem qualquer ação humana. O estado atual dessa matriz é consultável a qualquer momento.

**Why this priority**: É pré-requisito de dados para as demais histórias, mas não é uma jornada de valor autônoma para a pessoa usuária final e, em uma primeira entrega, a matriz pode ser populada por carga pontual; por isso vem depois das jornadas de reserva.

**Independent Test**: Anunciar a criação de uma sessão com o layout da sala e verificar que todas as poltronas passam a existir no estado livre, com fileira, número e tipo corretos, que reanunciar o mesmo fato não duplica nem reinicia poltronas, e que o estado atual da sessão pode ser consultado.

**Acceptance Scenarios**:

1. **Given** uma sessão ainda desconhecida do estoque, **When** o sistema recebe o anúncio de sessão criada com o layout da sala, **Then** todas as poltronas descritas no layout passam a existir no estado livre, cada uma com fileira, número e tipo.
2. **Given** uma sessão cuja matriz já foi provisionada, **When** o mesmo anúncio de sessão criada é recebido novamente, **Then** nenhuma poltrona é duplicada, nenhum estado corrente é reiniciado e o anúncio é considerado processado com sucesso.
3. **Given** um anúncio de sessão criada cujo layout descreve duas poltronas com a mesma fileira e mesmo número, **When** ele é processado, **Then** o sistema recusa a duplicidade, não provisiona a sessão parcialmente e registra a ocorrência de forma auditável.
4. **Given** uma sessão com poltronas em estados diversos, **When** o mapa de poltronas da sessão é consultado pelo canal síncrono, **Then** cada poltrona é apresentada com fileira, número, tipo, rótulo e estado corrente (livre, reservada ou ocupada).
5. **Given** uma sessão desconhecida do estoque, **When** seu mapa de poltronas é consultado, **Then** o sistema informa que a sessão não é conhecida, de forma distinguível de uma sessão conhecida e sem poltronas.

---

### Edge Cases

- O que acontece quando uma solicitação de bloqueio chega para uma sessão que não tem matriz de poltronas registrada? O pedido é recusado como inválido, sem criar reserva parcial.
- O que acontece quando o anúncio de sessão criada chega depois da primeira solicitação de bloqueio para aquela sessão? Os bloqueios recebidos antes do provisionamento são recusados como inválidos e passam a ser aceitos assim que a matriz existir.
- O que acontece quando uma solicitação tenta bloquear a sala inteira? Ela é recusada como inválida por exceder o limite de poltronas por bloqueio, impedindo que uma única solicitação retire toda a sessão do estoque.
- O que acontece quando a solicitação inclui a mesma poltrona repetida? A solicitação é recusada como inválida, em vez de contar a poltrona duas vezes.
- O que acontece quando o anúncio da reserva criada não pode ser entregue aos demais serviços após o bloqueio já ter sido concedido? O bloqueio permanece válido e o anúncio é entregue posteriormente, sem duplicar a reserva nem prender a resposta síncrona.
- O que acontece quando o mesmo anúncio de desfecho de pagamento é entregue mais de uma vez? O reprocessamento não produz efeito adicional sobre reserva ou poltronas.
- O que acontece quando anúncios de aprovação e de recusa chegam para a mesma reserva, ou fora de ordem? Prevalece o primeiro desfecho aplicado; o segundo é ignorado e registrado como divergência.
- O que acontece quando um anúncio de desfecho não pode ser processado por erro de formato ou reserva desconhecida? Ele é retirado do fluxo normal para inspeção posterior, sem bloquear o processamento dos demais anúncios.
- O que acontece quando a expiração e um desfecho de pagamento disputam a mesma reserva no mesmo instante? Apenas uma transição é aplicada; a reserva nunca termina em dois estados finais distintos.
- O que acontece quando o recurso de controle de concorrência fica indisponível? O serviço recusa novos bloqueios em vez de conceder bloqueios sem garantia de exclusividade.
- O que acontece quando o mapa de poltronas é consultado durante um bloqueio em andamento sobre a mesma sessão? A consulta devolve um retrato coerente, sem poltronas em estado intermediário e sem atrasar o bloqueio.
- O que acontece quando o certificado de serviço do chamador expira em produção? As solicitações passam a ser recusadas na conexão e a recusa é registrada de forma auditável, sem alteração de estado.
- O que acontece quando o serviço reinicia com reservas pendentes em andamento? Os prazos são preservados e continuam valendo a partir do instante de expiração já registrado.

## Requirements *(mandatory)*

### Functional Requirements

**Bloqueio síncrono de poltronas**

- **FR-001**: O sistema MUST expor uma operação síncrona de bloqueio que receba a sessão, a lista de poltronas solicitadas — identificadas pelo rótulo de fileira e número no escopo da sessão (ex.: "A1") — e a identidade da pessoa usuária, e devolva sucesso ou insucesso, o identificador da reserva criada, uma mensagem descritiva e o instante de expiração.
- **FR-002**: O sistema MUST tratar o bloqueio como tudo-ou-nada: se qualquer poltrona solicitada não estiver livre, nenhuma poltrona do pedido muda de estado e nenhuma reserva é criada.
- **FR-003**: O sistema MUST recusar como inválida — de forma distinguível da indisponibilidade — a solicitação com lista de poltronas vazia, com rótulos repetidos, com identidade da pessoa usuária ausente, com sessão desconhecida ou com poltronas que não existam na sessão informada.
- **FR-004**: O sistema MUST recusar como inválida a solicitação que exceda o número máximo de poltronas por bloqueio, cujo padrão é 10 e cujo valor MUST ser configurável por ambiente, informando o limite vigente na recusa.
- **FR-005**: O sistema MUST garantir exclusividade sobre cada poltrona durante o bloqueio, de modo que solicitações concorrentes sobre a mesma poltrona nunca resultem em mais de uma reserva ativa para ela.
- **FR-006**: O sistema MUST recusar novos bloqueios quando não for possível garantir a exclusividade descrita em FR-005, em vez de conceder bloqueios sem essa garantia.
- **FR-007**: O sistema MUST registrar, ao conceder o bloqueio, uma reserva no estado pendente, associada à sessão, à pessoa usuária e ao conjunto exato de poltronas bloqueadas, com instante de expiração igual ao momento da concessão acrescido de 10 minutos.
- **FR-008**: O sistema MUST marcar como reservadas todas as poltronas de uma reserva concedida, e MUST NOT permitir que uma poltrona reservada ou ocupada seja bloqueada por outra solicitação.
- **FR-009**: O sistema MUST responder à solicitação de bloqueio sem aguardar a entrega do anúncio de reserva criada aos demais serviços.

**Ciclo de vida da reserva e das poltronas**

- **FR-010**: O sistema MUST manter cada poltrona em exatamente um dos estados livre, reservada ou ocupada, e cada reserva em exatamente um dos estados pendente, confirmada, expirada ou cancelada.
- **FR-011**: O sistema MUST admitir apenas as transições de reserva pendente → confirmada, pendente → cancelada e pendente → expirada, e MUST NOT alterar o estado de uma reserva que já esteja confirmada, cancelada ou expirada.
- **FR-012**: O sistema MUST invalidar automaticamente toda reserva pendente cujo instante de expiração tenha sido atingido, marcando-a como expirada e devolvendo suas poltronas ao estado livre, sem intervenção manual.
- **FR-013**: O sistema MUST aplicar a invalidação por expiração também às reservas cujo prazo tenha vencido enquanto o serviço estava indisponível, tão logo ele volte a operar.
- **FR-014**: O sistema MUST assegurar que uma reserva confirmada nunca seja expirada e que suas poltronas permaneçam ocupadas.
- **FR-015**: O sistema MUST aplicar cada transição de estado de uma reserva e das poltronas associadas de forma indivisível: ou reserva e todas as suas poltronas mudam de estado, ou nenhuma muda.

**Anúncio de reserva criada**

- **FR-016**: O sistema MUST anunciar, para todo bloqueio concedido, o evento de reserva criada contendo o identificador da reserva, a sessão, a pessoa usuária, as poltronas bloqueadas — pelos mesmos rótulos recebidos na solicitação — e o instante de expiração.
- **FR-017**: O sistema MUST anunciar o evento de reserva criada exatamente uma vez por reserva do ponto de vista de efeito observável, e MUST NOT anunciá-lo quando o bloqueio for recusado.
- **FR-018**: O sistema MUST persistir a reserva antes de anunciá-la e MUST reenviar o anúncio caso a entrega falhe, sem criar reservas adicionais e sem invalidar o bloqueio já concedido.

**Reação aos desfechos de pagamento**

- **FR-019**: O sistema MUST consumir o anúncio de pagamento aprovado e, para a reserva pendente correspondente, marcá-la como confirmada e marcar todas as suas poltronas como ocupadas de forma definitiva.
- **FR-020**: O sistema MUST consumir o anúncio de pagamento recusado e, para a reserva pendente correspondente, marcá-la como cancelada, devolver todas as suas poltronas ao estado livre e liberar o controle de exclusividade associado.
- **FR-021**: O sistema MUST processar os anúncios de desfecho de pagamento de forma idempotente, usando o identificador da reserva como chave: reprocessar o mesmo anúncio não produz efeito adicional e é considerado sucesso.
- **FR-022**: O sistema MUST ignorar, sem alterar estado, o anúncio de desfecho dirigido a uma reserva que já esteja em estado final ou que não exista, e MUST registrar essa ocorrência de forma auditável.
- **FR-023**: O sistema MUST retirar do fluxo normal de processamento, para inspeção posterior, o anúncio que não puder ser processado por formato inválido ou por erro definitivo, sem interromper o processamento dos demais anúncios.
- **FR-024**: O sistema MUST confirmar o consumo de um anúncio somente após concluir com sucesso o efeito correspondente sobre reserva e poltronas.

**Matriz de poltronas**

- **FR-025**: O sistema MUST manter, para cada sessão, o conjunto de poltronas com fileira, número, tipo (normal, PCD, namoradeira) e estado atual.
- **FR-026**: O sistema MUST impedir a existência de duas poltronas com a mesma fileira e mesmo número dentro de uma mesma sessão.
- **FR-027**: O sistema MUST identificar cada poltrona por um rótulo determinístico derivado da fileira e do número (ex.: "A1"), único no escopo da sessão e estável ao longo de toda a vida da sessão, de modo que o cliente possa montá-lo a partir do layout da sala sem consulta prévia ao estoque.
- **FR-028**: O sistema MUST criar as poltronas de uma sessão no estado livre.
- **FR-029**: O sistema MUST expor, no mesmo canal síncrono da operação de bloqueio, uma operação de consulta que receba a sessão e devolva o mapa completo de suas poltronas, cada uma com rótulo, fileira, número, tipo e estado atual.
- **FR-030**: A operação de consulta MUST refletir o estado corrente no instante da leitura, sem defasagem tolerada, e MUST NOT alterar o estado de nenhuma poltrona ou reserva.
- **FR-031**: A operação de consulta MUST informar de forma distinguível quando a sessão solicitada não é conhecida pelo serviço.
- **FR-032**: O sistema MUST registrar, para cada poltrona, o instante da última alteração de estado.
- **FR-033**: O sistema MUST consumir o anúncio de sessão criada, contendo a sessão e o layout da sala (fileira, número e tipo de cada poltrona), e provisionar automaticamente a matriz de poltronas correspondente, sem intervenção manual.
- **FR-034**: O sistema MUST processar o anúncio de sessão criada de forma idempotente, usando o identificador da sessão como chave: reanunciar o mesmo fato não duplica poltronas, não reinicia o estado de poltronas já reservadas ou ocupadas e é considerado sucesso.
- **FR-035**: O sistema MUST provisionar a matriz de uma sessão de forma indivisível: ou todas as poltronas do layout são criadas, ou nenhuma é, e um layout com fileira e número repetidos é recusado como inválido.
- **FR-036**: O sistema MUST recusar como inválida toda solicitação de bloqueio dirigida a uma sessão cuja matriz de poltronas ainda não tenha sido provisionada.

**Segurança do canal síncrono**

- **FR-037**: O sistema MUST aceitar operações do canal síncrono — bloqueio e consulta do mapa de poltronas — apenas de chamadores que apresentem identidade de serviço válida, verificada por autenticação mútua no transporte, e MUST recusar a conexão de chamadores sem identidade válida, com identidade expirada ou emitida por autoridade não reconhecida.
- **FR-038**: O sistema MUST tratar a identidade da pessoa usuária recebida na solicitação como confiável, sem validá-la junto ao emissor de identidade, uma vez que o chamador foi autenticado conforme FR-037.
- **FR-039**: O sistema MUST registrar de forma auditável toda tentativa de conexão recusada por falha de autenticação de serviço, sem gravar material criptográfico em texto claro.
- **FR-040**: O material de identidade de serviço (certificados e âncoras de confiança) MUST ser fornecido externamente ao artefato e substituível sem alteração do código entregue.

**Configuração e operação**

- **FR-041**: Todas as configurações de ambiente — acesso ao armazenamento de dados, ao mecanismo de exclusividade, ao intermediário de mensagens, o prazo de expiração e o limite de poltronas por bloqueio — MUST ser fornecidas externamente ao artefato, sem valores sensíveis embutidos no código.
- **FR-042**: O sistema MUST emitir, para cada solicitação de bloqueio e para cada anúncio consumido, um registro estruturado e legível por máquina contendo identificador de correlação, operação, desfecho e duração, sem gravar dados sensíveis em texto claro.
- **FR-043**: O sistema MUST expor métricas de volume, latência e desfecho das solicitações de bloqueio (concedido, indisponível, inválido, falha) e do consumo de anúncios, consumíveis por um coletor externo.
- **FR-044**: O sistema MUST participar do rastreamento distribuído, aceitando o contexto recebido do serviço chamador e propagando-o, de modo que uma solicitação de reserva possa ser seguida de ponta a ponta entre os serviços.
- **FR-045**: O sistema MUST expor um indicador de saúde que permita a um orquestrador saber se a instância está apta a receber tráfego, refletindo a disponibilidade do armazenamento, do mecanismo de exclusividade e do intermediário de mensagens.
- **FR-046**: O sistema MUST distinguir, nas respostas de insucesso do bloqueio, entrada inválida, indisponibilidade das poltronas e falha temporária do serviço, de forma que o serviço chamador possa reagir sem interpretar texto livre.

### Key Entities *(include if feature involves data)*

- **Poltrona**: assento individual de uma sessão. Sessão a que pertence, fileira, número, tipo (normal, PCD, namoradeira), estado (livre, reservada, ocupada) e instantes de criação e de última alteração. É identificada pelo rótulo determinístico derivado de fileira + número (ex.: "A1"), único e estável no escopo da sessão; a combinação sessão + fileira + número não admite duplicidade.
- **Reserva**: intenção de compra de uma pessoa identificada sobre um conjunto de poltronas de uma sessão. Identificador próprio, sessão, pessoa usuária, instante de expiração, estado (pendente, confirmada, expirada, cancelada) e instante de criação.
- **Vínculo reserva–poltrona**: associação entre uma reserva e cada poltrona que ela bloqueia; uma reserva cobre uma ou mais poltronas e uma poltrona pertence a no máximo uma reserva não finalizada por vez.
- **Anúncio de sessão criada**: fato recebido do serviço de catálogo quando uma nova sessão é agendada, contendo a sessão e o layout da sala (fileira, número e tipo de cada poltrona); identificado pela sessão, que serve de chave de idempotência do provisionamento.
- **Anúncio de reserva criada**: fato publicado para os demais serviços quando um bloqueio é concedido, contendo reserva, sessão, pessoa usuária, poltronas e instante de expiração.
- **Anúncio de desfecho de pagamento**: fato recebido dos serviços de pagamento indicando aprovação ou recusa referente a uma reserva; identificado pela reserva, que serve de chave de idempotência.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 99% das solicitações de bloqueio recebem resposta conclusiva em menos de 100 milissegundos, medido no serviço, com a sessão carregada e sob concorrência típica de venda.
- **SC-002**: Em disputa simultânea pelas mesmas poltronas, no máximo uma solicitação é concedida: verificado por teste de concorrência com 100 solicitações paralelas sobre o mesmo conjunto de poltronas, em que exatamente uma obtém sucesso e nenhuma poltrona fica vinculada a duas reservas.
- **SC-003**: 100% das poltronas de reservas não pagas voltam a ficar disponíveis em até 30 segundos após o vencimento do prazo de 10 minutos.
- **SC-004**: 100% dos anúncios reprocessados — desfechos de pagamento e sessões criadas — resultam no mesmo estado final de reserva, poltronas e matriz do primeiro processamento, verificado por teste com entrega repetida do mesmo anúncio.
- **SC-005**: 100% dos bloqueios concedidos resultam em um anúncio de reserva criada disponível aos consumidores, mesmo quando o intermediário de mensagens esteve indisponível no instante da concessão.
- **SC-006**: Nenhuma poltrona termina em estado inconsistente após 1.000 ciclos de bloqueio seguidos aleatoriamente de aprovação, recusa ou abandono: toda poltrona está livre ou ocupada ao final, e toda reserva está em um único estado final.
- **SC-007**: 100% das solicitações de bloqueio inválidas são recusadas sem criar reserva e sem alterar o estado de qualquer poltrona.
- **SC-008**: Após uma parada de 15 minutos do serviço, 100% das reservas cujo prazo venceu durante a parada estão invalidadas e suas poltronas liberadas em até 1 minuto após o retorno.
- **SC-009**: 100% das solicitações de reserva podem ser reconstituídas de ponta a ponta a partir dos sinais emitidos — do bloqueio ao desfecho de pagamento — usando um único identificador de correlação.
- **SC-010**: Uma nova instância do serviço sobe e passa a atender tráfego apenas com configuração externa e material de identidade fornecido externamente, sem qualquer alteração no artefato entregue.
- **SC-011**: 100% das tentativas de bloqueio originadas de chamadores sem identidade de serviço válida são recusadas antes de qualquer alteração de estado, verificado por teste com chamador sem identidade, com identidade expirada e com identidade emitida por autoridade não reconhecida.
- **SC-012**: Durante indisponibilidade do mecanismo de exclusividade, 100% das solicitações de bloqueio são recusadas com falha temporária e nenhuma reserva é criada — nunca há concessão sem garantia de exclusividade.
- **SC-013**: 99% das consultas ao mapa de poltronas de uma sessão retornam em menos de 200 milissegundos para uma sala de até 500 lugares, e o resultado reflete alterações de estado ocorridas imediatamente antes da consulta.
- **SC-014**: 100% das solicitações que excedem o limite de poltronas por bloqueio são recusadas como inválidas sem criar reserva, verificado com o limite padrão de 10 e com um limite reconfigurado.
- **SC-015**: 100% das sessões anunciadas como criadas têm sua matriz de poltronas disponível para bloqueio em até 1 minuto após o anúncio, com todas as poltronas do layout no estado livre.

## Assumptions

- A matriz de poltronas de cada sessão é provisionada automaticamente pelo consumo do anúncio de sessão criada emitido pelo serviço de catálogo, que é o dono do cadastro de sessões e do layout das salas; este serviço não expõe interface administrativa para criar, editar ou excluir sessões e salas.
- O anúncio de sessão criada carrega o layout completo da sala (fileira, número e tipo de cada poltrona); este serviço não consulta o catálogo de forma síncrona para obtê-lo.
- Alterações posteriores no layout da sala (poltronas acrescentadas, removidas ou reclassificadas após a criação da sessão) estão fora do escopo desta feature.
- O serviço de catálogo é o único chamador da operação síncrona de bloqueio e já validou a existência da sessão e a autenticidade da pessoa usuária antes de chamar; este serviço autentica o chamador por identidade de serviço no transporte e, feito isso, confia na identidade da pessoa recebida, sem validar credenciais de pessoa usuária por conta própria.
- Existe (ou existirá) no ambiente de execução uma autoridade emissora de identidades de serviço e um processo de rotação desse material; este serviço apenas consome o material fornecido, não o emite nem o renova.
- O serviço de catálogo conhece o layout da sala (ele o publica no anúncio de sessão criada) e, portanto, consegue montar os rótulos de poltrona da solicitação de bloqueio sem consultar o estoque previamente; o anúncio de reserva criada devolve exatamente os rótulos recebidos.
- O serviço de pagamento já existe (ou será entregue em paralelo) e é a única fonte de verdade sobre a aprovação ou recusa de um pagamento; este serviço apenas reage aos desfechos anunciados e nunca consulta o pagamento de forma síncrona.
- O prazo de reserva é de 10 minutos por padrão, configurável por ambiente, e é contado a partir do instante em que o bloqueio é concedido.
- O limite de 10 poltronas por solicitação de bloqueio é a única contenção de volume aplicada por este serviço; ele não limita quantas solicitações uma mesma pessoa ou chamador pode fazer, o que continua sendo responsabilidade da camada de borda.
- O cancelamento voluntário da reserva pela pessoa usuária, o reembolso, a emissão do ingresso e a precificação estão fora do escopo desta feature.
- Uma poltrona ocupada permanece ocupada até o fim da vida útil da sessão; não há fluxo de devolução ao estoque após a confirmação nesta feature.
- Existe (ou existirá) infraestrutura externa de coleta de métricas e de rastreamento distribuído no ambiente de execução; este serviço apenas emite e propaga os sinais.
- O intermediário de mensagens garante entrega ao menos uma vez, podendo duplicar e reordenar anúncios; por isso a idempotência por identificador de reserva e de sessão é requisito e não otimização.
- A limitação de volume de requisições por origem é responsabilidade da camada que fica à frente do serviço, e não deste serviço, que atende apenas chamadores internos autenticados.
- Idioma dos dados e das mensagens de domínio é português do Brasil; internacionalização está fora do escopo.
