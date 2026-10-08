# Feature Specification: Roteamento do Pagamento via chi

**Feature Branch**: _não aplicável — o trabalho é commitado direto na `master` (regra do mantenedor)_

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Modifique "pagamento" de modo que ele passe a usar chi como mecanismo de rota"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comportamento HTTP idêntico após a troca (Priority: P1)

O mantenedor troca o mecanismo de roteamento HTTP do Servico-Pagamento, hoje o roteador da biblioteca padrão, pelo chi, mantendo todo o comportamento observável: as mesmas rotas, métodos, parâmetro de caminho (`reserva_id`), autenticação, respostas de erro, documentação (`/openapi.yaml`, `/docs`) e verificações de saúde (`/api/v1/health/live`, `/api/v1/health/ready`). Quem consome o serviço (o frontend, clientes REST, a documentação interativa) não percebe diferença.

**Why this priority**: sem equivalência funcional a troca não tem valor; é o núcleo da entrega.

**Independent Test**: executar a suíte existente do serviço sem enfraquecer asserções e percorrer o quickstart (consultar pagamento, escolher forma de pagamento) verificando os mesmos resultados de antes.

**Acceptance Scenarios**:

1. **Given** cada rota hoje registrada, **When** um cliente a chama com o mesmo método e caminho, **Then** o mesmo tratamento é executado e a resposta é igual à anterior (status, corpo e cabeçalhos relevantes).
2. **Given** as rotas com `reserva_id` no caminho, **When** chamadas com um identificador, **Then** o tratamento recebe exatamente o valor informado, e identificador que não é UUID continua recusado com o mesmo erro.
3. **Given** as rotas de consulta e de escolha de pagamento, **When** chamadas sem credencial válida, **Then** a resposta de não autenticado é a mesma de antes; documentação e saúde continuam públicas.

---

### User Story 2 - Rotas inexistentes e métodos não permitidos (Priority: P2)

Uma requisição para caminho inexistente, ou para caminho existente com método não suportado, continua recebendo resposta de erro previsível. Se o novo mecanismo mudar formato ou cabeçalhos dessas respostas, a mudança é conhecida e decidida, não acidental.

**Why this priority**: é onde roteadores diferentes mais divergem sem que ninguém perceba.

**Independent Test**: chamar um caminho inexistente e um método não permitido em rota existente, comparando status, cabeçalho `Allow` e corpo com o comportamento anterior.

**Acceptance Scenarios**:

1. **Given** um caminho que nenhuma rota atende, **When** é requisitado, **Then** a resposta é de "não encontrado", igual à anterior.
2. **Given** um caminho existente chamado com método não registrado para ele, **When** é requisitado, **Then** a resposta é de "método não permitido", com a lista de métodos aceitos, igual à anterior.
3. **Given** `/docs` e `/docs/`, **When** cada um é requisitado, **Then** ambos continuam servindo a documentação interativa.

---

### User Story 3 - Contrato continua verificado e a fronteira do núcleo se mantém (Priority: P3)

O teste que compara a API ao contrato OpenAPI versionado continua passando sem alterar o contrato. O domínio e os casos de uso seguem sem conhecer o chi, e a verificação automática de dependências do núcleo (`make lint`) passa a proibir também a nova biblioteca.

**Why this priority**: preserva a garantia de que a API não diverge do contrato e a regra de dependências apontando para dentro.

**Independent Test**: rodar os testes de contrato e `make lint`; verificar que nenhum pacote do núcleo importa o chi.

**Acceptance Scenarios**:

1. **Given** a troca concluída, **When** os testes de contrato rodam, **Then** passam sem alteração do contrato versionado.
2. **Given** a troca concluída, **When** a verificação de dependências roda, **Then** ela passa, e o chi consta entre as bibliotecas proibidas ao núcleo.
3. **Given** o núcleo do serviço, **When** os testes unitários rodam, **Then** executam sem banco, sem rede e sem servidor.

---

### Edge Cases

- Caminho inexistente, método não permitido ou barra final divergente: resposta definida e coberta por teste (ver User Story 2).
- `reserva_id` com caracteres especiais ou codificados: o tratamento recebe o mesmo valor que recebia antes.
- Rotas que compartilham prefixo (`/api/v1/pagamentos/reserva/{reserva_id}` com GET e POST): cada método resolve para o seu tratamento.
- Corpo ausente ou inválido na escolha de forma: mesma resposta de erro de antes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O roteamento HTTP do serviço MUST passar a ser feito pelo chi, substituindo o roteador da biblioteca padrão usado hoje.
- **FR-002**: O conjunto de rotas (método e caminho), o contrato REST e as respostas de sucesso e de erro MUST permanecer idênticos aos anteriores à troca.
- **FR-003**: O parâmetro de caminho `reserva_id` MUST chegar aos tratamentos com o mesmo valor de antes.
- **FR-004**: A exigência de autenticação nas rotas de pagamento, com a mesma resposta de recusa, e a natureza pública das rotas de documentação e saúde MUST permanecer.
- **FR-005**: As respostas para caminho inexistente e para método não permitido MUST permanecer as de hoje; qualquer diferença inevitável imposta pelo chi MUST ser levada ao mantenedor antes de ser aceita.
- **FR-006**: `/openapi.yaml`, `/docs`, `/docs/`, `/api/v1/health/live` e `/api/v1/health/ready` MUST continuar disponíveis com o mesmo comportamento.
- **FR-007**: Os testes que verificam a API contra o contrato OpenAPI MUST continuar passando sem alteração do contrato.
- **FR-008**: O núcleo (domínio e casos de uso) MUST NOT importar o chi; a verificação automática de dependências MUST continuar passando e MUST passar a proibir também a biblioteca nova.
- **FR-009**: A configuração do serviço (variáveis de ambiente, portas), o encerramento ordenado e a montagem do servidor MUST permanecer equivalentes para quem opera o ambiente.

### Key Entities

- **Rota**: associação entre método, caminho e tratamento.
- **Conjunto de rotas**: definição única de todas as rotas do serviço, entregue ao servidor HTTP.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos testes automatizados existentes do serviço passam sem enfraquecer nenhuma asserção.
- **SC-002**: 100% das rotas hoje registradas respondem com o mesmo status e o mesmo corpo de antes, verificado por teste automatizado.
- **SC-003**: O fluxo do quickstart (consultar e escolher forma de pagamento) produz resultados idênticos aos da versão anterior.
- **SC-004**: Nenhum pacote do núcleo depende da nova biblioteca (verificado pela verificação de dependências da esteira).
- **SC-005**: Nenhuma alteração é necessária nos contratos versionados, no frontend nem em outros serviços para que continuem funcionando com o pagamento.
- **SC-006**: A latência típica de atendimento das rotas não piora de forma perceptível (mesma ordem de grandeza em ambiente local).

## Assumptions

- A escolha do chi é decisão do mantenedor, não está em discussão; o escopo é o serviço `pagamento` apenas (catalogo e estoque têm feature própria; notificacao não muda).
- Contratos versionados e mensagens AMQP não mudam.
- Persistência, mensageria e autenticação JWT não são afetadas.
- Hoje o serviço não possui camada transversal (telemetria, recuperação de falhas) no roteador; a troca não a introduz — seria escopo novo.
- Se o chi impuser diferença inevitável de comportamento (por exemplo no corpo ou nos cabeçalhos de 404/405), o padrão é preservar o comportamento atual com tratamento explícito no adaptador; só se aceita a diferença com decisão do mantenedor.
- Divergências entre código e specs anteriores seguem o princípio de que o código é a fonte da verdade e são levadas ao mantenedor.
- O trabalho é commitado direto na `master`, sem branch de feature.
