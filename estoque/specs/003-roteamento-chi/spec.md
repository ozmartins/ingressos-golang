# Feature Specification: Roteamento do Estoque via chi

**Feature Branch**: _não aplicável — o trabalho é commitado direto na `master` (regra do mantenedor)_

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Modifique "estoque" de modo que ele passe a usar chi como mecanismo de rota"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comportamento HTTP idêntico após a troca (Priority: P1)

O mantenedor troca o mecanismo de roteamento HTTP do Servico-Estoque, hoje o roteador da biblioteca padrão, pelo chi, mantendo todo o comportamento observável nas duas superfícies HTTP do serviço: a API REST (bloqueio de poltronas, consulta do mapa, `/openapi.yaml`, `/docs`) e a porta de administração (`/health/live`, `/health/ready`). Quem consome o serviço (clientes REST, a documentação interativa, a orquestração de saúde do ambiente) não percebe diferença.

**Why this priority**: sem equivalência funcional a troca não tem valor; é o núcleo da entrega e o que torna a migração segura.

**Independent Test**: executar a suíte existente de testes do serviço sem enfraquecer asserções e percorrer o fluxo do quickstart (consultar mapa, bloquear poltronas) verificando os mesmos resultados de antes.

**Acceptance Scenarios**:

1. **Given** cada rota hoje registrada, **When** um cliente a chama com o mesmo método e caminho, **Then** o mesmo tratamento é executado e a resposta é igual à anterior (status, corpo e cabeçalhos relevantes).
2. **Given** uma rota com parâmetro de caminho (`/api/v1/sessoes/{sessao_id}/bloqueios` e `/api/v1/sessoes/{sessao_id}/poltronas`), **When** é chamada com um identificador, **Then** o tratamento recebe exatamente o valor informado.
3. **Given** as duas rotas da API REST (bloqueio e mapa), **When** são chamadas sem credencial válida, **Then** a resposta de não autenticado é a mesma de antes; e a documentação e a saúde continuam acessíveis sem credencial.
4. **Given** a porta de administração, **When** `/health/live` e `/health/ready` são consultados, **Then** retornam os mesmos status e corpos de antes, inclusive o código de indisponibilidade quando uma dependência essencial falha.

---

### User Story 2 - Comportamento para rotas inexistentes e métodos não permitidos (Priority: P2)

Uma requisição para um caminho que não existe, ou para um caminho existente com método não suportado, continua recebendo uma resposta de erro previsível. Se o novo mecanismo mudar o formato ou os cabeçalhos dessas respostas, a mudança é conhecida e decidida, não acidental.

**Why this priority**: é onde roteadores diferentes mais divergem sem que ninguém perceba; vale proteger depois do caminho feliz.

**Independent Test**: chamar um caminho inexistente e um método não permitido em uma rota existente, nas duas portas, e comparar status, cabeçalho `Allow` e corpo com o comportamento anterior.

**Acceptance Scenarios**:

1. **Given** um caminho que nenhuma rota atende, **When** é requisitado, **Then** a resposta é de "não encontrado", igual à anterior.
2. **Given** um caminho existente chamado com método não registrado para ele, **When** é requisitado, **Then** a resposta é de "método não permitido", com a lista de métodos aceitos, igual à anterior.
3. **Given** `/docs` e `/docs/`, **When** cada um é requisitado, **Then** ambos continuam servindo a documentação interativa.

---

### User Story 3 - Contrato continua verificado e a fronteira do núcleo se mantém (Priority: P3)

Quem mantém o código continua tendo a cópia embutida do contrato OpenAPI sincronizada com o contrato versionado, sem que a troca de roteador exija mexer nele. O domínio e os casos de uso seguem sem conhecer o chi, e a verificação automática de dependências do núcleo continua passando.

**Why this priority**: preserva a garantia de que rota registrada e rota documentada não divergem, e a regra de dependências apontando para dentro da constituição do serviço.

**Independent Test**: rodar os testes do pacote do contrato embutido, o teste de arquitetura e `make lint`; verificar que nenhum pacote do núcleo importa o chi.

**Acceptance Scenarios**:

1. **Given** a troca concluída, **When** o teste de sincronia do contrato embutido roda, **Then** passa sem alteração do contrato versionado.
2. **Given** a troca concluída, **When** o teste de arquitetura e a verificação de dependências rodam, **Then** passam, e o chi passa a constar entre as bibliotecas proibidas ao núcleo.
3. **Given** o núcleo do serviço, **When** os testes unitários rodam, **Then** continuam executando sem banco, sem rede e sem servidor.

---

### Edge Cases

- Caminho inexistente, método não permitido ou barra final divergente (`/docs` vs `/docs/`): resposta definida e coberta por teste.
- Identificador de sessão com caracteres especiais ou codificados na URL: o tratamento recebe o mesmo valor que recebia antes.
- Rotas que compartilham prefixo (`/api/v1/sessoes/{sessao_id}/bloqueios` e `.../poltronas`): cada uma resolve para o seu tratamento e método.
- Requisição `HEAD` ou `OPTIONS` em rotas registradas só para `GET`/`POST`: comportamento igual ao de antes (hoje `HEAD` é atendido onde `GET` é).
- Caminho com subcaminho sob `/docs/` (por exemplo `/docs/x`): continua servindo a documentação, como hoje.
- Falha na checagem de dependência durante `/health/ready`: o serviço continua respondendo indisponível/degradado exatamente como hoje.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O roteamento HTTP do serviço, na API REST e na porta de administração, MUST passar a ser feito pelo chi, substituindo o roteador da biblioteca padrão usado hoje.
- **FR-002**: O conjunto de rotas (método e caminho), o contrato REST e as respostas de sucesso e de erro MUST permanecer idênticos aos anteriores à troca.
- **FR-003**: Os parâmetros de caminho MUST chegar aos tratamentos com os mesmos valores de antes.
- **FR-004**: As rotas de bloqueio e de mapa MUST continuar exigindo autenticação, com a mesma resposta de recusa; documentação e saúde MUST continuar sem credencial, como hoje.
- **FR-005**: As respostas para caminho inexistente e para método não permitido MUST permanecer as de hoje; qualquer diferença inevitável imposta pelo chi MUST ser levada ao mantenedor antes de ser aceita.
- **FR-006**: `/openapi.yaml`, `/docs`, `/docs/`, `/health/live` e `/health/ready` MUST continuar disponíveis com o mesmo comportamento.
- **FR-007**: O contrato OpenAPI versionado e sua cópia embutida MUST permanecer inalterados, e o teste de sincronia entre eles MUST continuar passando.
- **FR-008**: O núcleo (domínio e casos de uso) MUST NOT importar o chi; o teste de arquitetura e a verificação de dependências MUST continuar passando e MUST passar a proibir também a biblioteca nova.
- **FR-009**: A configuração do serviço (variáveis de ambiente, portas HTTP, de administração e gRPC) MUST permanecer a mesma para quem opera o ambiente.
- **FR-010**: O encerramento ordenado do serviço MUST continuar funcionando de forma equivalente nas duas portas HTTP.

### Key Entities

- **Rota**: associação entre método, caminho e tratamento. A exigência de autenticação é verificada dentro do tratamento de cada rota REST (não por camada de roteamento).
- **Superfície HTTP**: cada uma das duas portas servidas pelo serviço — API REST e administração (saúde).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos testes automatizados existentes do serviço passam sem enfraquecer nenhuma asserção.
- **SC-002**: 100% das rotas hoje registradas respondem com o mesmo status e o mesmo corpo de antes, verificado por teste automatizado que percorre a lista de rotas.
- **SC-003**: O fluxo do quickstart (consultar mapa, bloquear poltronas) produz resultados idênticos aos da versão anterior.
- **SC-004**: Nenhum pacote do núcleo depende da nova biblioteca (verificado automaticamente).
- **SC-005**: Nenhuma alteração é necessária nos contratos versionados, no catálogo, no frontend nem em outros serviços para que continuem funcionando com o estoque.
- **SC-006**: A latência típica de atendimento das rotas não piora de forma perceptível em relação à versão anterior (mesma ordem de grandeza em ambiente local).

## Assumptions

- Aceito pelo mantenedor (2026-10-07): caminhos não canônicos (`//`, `..`) deixam de receber redirecionamento 307 e passam a receber 404; nenhum cliente do repositório os gera.
- A escolha do chi é uma decisão do mantenedor, não está em discussão; o escopo é o serviço `estoque` apenas (catalogo, pagamento e notificacao não mudam).
- A superfície gRPC, a mensageria AMQP, a persistência e a segurança (JWT, mTLS) não são afetadas; os contratos versionados não mudam.
- Se o chi impuser diferença inevitável de comportamento (por exemplo no corpo ou nos cabeçalhos de 404/405), o padrão é preservar o comportamento atual com um tratamento explícito no adaptador; só se aceita a diferença com decisão do mantenedor.
- Divergências entre o código atual e as specs anteriores são tratadas conforme o princípio de que o código é a fonte da verdade; qualquer divergência encontrada será levada ao mantenedor, não corrigida unilateralmente.
- O trabalho é commitado direto na `master`, sem branch de feature.
