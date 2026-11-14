# Feature Specification: Comandos de Escrita do Terminal Guiados por Campos

**Feature Branch**: `004-campos-comandos-escrita` (sem branch própria: o trabalho é commitado direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Modifique os comandos de scritp. Em vez de pedir um JSON, eles devem pedir os campos e a aplicação monta e envia o JSON pra API de catálogo"

## Contexto

Hoje os comandos de escrita do terminal (`filmes|cinemas|salas|sessoes criar` e `atualizar`) exigem que a pessoa digite o corpo da requisição como um JSON completo na mesma linha. Isso obriga a conhecer a estrutura exata, as aspas e os nomes de campo. Esta funcionalidade troca esse formato: o terminal **pede cada campo** à pessoa e a própria aplicação **monta o JSON** e o envia à API de catálogo. Os comandos de leitura, `remover`, `health`, `login` e `logout` não mudam.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Criar um registro respondendo aos campos (Priority: P1)

Como pessoa desenvolvedora autenticada, quero executar `filmes criar` (ou `cinemas|salas|sessoes criar`) sem informar nenhum JSON e ser guiada pelo terminal, campo a campo, para cadastrar o registro sem decorar a estrutura de dados.

**Why this priority**: É o ponto central do pedido e elimina a maior fricção atual: montar o JSON à mão.

**Independent Test**: Autenticado, executar `cinemas criar`, responder aos campos pedidos e ver o cinema criado (com identificador) na saída; conferir com `cinemas obter <id>` que os valores batem com o que foi digitado.

**Acceptance Scenarios**:

1. **Given** uma sessão autenticada, **When** o visitante executa `filmes criar`, **Then** o terminal pede, um de cada vez, os campos do filme, indicando o nome, se é obrigatório ou opcional e, quando houver, as opções válidas ou o valor padrão.
2. **Given** que todos os campos obrigatórios foram respondidos, **When** o último campo é confirmado, **Then** a aplicação monta o corpo da requisição, envia à API de catálogo e mostra o registro criado e o caminho informado pelo serviço, como já ocorre hoje.
3. **Given** um campo opcional, **When** a pessoa deixa a resposta em branco, **Then** o campo é omitido do corpo e o serviço aplica seu valor padrão.
4. **Given** um campo numérico (por exemplo `duracao_minutos`) ou lógico (por exemplo `ativo`), **When** a pessoa responde, **Then** o valor é enviado com o tipo correto, sem que ela precise digitar aspas ou sintaxe de JSON.
5. **Given** um campo obrigatório, **When** a resposta fica em branco, **Then** o terminal avisa que o campo é obrigatório e pergunta de novo, sem enviar nada.
6. **Given** um campo com opções fixas (por exemplo `status` do filme ou `tipo_tela` da sala), **When** a resposta não é uma das opções válidas, **Then** o terminal mostra as opções aceitas e pergunta de novo.

---

### User Story 2 - Atualizar um registro respondendo aos campos (Priority: P2)

Como pessoa desenvolvedora autenticada, quero executar `<controller> atualizar <id>` e ser guiada pelos campos, para alterar um registro sem montar o JSON completo.

**Why this priority**: Mesma mecânica da criação, aplicada à substituição de registros; depende da base entregue em US1.

**Independent Test**: Autenticado, executar `salas atualizar <id>`, responder aos campos e ver o registro atualizado na saída.

**Acceptance Scenarios**:

1. **Given** um id existente, **When** o visitante executa `filmes atualizar <id>`, **Then** o terminal pede os mesmos campos da criação e, ao concluir, envia a atualização e mostra o registro atualizado.
2. **Given** a atualização de uma sala, **When** o terminal pede os campos, **Then** pede apenas os campos que a API aceita nessa operação (número, tipo de tela e demais permitidos), sem exigir a planta de fileiras nem o cinema.
3. **Given** um id sem o argumento correspondente (`filmes atualizar` sem id), **When** executado, **Then** a saída mostra a mensagem de uso com a sintaxe esperada, como hoje.

---

### User Story 3 - Informar a planta de fileiras da sala (Priority: P3)

Como pessoa desenvolvedora autenticada, quero cadastrar uma sala informando as fileiras (letra, quantidade de assentos e tipo) de forma guiada, já que a planta é uma lista de itens, e não um valor simples.

**Why this priority**: `salas criar` só é completo com a planta; sem isso, US1 não cobre as salas. É a única parte estruturalmente mais complexa.

**Independent Test**: Autenticado, executar `salas criar`, informar cinema, número, tipo de tela e duas fileiras, e ver a sala criada com a capacidade total correta.

**Acceptance Scenarios**:

1. **Given** `salas criar` em andamento, **When** o terminal chega às fileiras, **Then** pede os dados de uma fileira por vez (letra, quantidade de assentos, tipo) e, após cada uma, pergunta se há outra fileira a adicionar.
2. **Given** que nenhuma fileira foi informada, **When** a pessoa tenta concluir, **Then** o terminal avisa que ao menos uma fileira é obrigatória.
3. **Given** uma letra de fileira inválida (fora de A–Z, vazia ou com mais de cinco letras), **When** informada, **Then** o terminal avisa e pergunta de novo.

---

### User Story 4 - Cancelar e corrigir durante o preenchimento (Priority: P4)

Como visitante, quero poder desistir de um preenchimento em andamento e ver o que foi montado antes do envio, para não enviar dados errados por engano.

**Why this priority**: Conforto e segurança de uso; o fluxo funciona sem isso, mas fica frágil.

**Independent Test**: Iniciar `filmes criar`, responder alguns campos, cancelar e confirmar que nada foi enviado à API e que o terminal voltou ao prompt normal.

**Acceptance Scenarios**:

1. **Given** um preenchimento em andamento, **When** a pessoa digita o comando de cancelamento indicado na instrução do terminal, **Then** o preenchimento é abandonado, nenhuma requisição é enviada e o terminal volta ao prompt normal.
2. **Given** que todos os campos foram respondidos, **When** o terminal vai enviar, **Then** exibe um resumo dos valores informados e pede confirmação antes de enviar.
3. **Given** a confirmação negada, **When** a pessoa responde que não, **Then** nada é enviado e o preenchimento é encerrado.

---

### Edge Cases

- Sem autenticação: o terminal informa que é preciso autenticar e indica `login` **antes** de pedir qualquer campo, para não desperdiçar o preenchimento.
- Erros de validação devolvidos pelo serviço (campos inválidos, conflito de número de sala etc.) continuam sendo exibidos por campo, e a pessoa pode repetir o comando.
- Token expirado durante o preenchimento: a orientação para autenticar de novo aparece no envio, como hoje.
- Valor numérico inválido (texto onde se espera número, zero ou negativo onde o mínimo é 1): o terminal avisa e pergunta de novo.
- Histórico de comandos: respostas aos campos, não devem poluir o histórico de comandos recuperável com a seta para cima.
- Entrada de um comando de escrita que ainda traz JSON na linha (formato antigo): o terminal não envia nada e orienta a usar o novo fluxo guiado.
- Resposta com o serviço fora do ar: mensagem clara de indisponibilidade, sem travar o terminal.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Os comandos `criar` de filmes, cinemas, salas e sessões MUST funcionar sem receber JSON; ao serem executados sem argumentos além do nome da ação, o terminal MUST conduzir a pessoa pelo preenchimento dos campos.
- **FR-002**: Os comandos `atualizar` desses quatro controllers MUST receber apenas o identificador na linha e conduzir o preenchimento dos campos aceitos pela API na atualização.
- **FR-003**: Para cada campo, o terminal MUST informar o nome, a obrigatoriedade e, quando houver, as opções válidas ou o valor padrão.
- **FR-004**: A aplicação MUST montar o corpo da requisição a partir das respostas, convertendo cada valor para o tipo esperado (texto, número inteiro, lógico, lista) e omitindo os campos opcionais deixados em branco.
- **FR-005**: A aplicação MUST enviar o corpo montado à API de catálogo pelos mesmos endpoints e com a mesma autenticação já usados hoje, e exibir o resultado do mesmo modo.
- **FR-006**: O terminal MUST validar cada resposta localmente (obrigatoriedade, tipo, opções fixas, mínimos e tamanhos máximos documentados no contrato da API) e perguntar novamente quando inválida, sem enviar requisição.
- **FR-007**: Para salas, o terminal MUST permitir informar uma ou mais fileiras (letra, assentos, tipo) em sequência, exigindo ao menos uma na criação.
- **FR-008**: O terminal MUST verificar a autenticação antes de iniciar o preenchimento de um comando de escrita e orientar o uso de `login` quando ausente.
- **FR-009**: A pessoa MUST poder cancelar o preenchimento a qualquer momento, sem que nenhuma requisição seja enviada.
- **FR-010**: Antes de enviar, o terminal MUST mostrar um resumo dos valores informados e pedir confirmação.
- **FR-011**: Comandos de escrita que recebam JSON na linha MUST ser recusados com orientação para o novo fluxo, sem envio de requisição.
- **FR-012**: As respostas dadas durante o preenchimento MUST NOT ser gravadas no histórico de comandos.
- **FR-013**: O `help` e as mensagens de uso MUST refletir a nova sintaxe (`criar` sem JSON; `atualizar <id>` sem JSON) e deixar de exibir exemplos com JSON.
- **FR-014**: Os comandos de leitura, `remover`, `health`, `login` e `logout` MUST manter o comportamento atual.

### Key Entities

- **Definição de campo**: descreve um dado pedido à pessoa: nome, rótulo, tipo (texto, inteiro, lógico, opção fixa, lista de fileiras), obrigatoriedade, valor padrão, restrições (mínimo, tamanho máximo, padrão de formato).
- **Formulário de comando**: conjunto ordenado de definições de campo associado a uma combinação controller + ação (`filmes criar`, `salas atualizar` etc.).
- **Preenchimento em andamento**: estado do diálogo no terminal: formulário ativo, respostas já aceitas e campo atual.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Uma pessoa consegue cadastrar um filme, um cinema, uma sala e uma sessão pelo terminal sem digitar nenhuma chave de objeto, aspas ou chaves, em 100% dos casos de cadastro válidos.
- **SC-002**: Nenhuma operação de escrita exige conhecer previamente os nomes dos campos: 100% deles são informados pelo próprio terminal durante o preenchimento.
- **SC-003**: Respostas inválidas (campo obrigatório vazio, número inválido, opção inexistente) são rejeitadas no próprio terminal em 100% dos casos, sem requisição à API.
- **SC-004**: Cancelar um preenchimento nunca resulta em requisição enviada nem em registro criado ou alterado.
- **SC-005**: O fluxo completo de cadastrar um cinema leva menos de 1 minuto para uma pessoa que nunca usou o comando.

## Assumptions

- "Pedir os campos" é um diálogo interativo no próprio terminal: uma pergunta por campo, respondida com Enter. Não é uma linha de argumentos `chave=valor`.
- O escopo cobre os oito comandos que hoje recebem JSON: `criar` e `atualizar` de filmes, cinemas, salas e sessões. Outros comandos permanecem como estão.
- Na atualização, a API substitui o registro; por isso os mesmos campos obrigatórios da criação (exceto os que o contrato exclui, como planta e cinema da sala) são pedidos de novo. Pré-preencher com os valores atuais do registro não faz parte do escopo.
- O formato antigo com JSON na linha é removido (não mantido como atalho), conforme o pedido de substituir o JSON pelos campos.
- Datas e horas informadas sem fuso (por exemplo `2026-12-01 19:30`) são interpretadas no fuso horário local do navegador e enviadas à API com o fuso explícito.
- Os campos, seus tipos, obrigatoriedades, opções e limites seguem o contrato publicado pela API de catálogo (filmes, cinemas, salas, sessões); o formulário reflete esse contrato.
- A autenticação (`login` por credenciais de máquina) e a exibição de erros e resultados permanecem como na funcionalidade anterior (003).
