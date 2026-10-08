# Feature Specification: Roteamento do Catálogo via chi

**Feature Branch**: _não aplicável — o trabalho é commitado direto na `master` (regra do mantenedor)_

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Modifica "Catalogo" de modo que ele passe a usa chi como mecanismo de rotas."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comportamento HTTP idêntico após a troca (Priority: P1)

O mantenedor troca o mecanismo de roteamento HTTP do Servico-Catalogo, hoje o roteador da biblioteca padrão, pelo chi, mantendo todo o comportamento observável: as mesmas rotas, métodos, parâmetros de caminho, proteção por autenticação, respostas de erro, documentação (`/openapi.yaml`, `/docs`) e verificação de saúde (`/health`). Quem consome o serviço (clientes REST, o frontend, a documentação interativa) não percebe diferença.

**Why this priority**: sem equivalência funcional a troca não tem valor; é o núcleo da entrega e o que torna a migração segura.

**Independent Test**: executar a suíte existente de testes do serviço sem enfraquecer asserções e percorrer o fluxo de quickstart (listar, cadastrar sala e sessão, reservar) verificando os mesmos resultados de antes.

**Acceptance Scenarios**:

1. **Given** cada rota hoje registrada, **When** um cliente a chama com o mesmo método e caminho, **Then** o mesmo tratamento é executado e a resposta é igual à anterior (status, corpo e cabeçalhos relevantes).
2. **Given** uma rota com parâmetro de caminho (por exemplo `/api/v1/filmes/{id}`), **When** é chamada com um identificador, **Then** o tratamento recebe exatamente o valor informado.
3. **Given** uma rota protegida, **When** é chamada sem credencial válida, **Then** a resposta de não autenticado é a mesma de antes; e as rotas públicas continuam acessíveis sem credencial.
4. **Given** os controles transversais de telemetria, recuperação de falhas e registro de requisições, **When** qualquer requisição é atendida (inclusive as que resultam em erro ou em falha interna), **Then** continuam sendo aplicados, na mesma ordem relativa de antes.

---

### User Story 2 - Comportamento para rotas inexistentes e métodos não permitidos (Priority: P2)

Uma requisição para um caminho que não existe, ou para um caminho existente com método não suportado, continua recebendo uma resposta de erro previsível. Se o novo mecanismo mudar o formato ou os cabeçalhos dessas respostas, a mudança é conhecida e decidida, não acidental.

**Why this priority**: é onde roteadores diferentes mais divergem sem que ninguém perceba; vale proteger depois do caminho feliz.

**Independent Test**: chamar um caminho inexistente e um método não permitido em uma rota existente, e comparar status, cabeçalho `Allow` e corpo com o comportamento anterior.

**Acceptance Scenarios**:

1. **Given** um caminho que nenhuma rota atende, **When** é requisitado, **Then** a resposta é de "não encontrado", igual à anterior.
2. **Given** um caminho existente chamado com método não registrado para ele, **When** é requisitado, **Then** a resposta é de "método não permitido", com a lista de métodos aceitos, igual à anterior.
3. **Given** `/docs` e `/docs/`, **When** cada um é requisitado, **Then** ambos continuam servindo a documentação interativa.

---

### User Story 3 - Contrato e tabela de rotas continuam verificados e a fronteira do núcleo se mantém (Priority: P3)

Quem mantém o código continua tendo uma única tabela declarativa de rotas, que serve de base ao teste de paridade com o contrato OpenAPI. O domínio e os casos de uso seguem sem conhecer o chi, e a verificação automática de dependências do núcleo continua passando.

**Why this priority**: preserva a garantia de que rota registrada e rota documentada não divergem, e a regra de dependências apontando para dentro da constituição do serviço.

**Independent Test**: rodar o teste de paridade entre contrato e rotas e a verificação de dependências (`make lint`); verificar que nenhum pacote do núcleo importa o chi.

**Acceptance Scenarios**:

1. **Given** a troca concluída, **When** o teste de paridade roda, **Then** ele passa sem alteração do contrato versionado.
2. **Given** a troca concluída, **When** a verificação de dependências roda, **Then** ela passa, e o chi passa a constar entre as bibliotecas proibidas ao núcleo.
3. **Given** o núcleo do serviço, **When** os testes unitários rodam, **Then** continuam executando sem banco, sem rede e sem servidor.

---

### Edge Cases

- Requisição com caminho inexistente, com método não permitido ou com barra final divergente: resposta definida e coberta por teste (ver User Story 2).
- Falha interna (pânico) dentro de um tratamento: continua sendo recuperada e convertida na resposta de erro padrão, sem derrubar o processo.
- Parâmetros de caminho com caracteres especiais ou codificados: o tratamento recebe o mesmo valor que recebia antes.
- Rotas que compartilham prefixo (por exemplo `/api/v1/sessoes` e `/api/v1/sessoes/{id}/reservar`): cada uma continua resolvendo para o seu tratamento.
- Identificador de requisição e contexto de telemetria continuam disponíveis aos tratamentos e aos registros de log.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O roteamento HTTP do serviço MUST passar a ser feito pelo chi, substituindo o roteador da biblioteca padrão usado hoje.
- **FR-002**: O conjunto de rotas (método e caminho), o contrato REST e as respostas de sucesso e de erro MUST permanecer idênticos aos anteriores à troca.
- **FR-003**: Os parâmetros de caminho MUST chegar aos tratamentos com os mesmos valores de antes.
- **FR-004**: As rotas marcadas como protegidas MUST continuar exigindo autenticação, com a mesma resposta de recusa; as demais MUST continuar públicas.
- **FR-005**: Telemetria, recuperação de falhas e registro de requisições MUST continuar envolvendo todas as requisições, incluindo as que não casam com nenhuma rota, na mesma ordem relativa de antes.
- **FR-006**: As respostas para caminho inexistente e para método não permitido MUST permanecer as de hoje; qualquer diferença inevitável imposta pelo chi MUST ser levada ao mantenedor antes de ser aceita.
- **FR-007**: `/openapi.yaml`, `/docs`, `/docs/` e `/health` MUST continuar disponíveis com o mesmo comportamento.
- **FR-008**: A tabela declarativa de rotas, com seus indicadores de documentada e protegida, MUST continuar existindo como única fonte das rotas registradas, e o teste de paridade com o contrato OpenAPI MUST continuar passando sem alteração do contrato.
- **FR-009**: O núcleo (domínio e casos de uso) MUST NOT importar o chi; a verificação automática de dependências do núcleo MUST continuar passando e MUST passar a proibir também a biblioteca nova.
- **FR-010**: A configuração do serviço (variáveis de ambiente, portas) MUST permanecer a mesma para quem opera o ambiente.
- **FR-011**: O encerramento ordenado do serviço e a verificação de saúde MUST continuar funcionando de forma equivalente.

### Key Entities

- **Rota**: associação entre método, caminho e tratamento, com indicadores de se é documentada no contrato e se exige autenticação.
- **Tabela de rotas**: conjunto declarativo de todas as rotas do serviço, usado tanto para registrá-las quanto para verificar a paridade com o contrato.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos testes automatizados existentes do serviço (unitários e de integração) passam sem enfraquecer nenhuma asserção.
- **SC-002**: 100% das rotas hoje registradas respondem com o mesmo status e o mesmo corpo de antes, verificado por teste automatizado que percorre a tabela de rotas.
- **SC-003**: O fluxo ponta a ponta do quickstart (listar, cadastrar sala e sessão, reservar) produz resultados idênticos aos da versão anterior.
- **SC-004**: Nenhum pacote do núcleo depende da nova biblioteca (verificado automaticamente pela verificação de dependências da esteira).
- **SC-005**: Nenhuma alteração é necessária nos contratos versionados, no frontend nem em outros serviços para que continuem funcionando com o catálogo.
- **SC-006**: A latência típica de atendimento das rotas não piora de forma perceptível em relação à versão anterior (mesma ordem de grandeza em ambiente local).

## Assumptions

- A escolha do chi é uma decisão do mantenedor, não está em discussão; o escopo é o serviço `catalogo` apenas (estoque, pagamento e notificacao não mudam).
- Os contratos versionados (OpenAPI e proto) e as mensagens AMQP não mudam, portanto não há nova versão de contrato.
- Persistência, mensageria, segurança (JWT) e a chamada gRPC ao estoque não são afetadas.
- Se o chi impuser diferença inevitável de comportamento (por exemplo no corpo ou nos cabeçalhos de 404/405), o padrão é preservar o comportamento atual com um tratamento explícito no adaptador; só se aceita a diferença com decisão do mantenedor.
- Divergências entre o código atual e as specs anteriores são tratadas conforme o princípio de que o código é a fonte da verdade; qualquer divergência encontrada será levada ao mantenedor, não corrigida unilateralmente.
- O trabalho é commitado direto na `master`, sem branch de feature.
