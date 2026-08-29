# Feature Specification: Catálogo de Filmes, Sessões e Reserva de Poltronas

**Feature Branch**: `001-catalogo-sessoes-reserva`

**Created**: 2026-08-29

**Status**: Draft

**Input**: User description: "As instruções estão no arquivo ers-catalogo.md" (ERS do microsserviço `Servico-Catalogo`)

## Clarifications

### Session 2026-08-29

- Q: Qual a política de resiliência nas chamadas ao serviço de estoque (retentativa? interrupção após falhas seguidas?) → A: Sem retentativa; após uma sequência de falhas consecutivas, o serviço passa a recusar rápido por um período, retomando automaticamente
- Q: A grade de sessões e as demais listagens devem ser paginadas? → A: Paginação obrigatória em todas as consultas de coleção (filmes, cinemas, salas e sessões), sem exceção
- Q: Como proteger os endpoints públicos anônimos contra volume excessivo de requisições? → A: Fora de escopo deste serviço; responsabilidade declarada da camada de borda (gateway/ingress)
- Q: Qual a profundidade de observabilidade exigida? → A: Logs estruturados + métricas por operação + rastreamento distribuído propagado até o serviço de estoque
- Q: Que defasagem de dados é tolerada nas consultas? → A: Nenhuma; toda consulta reflete o estado atual (decisão revista pelo usuário: a tolerância de 5 minutos foi removida por não restringir nenhuma implementação plausível nos volumes previstos)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Descobrir filmes em cartaz (Priority: P1)

Uma pessoa abre o aplicativo do cinema sem estar autenticada e quer saber quais filmes estão em cartaz ou entram em breve, com informações suficientes para decidir o que assistir: título, sinopse, duração, classificação etária, gênero e pôster.

**Why this priority**: É a porta de entrada de toda a jornada de compra. Sem a listagem de filmes, nenhuma outra funcionalidade tem valor para o cliente final, e ela já entrega valor sozinha (vitrine de programação).

**Independent Test**: Cadastrar um conjunto de filmes com status distintos, consultar a listagem sem qualquer credencial e verificar que os filmes retornados e seus atributos correspondem ao esperado, inclusive ao filtrar por situação.

**Acceptance Scenarios**:

1. **Given** existem filmes cadastrados com situações "em cartaz", "em breve" e "fora de cartaz", **When** a pessoa consulta a lista de filmes sem filtro, **Then** o sistema retorna a primeira página dos filmes em cartaz e em breve, cada um com título, sinopse, duração, classificação etária, gênero, imagem e situação.
2. **Given** existem filmes cadastrados, **When** a pessoa consulta a lista filtrando pela situação "em cartaz", **Then** apenas filmes nessa situação são retornados.
3. **Given** não existe nenhum filme que atenda ao filtro informado, **When** a pessoa consulta a lista, **Then** o sistema retorna uma página vazia e não uma condição de erro.
4. **Given** a pessoa informa um valor de situação inexistente, **When** consulta a lista, **Then** o sistema informa que o filtro é inválido, indicando os valores aceitos.

---

### User Story 2 - Consultar a grade de sessões (Priority: P1)

Depois de escolher um filme, a pessoa quer ver onde e quando pode assisti-lo: em quais cinemas, em qual sala e tipo de tela, em que horário, em qual idioma e por qual preço base — podendo filtrar por filme, cinema e data.

**Why this priority**: É o passo indispensável entre descobrir o filme e reservar poltronas. Entrega valor de forma autônoma (consulta de programação) e é pré-requisito da reserva.

**Independent Test**: Cadastrar sessões em cinemas, salas e datas diferentes e verificar que cada combinação de filtros (filme, cinema, data, e a ausência deles) retorna exatamente as sessões esperadas com os dados de exibição consolidados.

**Acceptance Scenarios**:

1. **Given** existem sessões cadastradas para vários filmes e cinemas, **When** a pessoa consulta a grade filtrando por um filme, **Then** apenas as sessões daquele filme são retornadas, cada uma com nome do cinema, número da sala, tipo de tela, data e hora de início, idioma e preço base.
2. **Given** existem sessões em datas diferentes, **When** a pessoa filtra por uma data específica, **Then** somente as sessões que começam naquela data são retornadas.
3. **Given** a pessoa combina filtros de filme, cinema e data, **When** consulta a grade, **Then** apenas as sessões que satisfazem simultaneamente todos os filtros são retornadas.
4. **Given** existem mais sessões do que cabem em uma página, **When** a pessoa avança para a página seguinte, **Then** o sistema retorna os próximos resultados sem repetir nem omitir sessões já exibidas, indicando se ainda há mais páginas.
5. **Given** a pessoa informa uma data em formato inválido, **When** consulta a grade, **Then** o sistema informa que o filtro é inválido, indicando o formato esperado.
6. **Given** existem sessões canceladas ou já finalizadas, **When** a pessoa consulta a grade sem filtro de situação, **Then** essas sessões não aparecem entre as opções oferecidas ao cliente.

---

### User Story 3 - Reservar poltronas para uma sessão (Priority: P1)

Uma pessoa autenticada escolhe uma sessão e um conjunto de poltronas e solicita a reserva. O sistema identifica quem é a pessoa a partir de suas credenciais, pede o bloqueio das poltronas ao serviço responsável pelo estoque e devolve rapidamente o resultado: reserva criada com prazo de expiração, ou a informação de que alguma poltrona não está mais disponível.

**Why this priority**: É a conversão da jornada — o momento em que a navegação vira intenção de compra. Depende das histórias anteriores para ter contexto, mas é a razão de existir do serviço.

**Independent Test**: Com uma sessão conhecida e credenciais válidas, solicitar a reserva de poltronas e verificar os três desfechos possíveis (sucesso, poltrona indisponível, serviço de estoque inacessível) usando um serviço de estoque simulado.

**Acceptance Scenarios**:

1. **Given** uma pessoa autenticada e uma sessão futura válida, **When** ela solicita a reserva de poltronas disponíveis, **Then** o sistema confirma a reserva informando o identificador da reserva e o instante em que ela expira.
2. **Given** uma pessoa autenticada, **When** ela solicita poltronas das quais ao menos uma já está ocupada ou reservada, **Then** o sistema informa que a solicitação não pôde ser atendida por indisponibilidade, sem reservar nenhuma das poltronas pedidas.
3. **Given** uma pessoa sem credenciais ou com credenciais inválidas, expiradas ou de emissor não reconhecido, **When** solicita uma reserva, **Then** o sistema recusa a solicitação por falta de autenticação válida e não contata o serviço de estoque.
4. **Given** o serviço de estoque está indisponível ou não responde dentro do tempo máximo tolerado, **When** a pessoa solicita uma reserva, **Then** o sistema informa, em até pouco mais desse tempo máximo, que o serviço está temporariamente indisponível, com mensagem padronizada.
5. **Given** uma pessoa autenticada, **When** ela solicita a reserva para uma sessão inexistente, **Then** o sistema informa que a sessão não foi encontrada e não contata o serviço de estoque.
6. **Given** uma pessoa autenticada, **When** ela envia uma lista de poltronas vazia ou com identificadores repetidos, **Then** o sistema recusa a solicitação explicando o problema.

---

### User Story 4 - Consultar cinemas e salas (Priority: P2)

A pessoa quer conhecer os cinemas disponíveis (nome, cidade, estado, endereço) e as salas de cada cinema (número, tipo de tela, capacidade), para escolher onde assistir antes mesmo de olhar horários.

**Why this priority**: Enriquece a navegação e apoia a escolha por localização, mas a jornada principal de compra funciona sem essa consulta dedicada, já que a grade de sessões já expõe cinema e sala.

**Independent Test**: Cadastrar cinemas com salas e verificar que a consulta pública retorna os cinemas com seus dados de localização e as salas associadas a cada um.

**Acceptance Scenarios**:

1. **Given** existem cinemas cadastrados, **When** a pessoa consulta a lista de cinemas, **Then** o sistema retorna nome, cidade, estado e endereço de cada cinema.
2. **Given** um cinema com salas cadastradas, **When** a pessoa consulta as salas desse cinema, **Then** o sistema retorna número, tipo de tela e capacidade total de cada sala.
3. **Given** um identificador de cinema inexistente, **When** a pessoa consulta suas salas, **Then** o sistema informa que o cinema não foi encontrado.

---

### Edge Cases

- Uma sessão referencia um filme ou uma sala que foi removido ou alterado: a grade não pode exibir registros incompletos nem falhar por completo.
- O relógio da sessão cruza o momento presente durante a navegação: sessões que já começaram não devem ser oferecidas para reserva.
- O serviço de estoque responde depois do tempo máximo tolerado, quando o cliente já recebeu a resposta de indisponibilidade: a reserva eventualmente criada do outro lado precisa expirar sozinha, sem deixar o cliente com cobrança ou bloqueio invisível.
- O serviço de estoque responde com sucesso, mas com dados incompletos (sem identificador de reserva ou sem prazo de expiração): a resposta ao cliente não pode alegar sucesso sem esses dados.
- Credencial válida em assinatura, porém sem o identificador da pessoa usuária: a reserva não pode prosseguir sem saber a quem atribuí-la.
- Duas solicitações simultâneas para as mesmas poltronas: apenas uma pode ser confirmada; a outra deve receber a resposta de indisponibilidade.
- O serviço de estoque falha de forma sustentada e o sistema entra em modo de recusa rápida: as pessoas usuárias precisam receber a mesma resposta padronizada de indisponibilidade temporária, sem diferença perceptível entre recusa rápida e timeout real.
- O sistema está em modo de recusa rápida no momento em que o serviço de estoque volta a funcionar: a retomada precisa acontecer sozinha, dentro do intervalo configurado.
- Consulta a qualquer coleção sem nenhum filtro, com o catálogo cheio: a resposta precisa continuar utilizável e previsível, limitada pelo tamanho de página.
- Posição de paginação além do fim dos resultados, ou tamanho de página negativo, zero ou acima do teto: o sistema precisa responder de forma previsível em vez de falhar.
- Registros são inseridos ou removidos entre a leitura de duas páginas consecutivas: a paginação não pode duplicar nem esconder registros de forma imprevisível para quem está navegando.
- Material de apoio ausente (filme sem sinopse ou sem imagem): a listagem deve continuar funcionando com os campos opcionais vazios.

## Requirements *(mandatory)*

### Functional Requirements

**Listagens (regra transversal)**

- **FR-001**: Toda consulta que retorna uma coleção — filmes, cinemas, salas e grade de sessões — MUST ser paginada, sem exceção: nenhuma consulta retorna o conjunto completo de registros em uma única resposta.
- **FR-002**: O sistema MUST aceitar em cada consulta paginada o tamanho de página e a posição desejados, aplicar um tamanho de página padrão quando não informados e recusar tamanhos acima de um teto máximo configurável.
- **FR-003**: Cada resposta paginada MUST informar, além dos itens da página, o total de registros que atendem aos filtros e se existem mais resultados a obter, de modo que o cliente possa navegar entre páginas sem adivinhar.
- **FR-004**: O sistema MUST aplicar uma ordenação determinística e estável em toda consulta paginada, de forma que páginas consecutivas não repitam nem omitam registros na ausência de alterações nos dados.
- **FR-005**: O sistema MUST retornar uma página vazia, e não erro, quando a posição solicitada estiver além do último resultado disponível.

**Catálogo de filmes**

- **FR-006**: O sistema MUST expor publicamente, sem exigir autenticação, a listagem de filmes com título, sinopse, duração em minutos, classificação etária, gênero, imagem e situação.
- **FR-007**: O sistema MUST permitir filtrar a listagem de filmes pela situação do filme, aceitando ao menos "em cartaz" e "em breve".
- **FR-008**: O sistema MUST omitir da listagem pública, quando nenhum filtro de situação for informado, os filmes fora de cartaz.
- **FR-009**: O sistema MUST recusar filtros com valores não reconhecidos, informando os valores aceitos.
- **FR-010**: Toda consulta MUST refletir o estado atual dos dados no momento da leitura, sem defasagem tolerada.

**Cinemas e salas**

- **FR-011**: O sistema MUST expor publicamente a listagem de cinemas com nome, cidade, estado e endereço.
- **FR-012**: O sistema MUST expor publicamente as salas de um cinema com número, tipo de tela e capacidade total.
- **FR-013**: O sistema MUST informar de forma distinguível quando um cinema solicitado não existe.

**Grade de sessões**

- **FR-014**: O sistema MUST expor publicamente a grade de sessões, retornando para cada sessão o filme, o nome do cinema, o número da sala, o tipo de tela, a data e hora de início, o idioma e o preço base.
- **FR-015**: O sistema MUST permitir filtrar a grade de sessões por filme, por cinema e por data, isoladamente ou em combinação.
- **FR-016**: O sistema MUST retornar somente sessões agendadas ou em andamento na consulta pública, excluindo sessões canceladas e finalizadas.
- **FR-017**: O sistema MUST retornar uma página vazia, e não erro, quando nenhum registro atender aos filtros informados.
- **FR-018**: O sistema MUST validar o formato da data informada como filtro e recusar valores fora do formato esperado.

**Reserva de poltronas**

- **FR-019**: O sistema MUST exigir credenciais válidas para solicitar a reserva de poltronas.
- **FR-020**: O sistema MUST validar as credenciais de forma autônoma, sem depender de uma consulta ao emissor a cada requisição, aceitando apenas credenciais assinadas pelo emissor confiável e dentro da validade.
- **FR-021**: O sistema MUST extrair a identidade da pessoa usuária a partir da credencial apresentada e usá-la como titular da reserva, recusando a solicitação quando essa identidade não estiver presente.
- **FR-022**: O sistema MUST validar, antes de solicitar o bloqueio ao serviço de estoque, que a sessão informada existe e ainda aceita reservas.
- **FR-023**: O sistema MUST validar a lista de poltronas solicitadas, recusando listas vazias e listas com identificadores duplicados.
- **FR-024**: O sistema MUST delegar o bloqueio efetivo das poltronas ao serviço de estoque, encaminhando a sessão, as poltronas solicitadas e a identidade da pessoa usuária.
- **FR-025**: O sistema MUST retornar, em caso de sucesso, a confirmação da reserva com seu identificador e o instante de expiração devolvidos pelo serviço de estoque.
- **FR-026**: O sistema MUST tratar a reserva como tudo-ou-nada: se qualquer poltrona solicitada estiver indisponível, nenhuma é reservada e a resposta indica indisponibilidade de forma distinguível dos demais erros.
- **FR-027**: O sistema MUST limitar a espera pela resposta do serviço de estoque a no máximo 2 segundos.
- **FR-028**: O sistema MUST responder com indicação padronizada de indisponibilidade temporária quando o serviço de estoque falhar ou exceder o tempo máximo de espera, sem expor detalhes internos da falha.
- **FR-029**: O sistema MUST NOT repetir automaticamente uma solicitação de bloqueio já enviada ao serviço de estoque: cada solicitação da pessoa usuária resulta em no máximo uma tentativa de bloqueio, evitando bloqueios órfãos quando a falha ocorre após o pedido ter sido recebido.
- **FR-030**: O sistema MUST interromper temporariamente as chamadas ao serviço de estoque após uma sequência configurável de falhas consecutivas, respondendo de imediato com indisponibilidade temporária durante esse período, e MUST retomar as chamadas automaticamente após um intervalo configurável, sem intervenção manual.
- **FR-031**: O sistema MUST NOT reter estado de reserva próprio: a titularidade sobre poltronas, bloqueios e expiração pertence ao serviço de estoque.

**Configuração e operação**

- **FR-032**: Todas as configurações de ambiente — acesso ao armazenamento de dados, emissor de credenciais e endereço do serviço de estoque — MUST ser fornecidas externamente ao artefato, sem valores sensíveis embutidos no código.
- **FR-033**: O sistema MUST registrar cada solicitação de reserva e seu desfecho de forma auditável, sem gravar credenciais em texto claro.
- **FR-034**: O sistema MUST emitir, para cada requisição atendida, um registro estruturado e legível por máquina contendo um identificador de correlação, a operação, o desfecho e a duração, sem gravar credenciais ou dados sensíveis em texto claro.
- **FR-035**: O sistema MUST expor métricas de volume, latência e desfecho por operação, incluindo métricas específicas das chamadas ao serviço de estoque (sucesso, indisponibilidade, tempo excedido e recusa rápida), consumíveis por um coletor externo.
- **FR-036**: O sistema MUST participar do rastreamento distribuído, aceitando o contexto de rastreamento recebido do cliente (ou iniciando um quando ausente) e propagando-o na chamada ao serviço de estoque, de modo que uma solicitação de reserva possa ser seguida de ponta a ponta entre os dois serviços.
- **FR-037**: O sistema MUST expor um indicador de saúde que permita a um orquestrador saber se a instância está apta a receber tráfego.
- **FR-038**: O sistema MUST responder a erros de forma padronizada e legível por máquina, distinguindo entrada inválida, recurso não encontrado, falta de autenticação, conflito de disponibilidade e indisponibilidade temporária.

### Key Entities *(include if feature involves data)*

- **Filme**: obra em exibição ou prevista. Título, sinopse, duração em minutos, classificação etária, gênero, imagem de divulgação e situação (em cartaz, em breve, fora de cartaz). É referenciado por sessões.
- **Cinema**: complexo físico onde ocorrem as exibições. Nome, cidade, estado e endereço. Agrupa salas.
- **Sala**: ambiente de exibição dentro de um cinema. Número, tipo de tela (2D, 3D, IMAX, VIP) e capacidade total. Pertence a exatamente um cinema.
- **Sessão**: exibição de um filme em uma sala em um instante específico. Data e hora de início, idioma (dublado, legendado), preço base e situação (agendada, em andamento, finalizada, cancelada). Relaciona um filme e uma sala; é o alvo das solicitações de reserva.
- **Solicitação de reserva**: intenção de compra de uma pessoa identificada sobre um conjunto de poltronas de uma sessão. Não é persistida por este serviço — é encaminhada ao serviço de estoque, que devolve identificador de reserva e prazo de expiração.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Uma pessoa consegue partir da descoberta de um filme até a confirmação da reserva de poltronas em no máximo 4 interações com o sistema.
- **SC-002**: Uma alteração feita pelo processo administrativo em filmes, cinemas, salas ou sessões é visível na consulta seguinte das pessoas usuárias.
- **SC-003**: 95% das consultas ao catálogo (uma página de filmes, cinemas, salas ou da grade de sessões) retornam resultado em menos de 1 segundo com o catálogo carregado com 500 filmes, 50 cinemas, 300 salas e 5.000 sessões, e em menos de 2 segundos com dez vezes esse volume. O tempo de resposta cresce com o total de registros que atendem aos filtros — a margem acima é o limite de volume em que o desenho de paginação atual permanece válido.
- **SC-004**: 99% das solicitações de reserva recebem uma resposta conclusiva — confirmação, indisponibilidade ou indisponibilidade temporária — em menos de 2,5 segundos, mesmo quando o serviço de estoque está lento ou fora do ar.
- **SC-005**: Nenhuma solicitação de reserva é aceita sem credencial válida: 100% das tentativas sem credencial, com credencial expirada ou com credencial de emissor não reconhecido são recusadas.
- **SC-006**: Em disputas simultâneas pelas mesmas poltronas, no máximo uma solicitação é confirmada, verificado por teste de concorrência com 50 solicitações paralelas sobre o mesmo conjunto de poltronas.
- **SC-007**: Durante uma falha sustentada do serviço de estoque, as solicitações de reserva recebem resposta de indisponibilidade temporária em menos de 200 ms (recusa rápida, sem esperar o tempo máximo), e o serviço volta a encaminhar solicitações automaticamente em até 1 minuto após o restabelecimento do estoque.
- **SC-008**: 100% das consultas de coleção respeitam o teto de página: nenhuma resposta contém mais itens que o máximo configurado, mesmo com filtro amplo ou tamanho de página abusivo solicitado.
- **SC-009**: 100% das respostas de erro seguem o formato padronizado, permitindo ao aplicativo cliente distinguir cada categoria de falha sem interpretar texto livre.
- **SC-010**: Uma nova instância do serviço sobe e passa a atender tráfego apenas com configuração externa, sem qualquer alteração no artefato entregue.
- **SC-011**: 100% das solicitações de reserva podem ser reconstituídas de ponta a ponta a partir dos sinais emitidos — do recebimento no catálogo até o desfecho no serviço de estoque — usando um único identificador de correlação.
- **SC-012**: Uma indisponibilidade completa do serviço de estoque não impede a navegação: as consultas de catálogo e de sessões continuam respondendo normalmente.

## Assumptions

- O cadastro e a manutenção de filmes, cinemas, salas e sessões (criação, edição, exclusão) são realizados por um processo administrativo fora do escopo desta feature; este serviço apenas consulta esses dados e os expõe.
- O serviço de estoque já existe (ou será entregue em paralelo) e é a única fonte de verdade sobre disponibilidade de poltronas, expiração e titularidade de reservas.
- As credenciais das pessoas usuárias são emitidas por um provedor de identidade externo já operante; este serviço apenas valida assinatura e validade, sem gerenciar cadastro, login ou renovação.
- Existe (ou existirá) uma infraestrutura externa de coleta de métricas e de rastreamento distribuído no ambiente de execução; este serviço apenas emite e propaga os sinais, não os armazena nem os visualiza.
- O pagamento, a emissão do ingresso e o cancelamento da reserva estão fora do escopo: a feature termina na confirmação do bloqueio temporário das poltronas.
- O mapa de poltronas de cada sessão (quais poltronas existem e quais estão livres) é responsabilidade do serviço de estoque; este serviço não valida se um identificador de poltrona existe na sala.
- A limitação de volume de requisições por origem (rate limiting / throttling) dos endpoints públicos é responsabilidade da camada de borda (gateway/ingress) que fica à frente do serviço, e não deste serviço; o teto de página aplicado a todas as consultas de coleção é a única contenção de volume aplicada aqui.
- Os dados são consultados por clientes finais (aplicativo/front-end); não há requisito de exportação em massa ou integração B2B nesta feature.
- Toda consulta de coleção é paginada desde a primeira versão, inclusive as de baixa cardinalidade (cinemas, salas), para que o contrato exposto ao cliente seja uniforme e não precise de mudança incompatível quando o volume crescer.
- Idioma dos dados de catálogo é português do Brasil; internacionalização está fora do escopo.
