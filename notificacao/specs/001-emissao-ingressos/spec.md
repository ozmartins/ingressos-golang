# Feature Specification: Emissão e Validação de Ingressos Digitais (Servico-Notificacao)

**Feature Branch**: `001-emissao-ingressos`

**Created**: 2026-08-30

**Status**: Draft

**Input**: User description: "Sigas as instruções do arquivo ers-notificacao.md" (ERS do microsserviço `Servico-Notificacao`, arquivo `notificacao/erp-notificacao.md`)

## Clarifications

### Session 2026-08-30

- Q: Qual o escopo da entrega do aviso à pessoa nesta feature? → A: Canal simulado atrás de uma porta de domínio, com sucesso e falha controláveis nos testes; a integração com provedor real de e-mail/push/SMS fica fora do escopo
- Q: O que leva um ingresso ao estado cancelado nesta feature? → A: Nada — o estado é honrado (a validação nega e a listagem o exibe), mas nenhuma operação desta feature o produz; o gatilho de cancelamento é escopo de feature futura
- Q: Como distinguir falha transitória de anúncio malformado no processamento? → A: Falha transitória é reentregue com limite de tentativas e, esgotado o limite, cai na mesma quarentena; anúncio malformado nunca é retentado, vai direto para a quarentena
- Q: Qual a forma da listagem de ingressos da pessoa? → A: Lista completa, sem paginação, ordenada do mais recente para o mais antigo, com filtro opcional por estado
- Q: Quando o aviso é disparado em relação ao processamento do anúncio? → A: Dentro do mesmo processamento, depois de o ingresso estar gravado; a falha do aviso é capturada e registrada, nunca propagada

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Emitir o ingresso assim que o pagamento é confirmado (Priority: P1)

Quando o pagamento de uma reserva é confirmado, o serviço reage a esse anúncio por conta própria, sem que ninguém espere por ele: registra o ingresso digital da reserva e gera o código de acesso que a pessoa vai apresentar na entrada da sala. A pessoa que comprou não precisa fazer nada; quando ela abrir o aplicativo, o ingresso já existe.

**Why this priority**: É a razão de existir do serviço. Sem a emissão não há ingresso para consultar nem para validar na portaria — nenhuma das outras jornadas tem objeto. Entrega valor sozinha: uma reserva paga passa a ter um bilhete verificável.

**Independent Test**: Anunciar um pagamento confirmado e verificar que exatamente um ingresso é registrado para aquela reserva, com código de acesso único, estado válido e instante de criação; reanunciar o mesmo pagamento e verificar que nenhum segundo ingresso aparece.

**Acceptance Scenarios**:

1. **Given** um anúncio de pagamento confirmado com reserva, pessoa, transação, valor e instante de pagamento válidos, **When** o serviço o recebe, **Then** ele registra um ingresso para aquela reserva, com identificador próprio, código de acesso único, estado válido e instante de criação.
2. **Given** um anúncio de pagamento confirmado já processado antes, **When** o mesmo anúncio é entregue novamente, **Then** nenhum segundo ingresso é criado para a reserva, o código de acesso do ingresso existente permanece o mesmo e a reentrega é concluída sem erro.
3. **Given** dois anúncios da mesma reserva processados ao mesmo tempo, **When** ambos tentam emitir, **Then** no máximo um ingresso passa a existir para aquela reserva.
4. **Given** um anúncio em que falta um dado obrigatório, um identificador está malformado ou o instante de pagamento é inválido, **When** ele é recebido, **Then** nenhum ingresso é emitido, o anúncio é retirado de circulação para inspeção humana e a ocorrência fica registrada de forma auditável.
5. **Given** dois anúncios de reservas diferentes, **When** ambos são emitidos, **Then** os códigos de acesso gerados são distintos entre si.
6. **Given** um anúncio válido cujo processamento falha por indisponibilidade temporária de um recurso do serviço, **When** a falha ocorre, **Then** nenhum ingresso parcial fica registrado, o anúncio é reentregue para nova tentativa e, se o limite de tentativas se esgotar sem sucesso, ele vai para a quarentena com o motivo do esgotamento — nunca para a quarentena na primeira falha.

---

### User Story 2 - Validar o ingresso na entrada da sala (Priority: P2)

Na portaria do cinema, o dispositivo de leitura apresenta o código de acesso do ingresso e recebe imediatamente o veredito: entrada autorizada ou negada, com o motivo. Um ingresso autorizado é dado como utilizado no mesmo ato, de modo que a mesma leitura não autoriza uma segunda entrada.

**Why this priority**: É o que transforma o ingresso emitido em entrada efetiva na sala e é a única defesa do cinema contra reuso e falsificação de bilhete. Depende da emissão existir, mas é testável sozinha a partir de um ingresso já emitido.

**Independent Test**: A partir de um ingresso emitido, apresentar seu código na validação e verificar que a primeira apresentação autoriza e dá baixa, e que a segunda apresentação nega informando reuso.

**Acceptance Scenarios**:

1. **Given** um ingresso válido, **When** seu código é apresentado na validação por um dispositivo autorizado, **Then** a entrada é autorizada, o ingresso passa a utilizado, o instante de utilização é gravado e a resposta identifica o ingresso e esse instante.
2. **Given** um ingresso já utilizado, **When** seu código é apresentado de novo, **Then** a entrada é negada informando reuso, e o instante de utilização originalmente gravado não é alterado.
3. **Given** um ingresso cancelado, **When** seu código é apresentado, **Then** a entrada é negada informando que o ingresso não está válido, e o ingresso permanece cancelado.
4. **Given** um código inexistente, adulterado ou cuja autenticidade não se confirma, **When** ele é apresentado, **Then** a entrada é negada com uma resposta que não permite distinguir código inexistente de código adulterado.
5. **Given** o mesmo código válido apresentado simultaneamente em dois dispositivos, **When** ambas as leituras são processadas, **Then** exatamente uma autoriza a entrada e a outra é negada por reuso.
6. **Given** uma requisição de validação sem credencial de dispositivo ou com credencial não reconhecida, **When** ela chega, **Then** ela é recusada sem consultar o ingresso e sem revelar se o código existe.

---

### User Story 3 - Consultar os meus ingressos (Priority: P3)

A pessoa autenticada abre o aplicativo e vê os ingressos que possui — os válidos, para apresentar na entrada, e os já utilizados ou cancelados, como histórico de compras — cada um com o código de acesso, a reserva de origem, o estado e a data de emissão.

**Why this priority**: É como a pessoa chega até o bilhete que já lhe pertence. Sem essa consulta o ingresso existe mas é inalcançável pelo dono. Fica abaixo da validação porque um ingresso pode ser apresentado por outros meios, mas a portaria não tem substituto.

**Independent Test**: Autenticar-se como uma pessoa que tem ingressos emitidos e verificar que a listagem devolve exatamente os ingressos dela, do mais recente para o mais antigo, com estado e data, que o filtro por estado recorta corretamente, e que nenhum ingresso de outra pessoa aparece.

**Acceptance Scenarios**:

1. **Given** uma pessoa autenticada com ingressos emitidos, **When** ela consulta os próprios ingressos, **Then** recebe todos os seus ingressos — válidos, utilizados e cancelados — cada um com identificador, reserva de origem, código de acesso, estado e instante de criação.
2. **Given** uma pessoa autenticada com ingressos emitidos em instantes diferentes, **When** ela consulta, **Then** os ingressos vêm ordenados do mais recente para o mais antigo.
3. **Given** uma pessoa autenticada com ingressos em mais de um estado, **When** ela consulta filtrando por um estado, **Then** recebe apenas os ingressos naquele estado, na mesma ordem, e nenhum dos demais.
4. **Given** uma consulta com filtro de estado não reconhecido, **When** ela chega, **Then** é recusada como pedido inválido, e não devolve a lista inteira.
5. **Given** uma pessoa autenticada sem nenhum ingresso, **When** ela consulta, **Then** recebe uma listagem vazia, e não um erro.
6. **Given** ingressos pertencentes a outras pessoas, **When** a pessoa autenticada consulta, **Then** nenhum ingresso de terceiro aparece na resposta.
7. **Given** uma requisição sem credencial de pessoa, com credencial expirada ou com credencial cuja autenticidade não se confirma, **When** ela chega, **Then** é recusada e nenhum ingresso é revelado.

---

### User Story 4 - Registrar o aviso de confirmação enviado à pessoa (Priority: P4)

A cada ingresso emitido, o serviço avisa a pessoa que a compra foi confirmada e guarda o registro desse aviso: por qual canal saiu, se saiu, e o detalhe do que aconteceu quando não saiu. O registro é o que permite descobrir depois quem ficou sem aviso e reenviar.

**Why this priority**: É rastreabilidade e base para reenvio, não pré-requisito da entrada na sala: a pessoa que não recebeu o aviso ainda consegue ver e usar o ingresso. Testável sozinha a partir de um ingresso emitido.

**Independent Test**: Emitir um ingresso com o canal de aviso funcionando e verificar que fica um registro de aviso enviado; repetir com o canal falhando e verificar que o ingresso continua emitido e válido e que fica um registro de falha com o detalhe do erro.

**Acceptance Scenarios**:

1. **Given** um ingresso recém-emitido e o canal de aviso operante, **When** o aviso é disparado, **Then** fica registrado um aviso para aquele ingresso e aquela pessoa, com o canal usado, o estado de enviado e o instante do envio.
2. **Given** um ingresso recém-emitido e o canal de aviso indisponível ou recusando o envio, **When** o disparo falha, **Then** o ingresso permanece emitido e válido, fica registrado um aviso com estado de falha e o detalhe do motivo, e o anúncio de pagamento é dado por processado com sucesso — sem reentrega e sem quarentena.
3. **Given** um aviso registrado como falho, **When** alguém consulta os avisos daquele ingresso, **Then** o registro está lá, associado ao ingresso e à pessoa, identificável como pendente de reenvio.

---

### Edge Cases

- **Anúncio de pagamento de uma reserva que já tem ingresso cancelado**: a reserva já foi consumida uma vez; a reentrega não ressuscita nem duplica o ingresso — o estado atual do ingresso prevalece.
- **Código de acesso apresentado com formato irreconhecível** (vazio, truncado, tamanho absurdo): negado como qualquer código inválido, sem custo de consulta ao acervo de ingressos.
- **Credencial de dispositivo de portaria válida usada para consultar ingressos de cliente**, ou credencial de pessoa usada para validar na portaria: cada operação aceita apenas o tipo de credencial que lhe corresponde; a outra é recusada.
- **Falha do serviço entre gravar o ingresso e disparar o aviso** (queda do processo, não falha do canal): o ingresso emitido permanece e a ausência de registro de aviso identifica o caso para reenvio. Se o anúncio for reentregue, a idempotência por reserva impede ingresso novo — e, como o ingresso já existe, o aviso pode acabar não saindo por essa via, o que reforça o registro como a única evidência de quem ficou sem aviso.
- **Pessoa consulta enquanto uma validação está em curso** para um de seus ingressos: a listagem devolve um estado coerente do ingresso — válido ou utilizado — nunca um estado intermediário.
- **Anúncio válido que falha de forma transitória repetidas vezes**: cada tentativa é contada; enquanto o limite não se esgota o anúncio continua sendo material de reprocessamento automático, e a idempotência por reserva impede que uma tentativa tardia bem-sucedida some com uma anterior.
- **Volume de ingressos de uma mesma pessoa muito grande**: a listagem devolve tudo de uma vez e continua respondendo dentro do prazo aceitável; se o volume por pessoa vier a tornar isso inviável, paginação é feature nova, não ajuste silencioso desta.

## Requirements *(mandatory)*

### Functional Requirements

**Emissão**

- **FR-001**: O sistema MUST reagir a cada anúncio de pagamento confirmado sem que nenhum interessado precise esperar pela emissão.
- **FR-002**: O sistema MUST verificar, antes de emitir, que o anúncio traz reserva, pessoa, transação e instante de pagamento presentes e bem formados; caso contrário MUST NOT emitir ingresso, MUST retirar o anúncio de circulação para inspeção humana e MUST registrar a ocorrência de forma auditável.
- **FR-003**: O sistema MUST registrar, para um anúncio válido, um ingresso contendo identificador próprio, a reserva de origem, a pessoa dona, o código de acesso, o estado válido e o instante de criação.
- **FR-004**: O sistema MUST garantir no máximo um ingresso por reserva, inclusive sob reentrega do mesmo anúncio e sob processamento simultâneo de anúncios da mesma reserva; a reentrega MUST ser concluída sem erro e sem alterar o ingresso existente.
- **FR-005**: O sistema MUST gerar um código de acesso único entre todos os ingressos, não adivinhável a partir de outro código, e MUST NOT expor nele, em texto legível, dado da pessoa, da reserva ou da compra.
- **FR-006**: O sistema MUST ser capaz de verificar, no ato da validação, que um código de acesso apresentado foi gerado por ele e não foi adulterado.
- **FR-022**: O sistema MUST distinguir defeito permanente de defeito transitório ao processar um anúncio. Um anúncio malformado (FR-002) MUST NOT ser retentado e vai direto para a quarentena. Uma falha transitória do próprio serviço ou de seus recursos MUST provocar nova tentativa de processamento do mesmo anúncio, limitada a um número máximo de tentativas; esgotado o limite, o anúncio MUST ir para a mesma quarentena, com o motivo do esgotamento registrado. Nenhuma retentativa MUST resultar em ingresso duplicado (FR-004).

**Validação na portaria**

- **FR-007**: O sistema MUST autorizar a entrada quando o código apresentado corresponder a um ingresso em estado válido, e no mesmo ato MUST passá-lo a utilizado e gravar o instante da utilização.
- **FR-008**: O sistema MUST negar a entrada, informando reuso, quando o código corresponder a um ingresso já utilizado, e MUST NOT alterar o instante de utilização já gravado.
- **FR-009**: O sistema MUST negar a entrada quando o código corresponder a um ingresso cancelado.
- **FR-010**: O sistema MUST negar a entrada quando o código for inexistente, malformado ou de autenticidade não confirmada, com resposta que MUST NOT permitir distinguir esses casos entre si.
- **FR-011**: O sistema MUST garantir que apresentações simultâneas do mesmo código válido resultem em no máximo uma autorização de entrada.
- **FR-012**: O sistema MUST exigir credencial de dispositivo de portaria na operação de validação e MUST recusar a requisição sem consultar o ingresso quando ela estiver ausente ou não for reconhecida.

**Consulta pela pessoa**

- **FR-013**: Uma pessoa autenticada MUST poder listar os próprios ingressos, em todos os estados, cada um com identificador, reserva de origem, código de acesso, estado e instante de criação.
- **FR-023**: A listagem MUST devolver todos os ingressos da pessoa de uma vez, sem paginação, ordenados do instante de criação mais recente para o mais antigo.
- **FR-024**: A listagem MUST aceitar um filtro opcional por estado, devolvendo apenas os ingressos naquele estado quando ele for informado e todos quando não for; um estado não reconhecido MUST ser recusado como pedido inválido, e MUST NOT ser tratado como ausência de filtro.
- **FR-014**: O sistema MUST restringir a listagem aos ingressos da pessoa autenticada e MUST NOT revelar ingresso de terceiro por nenhum parâmetro da requisição.
- **FR-015**: O sistema MUST recusar a listagem quando a credencial da pessoa estiver ausente, expirada ou com autenticidade não confirmada.

**Aviso à pessoa**

- **FR-016**: O sistema MUST disparar, para cada ingresso emitido, um aviso de confirmação à pessoa dona, dentro do mesmo processamento do anúncio e somente depois de o ingresso estar gravado.
- **FR-025**: Uma falha no disparo do aviso MUST ser capturada e registrada no local do disparo e MUST NOT ser propagada como falha do processamento do anúncio: o anúncio MUST ser dado por processado com sucesso, sem reentrega e sem quarentena (FR-022), ainda que o aviso tenha falhado. Este requisito trata do **anúncio**; o efeito da mesma falha sobre o ingresso é a FR-018. Os dois se aplicam à mesma falha e nenhum implica o outro: um ingresso pode continuar válido (FR-018) enquanto a mensagem é reprocessada indevidamente, e é essa combinação que a FR-025 proíbe.
- **FR-017**: O sistema MUST registrar cada disparo de aviso com o ingresso, a pessoa, o canal, o resultado (enviado ou falho), o detalhe em caso de falha e o instante.
- **FR-018**: Uma falha no disparo do aviso MUST NOT desfazer, bloquear nem invalidar a emissão do ingresso; o ingresso MUST permanecer válido e o registro do aviso MUST ficar identificável como pendente de reenvio. Este requisito trata do **ingresso**; o efeito da mesma falha sobre o processamento do anúncio é a FR-025.

**Ciclo de vida e integridade**

- **FR-019**: O sistema MUST admitir apenas as transições de estado de válido para utilizado e de válido para cancelado; qualquer outra transição MUST ser rejeitada sem alterar o ingresso. Nesta feature apenas a transição para utilizado tem gatilho: nenhuma operação aqui leva um ingresso a cancelado, e o domínio MUST ainda assim rejeitar a saída de um ingresso cancelado para qualquer outro estado.
- **FR-020**: O sistema MUST NOT permitir alteração da reserva de origem, da pessoa dona, do código de acesso ou do instante de criação de um ingresso depois de emitido.
- **FR-021**: O sistema MUST registrar de forma auditável, em cada emissão, validação e disparo de aviso, o desfecho da operação e o ingresso ou anúncio envolvido, sem gravar o código de acesso em texto legível fora do próprio ingresso.

### Key Entities

- **Ingresso Emitido**: o bilhete digital de uma reserva paga. Identificado por si próprio, referencia exatamente uma reserva (relação de um para um) e a pessoa dona. Guarda o código de acesso apresentado na portaria, o estado atual (válido, utilizado ou cancelado), o instante em que foi utilizado, quando o foi, e o instante de emissão.
- **Registro de Aviso**: o comprovante de uma tentativa de avisar a pessoa sobre um ingresso. Pertence a um ingresso e à pessoa dona dele; guarda o canal usado, o resultado (enviado ou falho), o detalhe do que aconteceu e o instante da tentativa. Um ingresso pode ter vários registros ao longo do tempo.
- **Anúncio de Pagamento Confirmado**: o fato externo que dá origem a um ingresso. Traz a reserva, a pessoa, a transação, o valor pago e o instante do pagamento. Não é guardado como entidade própria: sua marca no sistema é o ingresso que ele produziu, e é a reserva que ele carrega que impede a emissão em duplicidade.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% das reservas com pagamento confirmado passam a ter exatamente um ingresso, mesmo quando o anúncio de pagamento é entregue mais de uma vez ou processado em paralelo.
- **SC-002**: O ingresso está disponível para a pessoa em até 5 segundos após a confirmação do pagamento, em 95% dos casos.
- **SC-003**: A portaria recebe o veredito de uma leitura em até 1 segundo em 99% das validações, com o cinema operando em horário de pico.
- **SC-004**: Nenhuma entrada é autorizada duas vezes com o mesmo ingresso: 0 ocorrências de dupla autorização em uma bateria de leituras simultâneas do mesmo código.
- **SC-005**: Nenhum código de acesso forjado ou adulterado é aceito na portaria em uma bateria de tentativas de falsificação.
- **SC-006**: Nenhuma pessoa consegue ver ou validar ingresso de outra pessoa: 0 ocorrências em uma bateria de tentativas de acesso cruzado.
- **SC-007**: 100% das falhas de aviso ficam registradas com motivo e ingresso associado, e nenhuma delas impede a entrada da pessoa na sala.
- **SC-008**: 100% dos anúncios de pagamento malformados terminam em inspeção humana, sem ingresso emitido e sem descarte silencioso.
- **SC-010**: 100% dos anúncios que falham por indisponibilidade temporária e voltam a ser processados dentro do limite de tentativas terminam com o ingresso emitido, sem intervenção humana.
- **SC-009**: A pessoa vê a lista completa dos próprios ingressos, do mais recente para o mais antigo, em até 2 segundos em 95% das consultas, para uma pessoa com até 200 ingressos no histórico.

## Assumptions

- **Origem do anúncio**: o anúncio de pagamento confirmado é publicado pelo `Servico-Pagamento`, no formato já definido na ERS. Este serviço é consumidor: não define nem altera esse contrato, e uma divergência entre o publicado e o esperado é pergunta ao mantenedor, não ajuste unilateral.
- **Identidade da pessoa**: a autenticação da pessoa é a mesma já usada nos demais serviços da plataforma (credencial emitida pelo provedor de identidade central, verificada sem consulta a ele). Este serviço não cria, não guarda e não gerencia contas.
- **Credencial da portaria**: os dispositivos de portaria usam uma credencial de serviço distinta da credencial de pessoa, configurada no ambiente. Gestão do ciclo de vida dessa credencial (emissão, rotação, revogação por dispositivo) está fora do escopo desta feature.
- **Autenticidade do código de acesso**: o código é verificável por um segredo que só este serviço conhece, sem consulta ao acervo de ingressos para detectar adulteração. O segredo vem da configuração do ambiente; sua rotação está fora do escopo.
- **Reenvio de avisos**: o registro de falha existe para permitir reenvio, mas o disparo do reenvio — automático ou operado por alguém — está fora do escopo desta feature. Nada nesta feature reenvia sozinho.
- **Uma reserva, um ingresso**: a ERS estabelece a reserva como chave única de emissão, portanto uma reserva produz um único ingresso, ainda que corresponda a várias poltronas. Ingresso por poltrona está fora do escopo.
- **Entrega externa do aviso**: o canal de aviso é um ponto de saída substituível, atendido nesta feature por uma implementação simulada cujo sucesso ou falha pode ser controlado nos testes. A integração com provedor externo real de e-mail, push ou SMS está fora do escopo — o que a feature exige é que a tentativa aconteça, que o desfecho seja observado e que fique registrado.
- **Aviso não bloqueia a emissão**: o disparo síncrono é uma escolha de simplicidade, não um acoplamento: nenhuma promessa desta feature depende de o aviso ter saído, e SC-002 (ingresso disponível em 5 segundos) é medido sobre a gravação do ingresso, não sobre o aviso.
- **Conteúdo do aviso**: o texto e o formato da mensagem enviada à pessoa não são objeto desta feature; o que ela exige é que a tentativa aconteça e fique registrada.
- **Cancelamento não é produzido aqui**: o estado cancelado existe e é honrado na validação e na listagem, mas nenhuma operação desta feature leva um ingresso a ele — não há anúncio de cancelamento consumido nem operação de cancelamento exposta. O gatilho do cancelamento (estorno, cancelamento de sessão) é escopo de uma feature futura.
- **Sem expiração automática**: um ingresso válido não expira sozinho pela passagem da data da sessão; ele sai do estado válido apenas por utilização ou cancelamento.
- **Retenção**: ingressos e registros de aviso são mantidos indefinidamente, servindo de histórico de compra para a pessoa e de trilha de auditoria para o cinema.
